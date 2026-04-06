package service

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"dreamcreator/internal/application/assistant/dto"
	settingsservice "dreamcreator/internal/application/settings/service"
	workspaceservice "dreamcreator/internal/application/workspace/service"
	assistantassets "dreamcreator/internal/assets/assistant"
	"dreamcreator/internal/domain/assistant"
)

const (
	defaultAssistantName     = "DreamCreator"
	defaultAssistantCreature = "an ai agent for content creator"
)

var defaultAssistantEmojis = []string{
	"🤖",
	"✨",
	"🌟",
	"🧠",
	"🦋",
	"🪄",
	"🌙",
	"☀️",
	"🪐",
	"🎧",
	"🎨",
	"📚",
}

var assistantRoleOptions = []string{
	"General assistant",
}

var assistantVibeOptions = []string{
	"Formal",
	"Casual",
	"Snarky",
	"Warm",
	"Sharp",
	"Calm",
	"Chaotic",
}

const (
	assistantDefaultRole = "General assistant"
	assistantDefaultVibe = "Warm"
)

func pickRandomEmoji() string {
	if len(defaultAssistantEmojis) == 0 {
		return "🙂"
	}
	index := int(time.Now().UnixNano() % int64(len(defaultAssistantEmojis)))
	return defaultAssistantEmojis[index]
}

type AssistantService struct {
	repo            assistant.Repository
	workspaces      *workspaceservice.WorkspaceService
	settings        *settingsservice.SettingsService
	memorySummary   AssistantMemorySummaryReader
	httpClient      *http.Client
	cacheMu         sync.Mutex
	cache           autoLocaleCache
	resolveLocation func(ctx context.Context) string
	now             func() time.Time
	newID           func() string
}

type AssistantMemorySummaryReader interface {
	GetAssistantMemorySummary(ctx context.Context, assistantID string) (dto.AssistantMemorySummary, error)
}

func NewAssistantService(
	repo assistant.Repository,
	workspaceService *workspaceservice.WorkspaceService,
	settings *settingsservice.SettingsService,
) *AssistantService {
	return &AssistantService{
		repo:       repo,
		workspaces: workspaceService,
		settings:   settings,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		now:        time.Now,
		newID:      uuid.NewString,
	}
}

func (service *AssistantService) SetMemorySummaryReader(reader AssistantMemorySummaryReader) {
	if service == nil {
		return
	}
	service.memorySummary = reader
}

func (service *AssistantService) EnsureDefaults(ctx context.Context) error {
	if err := ensureBuiltinAssets(); err != nil {
		return err
	}
	items, err := service.repo.List(ctx, true)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		defaultAvatar := assistant.AssistantAvatar{
			Avatar3D: assistant.AssistantAvatarAssetRef{
				Source:  assistant.AvatarSourceBuiltin,
				AssetID: assistantassets.BuiltinAvatarAssetID,
			},
			Motion: assistant.AssistantAvatarAssetRef{
				Source:  assistant.AvatarSourceBuiltin,
				AssetID: assistantassets.BuiltinMotionAssetID,
			},
		}
		defaultAvatar = applyBuiltinAvatarDefaults(defaultAvatar)
		model := defaultAssistantModel()
		tools := defaultAssistantTools()
		call := defaultAssistantCall()
		memory := defaultAssistantMemory()
		now := service.now()
		builtin := true
		deletable := false
		enabled := true
		isDefault := true
		item, err := assistant.NewAssistant(assistant.AssistantParams{
			ID:        service.newID(),
			Builtin:   &builtin,
			Deletable: &deletable,
			Identity: assistant.AssistantIdentity{
				Name:     defaultAssistantName,
				Creature: defaultAssistantCreature,
				Emoji:    pickRandomEmoji(),
			},
			Avatar:    defaultAvatar,
			User:      defaultAssistantUser(),
			Model:     model,
			Tools:     tools,
			Skills:    defaultAssistantSkills(),
			Call:      call,
			Memory:    memory,
			Enabled:   &enabled,
			IsDefault: &isDefault,
			CreatedAt: &now,
			UpdatedAt: &now,
		})
		if err != nil {
			return err
		}
		if err := service.repo.Save(ctx, item); err != nil {
			return err
		}
		if service.workspaces != nil {
			if err := service.workspaces.EnsureAssistantWorkspace(ctx, item.ID); err != nil {
				return err
			}
		}
		return nil
	}
	hasDefault := false
	hasEnabled := false
	defaultIndex := -1
	for index, item := range items {
		changed := false
		if strings.TrimSpace(item.Identity.Emoji) == "" {
			item.Identity.Emoji = pickRandomEmoji()
			changed = true
		}
		if item.Builtin && item.Deletable {
			item.Deletable = false
			changed = true
		}
		updatedAvatar, avatarChanged, err := ensureBuiltinAvatarRefs(item.Avatar)
		if err != nil {
			return err
		}
		if avatarChanged {
			item.Avatar = updatedAvatar
			changed = true
		}
		if !item.Model.Embedding.Inherit &&
			strings.TrimSpace(item.Model.Embedding.Primary) == "" &&
			len(item.Model.Embedding.Fallbacks) == 0 {
			item.Model.Embedding.Inherit = true
			changed = true
		}
		if !item.Model.Image.Inherit &&
			strings.TrimSpace(item.Model.Image.Primary) == "" &&
			len(item.Model.Image.Fallbacks) == 0 {
			item.Model.Image.Inherit = true
			changed = true
		}
		if changed {
			item.UpdatedAt = service.now()
			if err := service.repo.Save(ctx, item); err != nil {
				return err
			}
		}
		items[index] = item
		if item.IsDefault {
			hasDefault = true
			defaultIndex = index
		}
		if item.Enabled {
			hasEnabled = true
		}
	}
	if !hasEnabled && len(items) > 0 {
		targetIndex := defaultIndex
		if targetIndex < 0 {
			targetIndex = 0
		}
		target := items[targetIndex]
		target.Enabled = true
		target.UpdatedAt = service.now()
		if err := service.repo.Save(ctx, target); err != nil {
			return err
		}
		items[targetIndex] = target
	}
	if hasDefault {
		if service.workspaces != nil {
			if err := service.ensureAssistantWorkspaces(ctx, items); err != nil {
				return err
			}
		}
		return nil
	}
	if err := service.repo.SetDefault(ctx, items[0].ID); err != nil {
		return err
	}
	if service.workspaces != nil {
		if err := service.ensureAssistantWorkspaces(ctx, items); err != nil {
			return err
		}
	}
	return nil
}

func (service *AssistantService) ListAssistants(ctx context.Context, includeDisabled bool) ([]dto.Assistant, error) {
	items, err := service.repo.List(ctx, includeDisabled)
	if err != nil {
		return nil, err
	}
	result := make([]dto.Assistant, 0, len(items))
	for _, item := range items {
		updated, changed, err := ensureBuiltinAvatarRefs(item.Avatar)
		if err != nil {
			return nil, err
		}
		if changed {
			item.Avatar = updated
			item.UpdatedAt = service.now()
			_ = service.repo.Save(ctx, item)
		}
		result = append(result, toDTO(item))
	}
	return result, nil
}

func (service *AssistantService) GetAssistant(ctx context.Context, id string) (dto.Assistant, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return dto.Assistant{}, assistant.ErrAssistantNotFound
	}
	item, err := service.repo.Get(ctx, id)
	if err != nil {
		return dto.Assistant{}, err
	}
	updated, changed, err := ensureBuiltinAvatarRefs(item.Avatar)
	if err != nil {
		return dto.Assistant{}, err
	}
	if changed {
		item.Avatar = updated
		item.UpdatedAt = service.now()
		_ = service.repo.Save(ctx, item)
	}
	updatedUser, userChanged := service.populateAutoUserLocale(ctx, item.User)
	if userChanged {
		item.User = updatedUser
		item.UpdatedAt = service.now()
		_ = service.repo.Save(ctx, item)
	}
	return toDTO(item), nil
}

func (service *AssistantService) CreateAssistant(ctx context.Context, request dto.CreateAssistantRequest) (dto.Assistant, error) {
	identity := request.Identity
	identity.Emoji = strings.TrimSpace(identity.Emoji)
	if identity.Emoji == "" {
		identity.Emoji = pickRandomEmoji()
	}
	avatar := request.Avatar
	user := request.User
	if updatedUser, changed := service.populateAutoUserLocale(ctx, user); changed {
		user = updatedUser
	}
	model := request.Model
	if reflect.DeepEqual(model, assistant.AssistantModel{}) {
		model = defaultAssistantModel()
	}
	tools := request.Tools
	if reflect.DeepEqual(tools, assistant.AssistantTools{}) {
		tools = defaultAssistantTools()
	}
	skills := request.Skills
	if reflect.DeepEqual(skills, assistant.AssistantSkills{}) {
		skills = defaultAssistantSkills()
	}
	call := request.Call
	if reflect.DeepEqual(call, assistant.AssistantCall{}) {
		call = defaultAssistantCall()
	}
	memory := request.Memory
	if reflect.DeepEqual(memory, assistant.AssistantMemory{}) {
		memory = defaultAssistantMemory()
	}
	updatedAvatar, _, err := ensureBuiltinAvatarRefs(avatar)
	if err != nil {
		return dto.Assistant{}, err
	}
	avatar = updatedAvatar
	builtin := false
	deletable := true
	now := service.now()
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	item, err := assistant.NewAssistant(assistant.AssistantParams{
		ID:        service.newID(),
		Builtin:   &builtin,
		Deletable: &deletable,
		Identity:  identity,
		Avatar:    avatar,
		User:      user,
		Model:     model,
		Tools:     tools,
		Skills:    skills,
		Call:      call,
		Memory:    memory,
		Enabled:   &enabled,
		IsDefault: &request.IsDefault,
		CreatedAt: &now,
		UpdatedAt: &now,
	})
	if err != nil {
		return dto.Assistant{}, err
	}
	if err := service.repo.Save(ctx, item); err != nil {
		return dto.Assistant{}, err
	}
	if service.workspaces != nil {
		if err := service.workspaces.EnsureAssistantWorkspace(ctx, item.ID); err != nil {
			return dto.Assistant{}, err
		}
	}
	if request.IsDefault {
		if err := service.repo.SetDefault(ctx, item.ID); err != nil {
			return dto.Assistant{}, err
		}
		item.IsDefault = true
	}
	return toDTO(item), nil
}

func (service *AssistantService) UpdateAssistant(ctx context.Context, request dto.UpdateAssistantRequest) (dto.Assistant, error) {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return dto.Assistant{}, assistant.ErrAssistantNotFound
	}
	current, err := service.repo.Get(ctx, id)
	if err != nil {
		return dto.Assistant{}, err
	}
	identity := current.Identity
	if request.Identity != nil {
		identity = *request.Identity
	}
	identity.Emoji = strings.TrimSpace(identity.Emoji)
	if identity.Emoji == "" {
		identity.Emoji = pickRandomEmoji()
	}
	avatar := current.Avatar
	if request.Avatar != nil {
		avatar = *request.Avatar
	}
	user := current.User
	if request.User != nil {
		user = *request.User
	}
	if updatedUser, changed := service.populateAutoUserLocale(ctx, user); changed {
		user = updatedUser
	}
	model := current.Model
	if request.Model != nil {
		model = *request.Model
	}
	tools := current.Tools
	if request.Tools != nil {
		tools = *request.Tools
	}
	skills := current.Skills
	if request.Skills != nil {
		skills = *request.Skills
	}
	call := current.Call
	if request.Call != nil {
		call = *request.Call
	}
	memory := current.Memory
	if request.Memory != nil {
		memory = *request.Memory
	}
	updatedAvatar, _, err := ensureBuiltinAvatarRefs(avatar)
	if err != nil {
		return dto.Assistant{}, err
	}
	avatar = updatedAvatar
	enabled := current.Enabled
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	isDefault := current.IsDefault
	if request.IsDefault != nil {
		isDefault = *request.IsDefault
	}
	now := service.now()
	item, err := assistant.NewAssistant(assistant.AssistantParams{
		ID:        current.ID,
		Builtin:   &current.Builtin,
		Deletable: &current.Deletable,
		Identity:  identity,
		Avatar:    avatar,
		User:      user,
		Model:     model,
		Tools:     tools,
		Skills:    skills,
		Call:      call,
		Memory:    memory,
		Enabled:   &enabled,
		IsDefault: &isDefault,
		CreatedAt: &current.CreatedAt,
		UpdatedAt: &now,
	})
	if err != nil {
		return dto.Assistant{}, err
	}
	if err := service.repo.Save(ctx, item); err != nil {
		return dto.Assistant{}, err
	}
	if request.IsDefault != nil && *request.IsDefault {
		if err := service.repo.SetDefault(ctx, item.ID); err != nil {
			return dto.Assistant{}, err
		}
		item.IsDefault = true
	}
	return toDTO(item), nil
}

func (service *AssistantService) DeleteAssistant(ctx context.Context, request dto.DeleteAssistantRequest) error {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return assistant.ErrInvalidAssistantID
	}
	current, err := service.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if current.Builtin || !current.Deletable {
		return assistant.ErrAssistantDeletionNotAllowed
	}
	if err := service.repo.Delete(ctx, id); err != nil {
		return err
	}
	if current.IsDefault {
		items, err := service.repo.List(ctx, true)
		if err != nil {
			return err
		}
		if len(items) > 0 {
			return service.repo.SetDefault(ctx, items[0].ID)
		}
	}
	return nil
}

func (service *AssistantService) SetDefaultAssistant(ctx context.Context, request dto.SetDefaultAssistantRequest) error {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return assistant.ErrInvalidAssistantID
	}
	return service.repo.SetDefault(ctx, id)
}

func (service *AssistantService) GetAssistantMemorySummary(ctx context.Context, assistantID string) (dto.AssistantMemorySummary, error) {
	if service == nil {
		return dto.AssistantMemorySummary{}, nil
	}
	if service.memorySummary == nil {
		return dto.AssistantMemorySummary{}, nil
	}
	return service.memorySummary.GetAssistantMemorySummary(ctx, assistantID)
}

func (service *AssistantService) GetAssistantProfileOptions(_ context.Context) (dto.AssistantProfileOptions, error) {
	return dto.AssistantProfileOptions{
		Roles:       append([]string(nil), assistantRoleOptions...),
		DefaultRole: assistantDefaultRole,
		Vibes:       append([]string(nil), assistantVibeOptions...),
		DefaultVibe: assistantDefaultVibe,
	}, nil
}

func toDTO(item assistant.Assistant) dto.Assistant {
	return dto.Assistant{
		ID:        item.ID,
		Builtin:   item.Builtin,
		Deletable: item.Deletable,
		Identity:  item.Identity,
		Avatar:    item.Avatar,
		User:      item.User,
		Model:     item.Model,
		Tools:     item.Tools,
		Skills:    item.Skills,
		Call:      item.Call,
		Memory:    item.Memory,
		Readiness: buildAssistantReadiness(item),
		Enabled:   item.Enabled,
		IsDefault: item.IsDefault,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
		UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
	}
}

func buildAssistantReadiness(item assistant.Assistant) dto.AssistantReadiness {
	missing := make([]string, 0, 1)

	if strings.TrimSpace(item.Model.Agent.Primary) == "" {
		missing = append(missing, "model.agent.primary")
	}

	return dto.AssistantReadiness{
		Ready:   len(missing) == 0,
		Missing: missing,
	}
}

func (service *AssistantService) ensureAssistantWorkspaces(ctx context.Context, items []assistant.Assistant) error {
	for _, item := range items {
		if err := service.workspaces.EnsureAssistantWorkspace(ctx, item.ID); err != nil {
			return err
		}
	}
	return nil
}

func defaultAssistantUser() assistant.AssistantUser {
	return assistant.AssistantUser{
		Language: assistant.UserLocale{Mode: "auto"},
		Timezone: assistant.UserLocale{Mode: "auto"},
		Location: assistant.UserLocale{Mode: "auto"},
	}
}

func defaultAssistantModel() assistant.AssistantModel {
	return assistant.AssistantModel{
		Agent: assistant.ModelConfig{
			Stream:      true,
			Temperature: 0.7,
			MaxTokens:   2048,
		},
		Image: assistant.ModelConfig{
			Inherit:     true,
			Stream:      true,
			Temperature: 0.7,
			MaxTokens:   2048,
		},
		Embedding: assistant.ModelConfig{
			Inherit:     true,
			Stream:      true,
			Temperature: 0.7,
			MaxTokens:   2048,
		},
	}
}

func defaultAssistantTools() assistant.AssistantTools {
	return assistant.AssistantTools{}
}

func defaultAssistantSkills() assistant.AssistantSkills {
	return assistant.AssistantSkills{
		Mode:              assistant.SkillsModeOn,
		MaxSkillsInPrompt: assistant.DefaultAssistantMaxSkillsInPrompt,
		MaxPromptChars:    assistant.DefaultAssistantSkillsPromptChars,
	}
}

func defaultAssistantCall() assistant.AssistantCall {
	return assistant.AssistantCall{
		Tools:  assistant.CallToolsConfig{Mode: assistant.CallModeAuto},
		Skills: assistant.CallSkillsConfig{Mode: assistant.CallModeAuto},
	}
}

func defaultAssistantMemory() assistant.AssistantMemory {
	return assistant.AssistantMemory{Enabled: true}
}

func applyBuiltinAvatarDefaults(avatar assistant.AssistantAvatar) assistant.AssistantAvatar {
	updated, _, err := ensureBuiltinAvatarRefs(avatar)
	if err != nil {
		return avatar
	}
	return updated
}

package assistant

import "strings"

const (
	PromptModeFull    = "full"
	PromptModeMinimal = "minimal"
	PromptModeNone    = "none"

	CallModeAuto   = "auto"
	CallModeCustom = "custom"

	SkillsModeOn  = "on"
	SkillsModeOff = "off"

	AvatarSourceBuiltin = "builtin"
	AvatarSourceUser    = "user"
)

const (
	DefaultAssistantMaxSkillsInPrompt = 150
	DefaultAssistantSkillsPromptChars = 30_000
)

type AssistantIdentity struct {
	Name     string        `json:"name,omitempty"`
	Creature string        `json:"creature,omitempty"`
	Emoji    string        `json:"emoji,omitempty"`
	Role     string        `json:"role,omitempty"`
	Soul     AssistantSoul `json:"soul,omitempty"`
}

type AssistantSoul struct {
	CoreTruths []string `json:"coreTruths,omitempty"`
	Boundaries []string `json:"boundaries,omitempty"`
	Rules      []string `json:"rules,omitempty"`
	Vibe       string   `json:"vibe,omitempty"`
	Continuity string   `json:"continuity,omitempty"`
}

type AssistantAvatar struct {
	Avatar3D AssistantAvatarAssetRef `json:"avatar3d,omitempty"`
	Motion   AssistantAvatarAssetRef `json:"motion,omitempty"`
}

type AssistantAvatarAssetRef struct {
	Path        string            `json:"path,omitempty"`
	DisplayName string            `json:"displayName,omitempty"`
	Source      string            `json:"source,omitempty"`
	AssetID     string            `json:"assetId,omitempty"`
	Meta        map[string]string `json:"meta,omitempty"`
}

type AssistantUser struct {
	Name             string           `json:"name,omitempty"`
	PreferredAddress string           `json:"preferredAddress,omitempty"`
	Pronouns         string           `json:"pronouns,omitempty"`
	Notes            string           `json:"notes,omitempty"`
	Language         UserLocale       `json:"language,omitempty"`
	Timezone         UserLocale       `json:"timezone,omitempty"`
	Location         UserLocale       `json:"location,omitempty"`
	Extra            []UserExtraField `json:"extra,omitempty"`
}

type UserLocale struct {
	Mode    string `json:"mode,omitempty"`
	Value   string `json:"value,omitempty"`
	Current string `json:"current,omitempty"`
}

type UserExtraField struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type ModelConfig struct {
	Inherit     bool     `json:"inherit"`
	Primary     string   `json:"primary,omitempty"`
	Fallbacks   []string `json:"fallbacks,omitempty"`
	Stream      bool     `json:"stream"`
	Temperature float32  `json:"temperature"`
	MaxTokens   int      `json:"maxTokens"`
}

type AssistantModel struct {
	Agent     ModelConfig `json:"agent"`
	Image     ModelConfig `json:"image"`
	Embedding ModelConfig `json:"embedding"`
}

type AssistantTools struct {
	Items []AssistantToolItem `json:"items,omitempty"`
}

type AssistantToolItem struct {
	ID      string `json:"id,omitempty"`
	Enabled bool   `json:"enabled"`
}

type AssistantSkills struct {
	Mode              string `json:"mode,omitempty"`
	MaxSkillsInPrompt int    `json:"maxSkillsInPrompt,omitempty"`
	MaxPromptChars    int    `json:"maxPromptChars,omitempty"`
}

type AssistantCall struct {
	Tools  CallToolsConfig  `json:"tools"`
	Skills CallSkillsConfig `json:"skills"`
}

type CallToolsConfig struct {
	Mode      string   `json:"mode,omitempty"`
	AllowList []string `json:"allowList,omitempty"`
	DenyList  []string `json:"denyList,omitempty"`
}

type CallSkillsConfig struct {
	Mode      string   `json:"mode,omitempty"`
	AllowList []string `json:"allowList,omitempty"`
}

type AssistantMemory struct {
	Enabled bool `json:"enabled"`
}

func normalizeAssistantIdentity(identity AssistantIdentity) AssistantIdentity {
	identity.Name = strings.TrimSpace(identity.Name)
	identity.Creature = strings.TrimSpace(identity.Creature)
	identity.Emoji = strings.TrimSpace(identity.Emoji)
	identity.Role = strings.TrimSpace(identity.Role)
	identity.Soul = normalizeAssistantSoul(identity.Soul)
	return identity
}

func normalizeAssistantSoul(soul AssistantSoul) AssistantSoul {
	soul.CoreTruths = normalizeStringSlice(soul.CoreTruths)
	soul.Boundaries = normalizeStringSlice(soul.Boundaries)
	soul.Rules = normalizeStringSlice(soul.Rules)
	soul.Vibe = strings.TrimSpace(soul.Vibe)
	soul.Continuity = strings.TrimSpace(soul.Continuity)
	return soul
}

func normalizeAssistantAvatar(avatar AssistantAvatar) AssistantAvatar {
	avatar.Avatar3D = normalizeAssistantAvatarAssetRef(avatar.Avatar3D)
	avatar.Motion = normalizeAssistantAvatarAssetRef(avatar.Motion)
	return avatar
}

func normalizeAssistantAvatarAssetRef(ref AssistantAvatarAssetRef) AssistantAvatarAssetRef {
	ref.Path = strings.TrimSpace(ref.Path)
	ref.DisplayName = strings.TrimSpace(ref.DisplayName)
	ref.Source = strings.TrimSpace(ref.Source)
	ref.AssetID = strings.TrimSpace(ref.AssetID)
	if ref.Path == "" {
		ref.DisplayName = ""
		ref.Meta = nil
		return ref
	}
	if len(ref.Meta) == 0 {
		ref.Meta = nil
	}
	return ref
}

func normalizeAssistantUser(user AssistantUser) AssistantUser {
	user.Name = strings.TrimSpace(user.Name)
	user.PreferredAddress = strings.TrimSpace(user.PreferredAddress)
	user.Pronouns = strings.TrimSpace(user.Pronouns)
	user.Notes = strings.TrimSpace(user.Notes)
	user.Language = normalizeUserLocale(user.Language)
	user.Timezone = normalizeUserLocale(user.Timezone)
	user.Location = normalizeUserLocale(user.Location)
	user.Extra = normalizeUserExtraFields(user.Extra)
	return user
}

func normalizeUserLocale(locale UserLocale) UserLocale {
	mode := strings.ToLower(strings.TrimSpace(locale.Mode))
	if mode != "manual" {
		mode = "auto"
	}
	locale.Mode = mode
	locale.Value = strings.TrimSpace(locale.Value)
	locale.Current = strings.TrimSpace(locale.Current)
	if locale.Mode == "auto" {
		locale.Value = ""
	}
	return locale
}

func ResolveUserLocaleValue(locale UserLocale) string {
	mode := strings.ToLower(strings.TrimSpace(locale.Mode))
	if mode == "manual" {
		return strings.TrimSpace(locale.Value)
	}
	if value := strings.TrimSpace(locale.Current); value != "" {
		return value
	}
	return ""
}

func normalizeUserExtraFields(extra []UserExtraField) []UserExtraField {
	if len(extra) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(extra))
	result := make([]UserExtraField, 0, len(extra))
	for _, field := range extra {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			continue
		}
		lower := strings.ToLower(key)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		value := strings.TrimSpace(field.Value)
		result = append(result, UserExtraField{Key: key, Value: value})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeAssistantModel(model AssistantModel) AssistantModel {
	model.Agent = normalizeModelConfig(model.Agent)
	model.Image = normalizeModelConfig(model.Image)
	model.Embedding = normalizeModelConfig(model.Embedding)
	return model
}

func normalizeModelConfig(model ModelConfig) ModelConfig {
	model.Primary = strings.TrimSpace(model.Primary)
	model.Fallbacks = normalizeStringSlice(model.Fallbacks)
	return model
}

func normalizeAssistantTools(tools AssistantTools) AssistantTools {
	if len(tools.Items) == 0 {
		return AssistantTools{}
	}
	seen := make(map[string]struct{}, len(tools.Items))
	items := make([]AssistantToolItem, 0, len(tools.Items))
	for _, item := range tools.Items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, AssistantToolItem{ID: id, Enabled: item.Enabled})
	}
	if len(items) == 0 {
		return AssistantTools{}
	}
	return AssistantTools{Items: items}
}

func normalizeAssistantSkills(skills AssistantSkills) AssistantSkills {
	mode := normalizeAssistantSkillsMode(skills.Mode)
	maxSkillsInPrompt := skills.MaxSkillsInPrompt
	if maxSkillsInPrompt <= 0 {
		maxSkillsInPrompt = DefaultAssistantMaxSkillsInPrompt
	}
	maxPromptChars := skills.MaxPromptChars
	if maxPromptChars <= 0 {
		maxPromptChars = DefaultAssistantSkillsPromptChars
	}
	return AssistantSkills{
		Mode:              mode,
		MaxSkillsInPrompt: maxSkillsInPrompt,
		MaxPromptChars:    maxPromptChars,
	}
}

func normalizeAssistantSkillsMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case SkillsModeOff, "none", "disabled":
		return SkillsModeOff
	default:
		return SkillsModeOn
	}
}

func normalizeAssistantCall(call AssistantCall) AssistantCall {
	call.Tools = normalizeCallToolsConfig(call.Tools)
	call.Skills = normalizeCallSkillsConfig(call.Skills)
	return call
}

func normalizeCallToolsConfig(call CallToolsConfig) CallToolsConfig {
	return CallToolsConfig{
		Mode:      normalizeCallMode(call.Mode),
		AllowList: normalizeStringSlice(call.AllowList),
		DenyList:  normalizeStringSlice(call.DenyList),
	}
}

func normalizeCallSkillsConfig(call CallSkillsConfig) CallSkillsConfig {
	return CallSkillsConfig{
		Mode:      normalizeCallMode(call.Mode),
		AllowList: normalizeStringSlice(call.AllowList),
	}
}

func normalizeCallMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case CallModeAuto:
		return CallModeAuto
	case CallModeCustom:
		return CallModeCustom
	case "off", "none", "disabled":
		return "off"
	default:
		return CallModeAuto
	}
}

func normalizeAssistantMemory(memory AssistantMemory) AssistantMemory {
	return memory
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

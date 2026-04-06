package telegram

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	appcommands "dreamcreator/internal/application/commands"
	settingsdto "dreamcreator/internal/application/settings/dto"
	settingsservice "dreamcreator/internal/application/settings/service"
	skillsdto "dreamcreator/internal/application/skills/dto"
	skillsservice "dreamcreator/internal/application/skills/service"
	telegramapi "dreamcreator/internal/infrastructure/telegram"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"go.uber.org/zap"
)

type MenuSyncStatus struct {
	ChannelID     string    `json:"channelId"`
	Ready         bool      `json:"ready"`
	Synced        bool      `json:"synced"`
	Commands      int       `json:"commands"`
	Issues        []string  `json:"issues,omitempty"`
	OverflowCount int       `json:"overflowCount,omitempty"`
	Error         string    `json:"error,omitempty"`
	SyncedAt      time.Time `json:"syncedAt"`
}

type MenuService struct {
	settings   *settingsservice.SettingsService
	skills     *skillsservice.SkillsService
	httpClient *http.Client
	now        func() time.Time
}

func NewMenuService(settings *settingsservice.SettingsService, skills *skillsservice.SkillsService, httpClient *http.Client) *MenuService {
	return &MenuService{
		settings:   settings,
		skills:     skills,
		httpClient: httpClient,
		now:        time.Now,
	}
}

func (service *MenuService) Sync(ctx context.Context) (MenuSyncStatus, error) {
	if service == nil || service.settings == nil {
		return MenuSyncStatus{ChannelID: "telegram"}, fmt.Errorf("settings service unavailable")
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return MenuSyncStatus{ChannelID: "telegram"}, err
	}
	return service.SyncFromSettings(ctx, current)
}

func (service *MenuService) SyncFromSettings(ctx context.Context, settings settingsdto.Settings) (MenuSyncStatus, error) {
	cfg := ResolveMenuConfig(settings)
	status := MenuSyncStatus{
		ChannelID: "telegram",
		Ready:     cfg.Enabled && strings.TrimSpace(cfg.BotToken) != "",
		SyncedAt:  service.now(),
	}
	if !status.Ready {
		if !cfg.Enabled {
			status.Error = "telegram channel disabled"
		} else {
			status.Error = "telegram bot token missing"
		}
		return status, nil
	}

	allNativeCommands := appcommands.ListNativeCommandSpecsForSettings(settings, "telegram")
	nativeCommands := FilterTelegramMenuNativeCommandSpecs(allNativeCommands)
	reserved := make(map[string]struct{}, len(allNativeCommands))
	for _, command := range allNativeCommands {
		name := NormalizeTelegramCommandName(command.Name)
		if name != "" {
			reserved[name] = struct{}{}
		}
	}

	customResolution := ResolveTelegramCustomCommands(ResolveCustomCommandsParams{
		Commands:         cfg.CustomCommands,
		ReservedCommands: reserved,
	})
	issues := make([]string, 0, len(customResolution.Issues))
	for _, issue := range customResolution.Issues {
		issues = append(issues, issue.Message)
	}

	allCommands := make([]MenuCommand, 0)
	if cfg.NativeCommandsEnabled {
		for _, command := range nativeCommands {
			normalized := NormalizeTelegramCommandName(command.Name)
			if normalized == "" || !telegramCommandNamePattern.MatchString(normalized) {
				issues = append(issues, fmt.Sprintf(`Native command "%s" is invalid for Telegram.`, command.Name))
				continue
			}
			description := NormalizeTelegramCommandDescription(command.Description)
			if description == "" {
				issues = append(issues, fmt.Sprintf(`Native command "%s" is missing a description.`, command.Name))
				continue
			}
			if utf8.RuneCountInString(strings.TrimSpace(command.Description)) > utf8.RuneCountInString(description) {
				issues = append(issues, fmt.Sprintf(`Native command "%s" description exceeded 256 characters and was truncated.`, command.Name))
			}
			allCommands = append(allCommands, MenuCommand{
				Command:     normalized,
				Description: description,
			})
		}
	}

	if cfg.NativeCommandsEnabled && cfg.NativeSkillsEnabled {
		pluginSpecs := service.resolveSkillCommandSpecs(ctx, settings)
		existing := make(map[string]struct{}, len(allCommands)+len(customResolution.Commands))
		for _, command := range allCommands {
			existing[strings.ToLower(command.Command)] = struct{}{}
		}
		for _, command := range customResolution.Commands {
			existing[strings.ToLower(command.Command)] = struct{}{}
		}
		pluginResult, _ := BuildPluginTelegramMenuCommands(BuildPluginCommandsParams{
			Specs:            pluginSpecs,
			ExistingCommands: existing,
		})
		if len(pluginResult.Commands) > 0 {
			allCommands = append(allCommands, pluginResult.Commands...)
		}
		if len(pluginResult.Issues) > 0 {
			issues = append(issues, pluginResult.Issues...)
		}
	}

	allCommands = append(allCommands, customResolution.Commands...)
	capped := BuildCappedTelegramMenuCommands(BuildCappedCommandsParams{
		AllCommands: allCommands,
	})
	status.Commands = len(capped.CommandsToRegister)
	status.OverflowCount = capped.OverflowCount
	if capped.OverflowCount > 0 {
		issues = append(issues, fmt.Sprintf("Telegram limits bots to %d commands; registering first %d.", capped.MaxCommands, capped.MaxCommands))
	}
	status.Issues = issues

	client := telegramapi.NewClient(cfg.BotToken, service.httpClient)
	scopeTargets := []telego.BotCommandScope{
		tu.ScopeDefault(),
		tu.ScopeAllPrivateChats(),
		tu.ScopeAllGroupChats(),
	}
	commandsPayload := toAPIMenuCommands(capped.CommandsToRegister)
	var firstDeleteErr error
	for _, scope := range scopeTargets {
		if err := client.DeleteMyCommandsScoped(ctx, scope, ""); err != nil {
			if firstDeleteErr == nil {
				firstDeleteErr = err
			}
			status.Issues = append(status.Issues, fmt.Sprintf("Failed to clear Telegram menu scope %s: %s", scope.ScopeType(), err.Error()))
		}
	}
	if len(capped.CommandsToRegister) == 0 {
		if firstDeleteErr != nil {
			status.Error = firstDeleteErr.Error()
			return status, firstDeleteErr
		}
		status.Synced = true
		return status, nil
	}
	for _, scope := range scopeTargets {
		if err := client.SetMyCommandsScoped(ctx, commandsPayload, scope, ""); err != nil {
			status.Error = err.Error()
			return status, err
		}
	}
	status.Synced = true
	return status, nil
}

func toAPIMenuCommands(commands []MenuCommand) []telegramapi.MenuCommand {
	if len(commands) == 0 {
		return nil
	}
	result := make([]telegramapi.MenuCommand, 0, len(commands))
	for _, command := range commands {
		result = append(result, telegramapi.MenuCommand{
			Command:     command.Command,
			Description: command.Description,
		})
	}
	return result
}

func (service *MenuService) SyncFromSettingsAsync(ctx context.Context, settings settingsdto.Settings) {
	if service == nil {
		return
	}
	go func() {
		if _, err := service.SyncFromSettings(ctx, settings); err != nil {
			zap.L().Warn("telegram menu sync failed", zap.Error(err))
		}
	}()
}

func (service *MenuService) resolveSkillCommandSpecs(ctx context.Context, settings settingsdto.Settings) []PluginCommandSpec {
	if service == nil || service.skills == nil {
		return nil
	}
	providerID := strings.TrimSpace(settings.AgentModelProviderID)
	if providerID == "" {
		return nil
	}
	response, err := service.skills.ResolveSkillPromptItems(ctx, skillsdto.ResolveSkillPromptRequest{ProviderID: providerID})
	if err != nil {
		zap.L().Warn("telegram menu skills resolve failed", zap.Error(err))
		return nil
	}
	items := response.Items
	if len(items) == 0 {
		return nil
	}
	result := make([]PluginCommandSpec, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		result = append(result, PluginCommandSpec{
			Name:        item.Name,
			Description: strings.TrimSpace(item.Description),
		})
	}
	return result
}

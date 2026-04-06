package runtime

import (
	"context"
	"strconv"
	"strings"
	"time"

	"dreamcreator/internal/application/agentruntime"
	"dreamcreator/internal/application/gateway/queue"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	settingsdto "dreamcreator/internal/application/settings/dto"
	skillsdto "dreamcreator/internal/application/skills/dto"
	skillsservice "dreamcreator/internal/application/skills/service"
	tooldto "dreamcreator/internal/application/tools/dto"
	domainassistant "dreamcreator/internal/domain/assistant"
	"dreamcreator/internal/infrastructure/llm"
)

const (
	defaultSubagentMaxSteps       = 8
	defaultSubagentTimeoutSeconds = 300
)

type runFlags struct {
	PersistRun             bool
	PersistMessages        bool
	PersistEvents          bool
	PersistUsage           bool
	PersistContextSnapshot bool
	UseQueue               bool
	IsSubagent             bool
}

func resolveThinkingLevel(config runtimedto.ThinkingConfig, metadata map[string]any) string {
	if level := normalizeThinkingLevel(config.Mode); level != "" {
		return level
	}
	if config.Enabled {
		return "low"
	}
	for _, key := range []string{"thinking", "thinkingLevel", "thinkLevel"} {
		if level := normalizeThinkingLevel(resolveMetadataString(metadata, key)); level != "" {
			return level
		}
	}
	return ""
}

func resolveStructuredOutputConfig(metadata map[string]any) llm.StructuredOutputConfig {
	if metadata == nil {
		return llm.StructuredOutputConfig{}
	}
	value, ok := metadata["structuredOutput"]
	if !ok || value == nil {
		return llm.StructuredOutputConfig{}
	}
	configMap, ok := value.(map[string]any)
	if !ok {
		return llm.StructuredOutputConfig{}
	}
	config := llm.StructuredOutputConfig{
		Mode:   resolveMetadataString(configMap, "mode"),
		Name:   resolveMetadataString(configMap, "name"),
		Strict: true,
	}
	if strict, ok := resolveMetadataBool(configMap, "strict"); ok {
		config.Strict = strict
	}
	if schemaValue, ok := configMap["schema"]; ok {
		if schemaMap, ok := schemaValue.(map[string]any); ok {
			config.Schema = schemaMap
		}
	}
	return config
}

func normalizeThinkingLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "off", "none", "disabled", "disable", "false", "0":
		return "off"
	case "minimal", "min", "think":
		return "minimal"
	case "low", "on", "enabled", "enable", "true", "1":
		return "low"
	case "medium", "med":
		return "medium"
	case "high", "max", "ultra":
		return "high"
	case "xhigh", "x-high", "x_high", "extrahigh", "extra-high", "extra_high", "extra high":
		return "xhigh"
	default:
		return ""
	}
}

func resolveRunFlags(metadata map[string]any) runFlags {
	flags := runFlags{
		PersistRun:             true,
		PersistMessages:        true,
		PersistEvents:          true,
		PersistUsage:           true,
		PersistContextSnapshot: true,
		UseQueue:               true,
		IsSubagent:             false,
	}
	useQueueExplicit := false
	if value, ok := resolveMetadataBool(metadata, "persistRun"); ok {
		flags.PersistRun = value
	}
	if value, ok := resolveMetadataBool(metadata, "persistMessages"); ok {
		flags.PersistMessages = value
	}
	if value, ok := resolveMetadataBool(metadata, "persistEvents"); ok {
		flags.PersistEvents = value
	}
	if value, ok := resolveMetadataBool(metadata, "persistUsage"); ok {
		flags.PersistUsage = value
	}
	if value, ok := resolveMetadataBool(metadata, "persistContextSnapshot"); ok {
		flags.PersistContextSnapshot = value
	}
	if value, ok := resolveMetadataBool(metadata, "useQueue"); ok {
		flags.UseQueue = value
		useQueueExplicit = true
	}
	if value, ok := resolveMetadataBool(metadata, "isSubagent"); ok {
		flags.IsSubagent = value
	} else if value, ok := resolveMetadataBool(metadata, "subagent"); ok {
		flags.IsSubagent = value
	}
	if !flags.PersistRun && !useQueueExplicit {
		flags.UseQueue = false
	}
	return flags
}

func resolveRunKind(request runtimedto.RuntimeRunRequest, flags runFlags) string {
	kind := strings.ToLower(strings.TrimSpace(request.RunKind))
	if kind == "" {
		kind = strings.ToLower(strings.TrimSpace(resolveMetadataString(request.Metadata, "runKind")))
	}
	switch kind {
	case "user", "heartbeat", "cron", "subagent":
		return kind
	}
	if value, ok := resolveMetadataBool(request.Metadata, "heartbeat"); ok && value {
		return "heartbeat"
	}
	if value, ok := resolveMetadataBool(request.Metadata, "cron"); ok && value {
		return "cron"
	}
	if flags.IsSubagent {
		return "subagent"
	}
	return "user"
}

func resolvePromptMode(requested string, workspaceMode string, isSubagent bool) string {
	mode := strings.ToLower(strings.TrimSpace(requested))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(workspaceMode))
	}
	if mode == "" {
		if isSubagent {
			return domainassistant.PromptModeMinimal
		}
		return domainassistant.PromptModeFull
	}
	switch mode {
	case domainassistant.PromptModeFull:
		return domainassistant.PromptModeFull
	case domainassistant.PromptModeMinimal:
		return domainassistant.PromptModeMinimal
	case domainassistant.PromptModeNone:
		return domainassistant.PromptModeNone
	default:
		if isSubagent {
			return domainassistant.PromptModeMinimal
		}
		return domainassistant.PromptModeFull
	}
}

func resolveExtraSystemPrompt(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	prompt := resolveMetadataString(metadata, "extraSystemPrompt")
	if prompt == "" {
		prompt = resolveMetadataString(metadata, "systemPrompt")
	}
	return strings.TrimSpace(prompt)
}

func resolveLoopMaxSteps(metadata map[string]any, isSubagent bool) int {
	if value, ok := resolveMetadataInt(metadata, "maxSteps"); ok {
		if value <= 0 {
			return 0
		}
		return value
	}
	if isSubagent {
		return defaultSubagentMaxSteps
	}
	return 0
}

func resolveToolLoopDetectionConfig(metadata map[string]any, settings settingsdto.GatewayToolLoopSettings) agentruntime.ToolLoopDetectionConfig {
	config := agentruntime.ToolLoopDetectionConfig{
		Enabled:                       settings.Enabled,
		WarnThreshold:                 settings.WarnThreshold,
		CriticalThreshold:             firstNonZero(settings.CriticalThreshold, settings.AbortThreshold),
		GlobalCircuitBreakerThreshold: settings.GlobalCircuitBreakerThreshold,
		HistorySize:                   firstNonZero(settings.HistorySize, settings.WindowSize),
		Detectors: agentruntime.ToolLoopDetectors{
			GenericRepeat:       settings.Detectors.GenericRepeat,
			KnownPollNoProgress: settings.Detectors.KnownPollNoProgress,
			PingPong:            settings.Detectors.PingPong,
		},
	}
	if value, ok := resolveMetadataBool(metadata, "toolLoopEnabled"); ok {
		config.Enabled = value
	}
	if value, ok := resolveMetadataInt(metadata, "toolLoopThreshold"); ok && value > 0 {
		config.CriticalThreshold = value
	}
	if value, ok := resolveMetadataInt(metadata, "toolLoopWarnThreshold"); ok && value > 0 {
		config.WarnThreshold = value
	}
	if value, ok := resolveMetadataInt(metadata, "toolLoopHistorySize"); ok && value > 0 {
		config.HistorySize = value
	}
	if value, ok := resolveMetadataInt(metadata, "toolLoopWindowSize"); ok && value > 0 {
		config.HistorySize = value
	}
	if value, ok := resolveMetadataInt(metadata, "toolLoopGlobalThreshold"); ok && value > 0 {
		config.GlobalCircuitBreakerThreshold = value
	}
	return config
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func resolveRunLane(metadata map[string]any, isSubagent bool) string {
	if metadata != nil {
		switch strings.ToLower(strings.TrimSpace(resolveMetadataString(metadata, "runKind"))) {
		case "heartbeat", "cron":
			return queue.LaneCron
		case "subagent":
			return queue.LaneSubagent
		}
		if lane := resolveMetadataString(metadata, "lane"); lane != "" {
			return lane
		}
		if lane := resolveMetadataString(metadata, "queueLane"); lane != "" {
			return lane
		}
		if lane := resolveMetadataString(metadata, "runLane"); lane != "" {
			return lane
		}
		if value, ok := resolveMetadataBool(metadata, "cron"); ok && value {
			return queue.LaneCron
		}
		if value, ok := resolveMetadataBool(metadata, "heartbeat"); ok && value {
			return queue.LaneCron
		}
	}
	if isSubagent {
		return queue.LaneSubagent
	}
	return queue.LaneMain
}

func resolveLoopTimeout(metadata map[string]any, isSubagent bool) time.Duration {
	if value, ok := resolveMetadataInt(metadata, "timeoutSeconds"); ok {
		if value <= 0 {
			return 0
		}
		return time.Duration(value) * time.Second
	}
	if value, ok := resolveMetadataInt(metadata, "runTimeoutSeconds"); ok {
		if value <= 0 {
			return 0
		}
		return time.Duration(value) * time.Second
	}
	if value, ok := resolveMetadataInt(metadata, "timeoutMs"); ok {
		if value <= 0 {
			return 0
		}
		return time.Duration(value) * time.Millisecond
	}
	if isSubagent {
		return time.Duration(defaultSubagentTimeoutSeconds) * time.Second
	}
	return 0
}

func mergeToolExecutionConfig(config runtimedto.ToolExecutionConfig, call domainassistant.AssistantCall, isSubagent bool) runtimedto.ToolExecutionConfig {
	if isSubagent {
		return config
	}
	if strings.TrimSpace(config.Mode) == "" {
		config.Mode = call.Tools.Mode
	}
	if len(config.AllowList) == 0 {
		config.AllowList = call.Tools.AllowList
	}
	if len(config.DenyList) == 0 {
		config.DenyList = call.Tools.DenyList
	}
	return config
}

func (service *Service) filterToolSpecs(specs []tooldto.ToolSpec, config runtimedto.ToolExecutionConfig, assistantTools domainassistant.AssistantTools) []tooldto.ToolSpec {
	if len(specs) == 0 {
		return nil
	}
	if isToolModeDisabled(config.Mode) {
		return nil
	}
	assistantAllowed := make(map[string]struct{})
	if len(assistantTools.Items) > 0 {
		for _, item := range assistantTools.Items {
			if !item.Enabled {
				continue
			}
			id := strings.TrimSpace(item.ID)
			if id == "" {
				continue
			}
			assistantAllowed[strings.ToLower(id)] = struct{}{}
		}
	}
	allowSet := make(map[string]struct{})
	for _, item := range config.AllowList {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		allowSet[strings.ToLower(trimmed)] = struct{}{}
	}
	denySet := make(map[string]struct{})
	for _, item := range config.DenyList {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		denySet[strings.ToLower(trimmed)] = struct{}{}
	}
	result := make([]tooldto.ToolSpec, 0, len(specs))
	for _, spec := range specs {
		if !spec.Enabled {
			continue
		}
		name := strings.TrimSpace(spec.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if len(assistantAllowed) > 0 {
			if _, ok := assistantAllowed[key]; !ok {
				continue
			}
		}
		if len(allowSet) > 0 {
			if _, ok := allowSet[key]; !ok {
				continue
			}
		}
		if _, denied := denySet[key]; denied {
			continue
		}
		result = append(result, spec)
	}
	return result
}

func isToolModeDisabled(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "off", "none", "disabled":
		return true
	default:
		return false
	}
}

func (service *Service) resolveSkillPromptItems(
	ctx context.Context,
	providerID string,
	_ domainassistant.AssistantCall,
	assistantSkills domainassistant.AssistantSkills,
	promptMode string,
	isSubagent bool,
	workspaceRoot string,
) []skillsdto.SkillPromptItem {
	if service == nil || service.skills == nil {
		return nil
	}
	if (promptMode != domainassistant.PromptModeFull && promptMode != domainassistant.PromptModeMinimal) || isSubagent {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(assistantSkills.Mode), domainassistant.SkillsModeOff) {
		return nil
	}
	response, err := service.skills.ResolveSkillPromptItemsForWorkspace(ctx, skillsdto.ResolveSkillPromptRequest{
		ProviderID: strings.TrimSpace(providerID),
	}, workspaceRoot)
	if err != nil {
		return nil
	}
	return attachSkillPaths(response.Items, workspaceRoot)
}

func attachSkillPaths(items []skillsdto.SkillPromptItem, workspaceRoot string) []skillsdto.SkillPromptItem {
	if len(items) == 0 {
		return nil
	}
	for index := range items {
		id := strings.TrimSpace(items[index].ID)
		if id == "" {
			continue
		}
		items[index].Path = skillsservice.ResolveSkillDocumentPath(id, workspaceRoot)
	}
	return items
}

func resolveMetadataBool(metadata map[string]any, key string) (bool, bool) {
	if metadata == nil || key == "" {
		return false, false
	}
	value, ok := metadata[key]
	if !ok {
		return false, false
	}
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return false, false
		}
		parsed, err := strconv.ParseBool(trimmed)
		if err != nil {
			return false, false
		}
		return parsed, true
	case int:
		return typed != 0, true
	case int64:
		return typed != 0, true
	case float64:
		return typed != 0, true
	default:
		return false, false
	}
}

func resolveMetadataInt(metadata map[string]any, key string) (int, bool) {
	if metadata == nil || key == "" {
		return 0, false
	}
	value, ok := metadata[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func resolveMetadataFloat(metadata map[string]any, key string) (float32, bool) {
	if metadata == nil || key == "" {
		return 0, false
	}
	value, ok := metadata[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float32:
		return typed, true
	case float64:
		return float32(typed), true
	case int:
		return float32(typed), true
	case int64:
		return float32(typed), true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(trimmed, 32)
		if err != nil {
			return 0, false
		}
		return float32(parsed), true
	default:
		return 0, false
	}
}

package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"

	"dreamcreator/internal/application/agentruntime"
	assistantdto "dreamcreator/internal/application/assistant/dto"
	"dreamcreator/internal/application/gateway/queue"
	"dreamcreator/internal/application/gateway/runtime/dto"
	gatewaytools "dreamcreator/internal/application/gateway/tools"
	settingsdto "dreamcreator/internal/application/settings/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
	domainassistant "dreamcreator/internal/domain/assistant"
	domainsession "dreamcreator/internal/domain/session"
	"dreamcreator/internal/domain/settings"
)

func (service *Service) ensureAssistantReady(ctx context.Context, request dto.RuntimeRunRequest, snapshot assistantdto.AssistantSnapshot) error {
	if service == nil || service.assistantSnapshots == nil {
		return nil
	}
	if snapshot.AssistantID == "" {
		sessionID, _, err := service.resolveSession(request)
		if err != nil {
			return err
		}
		loaded, err := service.assistantSnapshots.ResolveAssistantSnapshot(ctx, assistantdto.ResolveAssistantSnapshotRequest{
			ThreadID:    sessionID,
			AssistantID: strings.TrimSpace(request.AssistantID),
		})
		if err != nil {
			return err
		}
		snapshot = loaded
	}
	missing, err := service.checkAssistantReady(ctx, snapshot)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	return errors.New("assistant not ready: missing " + strings.Join(missing, ","))
}

func (service *Service) checkAssistantReady(ctx context.Context, snapshot assistantdto.AssistantSnapshot) ([]string, error) {
	missing := make([]string, 0, 2)
	if strings.TrimSpace(snapshot.Model.Agent.Primary) == "" {
		missing = append(missing, "model.agent.primary")
	}

	modelIndex, hasModels, err := service.buildModelIndex(ctx)
	if err != nil {
		return nil, err
	}
	if !hasModels {
		missing = append(missing, "providers.models")
	}
	if strings.TrimSpace(snapshot.Model.Agent.Primary) != "" {
		if providerID, modelName, err := parseModelRef(snapshot.Model.Agent.Primary); err != nil {
			missing = append(missing, "model.agent.primary")
		} else if !modelExists(modelIndex, providerID, modelName) {
			missing = append(missing, "model.agent.primary")
		}
	}
	return missing, nil
}

func (service *Service) buildModelIndex(ctx context.Context) (map[string]map[string]struct{}, bool, error) {
	index := make(map[string]map[string]struct{})
	if service == nil || service.providers == nil || service.models == nil {
		return index, false, nil
	}
	providersList, err := service.providers.List(ctx)
	if err != nil {
		return index, false, err
	}
	hasModels := false
	for _, provider := range providersList {
		if !provider.Enabled {
			continue
		}
		modelsList, err := service.models.ListByProvider(ctx, provider.ID)
		if err != nil {
			return index, false, err
		}
		for _, item := range modelsList {
			if !item.Enabled {
				continue
			}
			hasModels = true
			providerKey := strings.ToLower(strings.TrimSpace(item.ProviderID))
			modelKey := strings.ToLower(strings.TrimSpace(item.Name))
			if providerKey == "" || modelKey == "" {
				continue
			}
			bucket := index[providerKey]
			if bucket == nil {
				bucket = make(map[string]struct{})
				index[providerKey] = bucket
			}
			bucket[modelKey] = struct{}{}
		}
	}
	return index, hasModels, nil
}

func modelExists(index map[string]map[string]struct{}, providerID string, modelName string) bool {
	providerKey := strings.ToLower(strings.TrimSpace(providerID))
	modelKey := strings.ToLower(strings.TrimSpace(modelName))
	if providerKey == "" || modelKey == "" {
		return false
	}
	bucket, ok := index[providerKey]
	if !ok {
		return false
	}
	_, ok = bucket[modelKey]
	return ok
}

func (service *Service) updateQueuePolicy(settings settingsdto.GatewaySettings) {
	if service == nil || service.queue == nil {
		return
	}
	queueCaps := queue.GlobalCaps{}
	if settings.Queue.GlobalConcurrency > 0 {
		queueCaps.Steer = settings.Queue.GlobalConcurrency
		queueCaps.Followup = settings.Queue.GlobalConcurrency
		queueCaps.Collect = settings.Queue.GlobalConcurrency
	}
	policy := queue.Policy{
		DefaultMode: string(domainsession.QueueModeFollowup),
		DefaultCap:  settings.Queue.SessionConcurrency,
		GlobalCaps:  queueCaps,
	}
	service.queue.UpdatePolicy(policy)
	lanes := queue.LaneCaps{
		Main:     settings.Queue.Lanes.Main,
		Subagent: settings.Queue.Lanes.Subagent,
		Cron:     settings.Queue.Lanes.Cron,
	}
	if lanes.Subagent <= 0 && settings.Subagents.MaxConcurrent > 0 {
		lanes.Subagent = settings.Subagents.MaxConcurrent
	}
	service.queue.UpdateLaneCaps(lanes)
}

type resolvedRunModel struct {
	ProviderID string
	ModelName  string
	Config     domainassistant.ModelConfig
}

func (service *Service) resolveRunModel(ctx context.Context, override *dto.ModelSelection, assistantModel domainassistant.AssistantModel) (resolvedRunModel, model.BaseChatModel, error) {
	config := assistantModel.Agent
	if override != nil {
		providerID := strings.TrimSpace(override.ProviderID)
		modelName := strings.TrimSpace(override.Name)
		if providerID != "" && modelName != "" {
			chatModel, resolvedProviderID, resolvedModelName, err := service.resolveChatModel(ctx, providerID, modelName)
			if err != nil {
				return resolvedRunModel{}, nil, err
			}
			return resolvedRunModel{
				ProviderID: resolvedProviderID,
				ModelName:  resolvedModelName,
				Config:     config,
			}, chatModel, nil
		}
	}
	candidates := buildModelCandidates(config)
	if len(candidates) == 0 {
		return resolvedRunModel{}, nil, errors.New("assistant agent model not configured")
	}
	var lastErr error
	for _, ref := range candidates {
		providerID, modelName, err := parseModelRef(ref)
		if err != nil {
			lastErr = err
			continue
		}
		chatModel, resolvedProviderID, resolvedModelName, err := service.resolveChatModel(ctx, providerID, modelName)
		if err == nil {
			return resolvedRunModel{
				ProviderID: resolvedProviderID,
				ModelName:  resolvedModelName,
				Config:     config,
			}, chatModel, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("model not available")
	}
	return resolvedRunModel{}, nil, lastErr
}

func buildModelCandidates(config domainassistant.ModelConfig) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, 1+len(config.Fallbacks))
	if primary := strings.TrimSpace(config.Primary); primary != "" {
		key := strings.ToLower(primary)
		seen[key] = struct{}{}
		result = append(result, primary)
	}
	for _, fallback := range config.Fallbacks {
		trimmed := strings.TrimSpace(fallback)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func parseModelRef(value string) (string, string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", "", errors.New("model ref is empty")
	}
	if strings.Contains(trimmed, "/") {
		parts := strings.SplitN(trimmed, "/", 2)
		providerID := strings.TrimSpace(parts[0])
		modelName := strings.TrimSpace(parts[1])
		if providerID == "" || modelName == "" {
			return "", "", errors.New("model ref must include provider prefix")
		}
		return providerID, modelName, nil
	}
	if strings.Contains(trimmed, ":") {
		parts := strings.SplitN(trimmed, ":", 2)
		providerID := strings.TrimSpace(parts[0])
		modelName := strings.TrimSpace(parts[1])
		if providerID == "" || modelName == "" {
			return "", "", errors.New("model ref must include provider prefix")
		}
		return providerID, modelName, nil
	}
	return "", "", errors.New("model ref must include provider prefix")
}

func (service *Service) resolveChatModel(ctx context.Context, providerID string, modelName string) (model.BaseChatModel, string, string, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, "", "", errors.New("provider id is required")
	}
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return nil, "", "", errors.New("model name is required")
	}
	if service.providers == nil || service.secrets == nil || service.chatFactory == nil {
		return nil, "", "", errors.New("provider repositories unavailable")
	}
	provider, err := service.providers.Get(ctx, providerID)
	if err != nil {
		return nil, "", "", err
	}
	secret, err := service.secrets.GetByProviderID(ctx, providerID)
	if err != nil {
		return nil, "", "", err
	}
	apiKey := strings.TrimSpace(secret.APIKey)
	if apiKey == "" {
		return nil, "", "", errors.New("provider api key missing")
	}
	chatModel, err := service.chatFactory.NewChatModel(provider, apiKey, modelName)
	if err != nil {
		return nil, "", "", err
	}
	return chatModel, provider.ID, modelName, nil
}

func resolveToolCallingModel(chatModel model.BaseChatModel) (model.ToolCallingChatModel, bool) {
	toolModel, ok := chatModel.(model.ToolCallingChatModel)
	return toolModel, ok
}

func (service *Service) resolveToolAdapters(ctx context.Context, sessionKey string, runID string, config dto.ToolExecutionConfig, assistantTools domainassistant.AssistantTools, policyCtx tooldto.ToolPolicyContext) ([]*schema.ToolInfo, map[string]agentruntime.ToolDefinition) {
	if service == nil || service.tools == nil {
		return nil, map[string]agentruntime.ToolDefinition{}
	}
	if isToolModeDisabled(config.Mode) {
		return nil, map[string]agentruntime.ToolDefinition{}
	}
	specs := service.filterToolSpecs(service.tools.ListTools(ctx), config, assistantTools)
	if len(specs) == 0 {
		return nil, map[string]agentruntime.ToolDefinition{}
	}
	if policyCtx.SessionKey == "" {
		policyCtx.SessionKey = strings.TrimSpace(sessionKey)
	}
	infos := make([]*schema.ToolInfo, 0, len(specs))
	tools := make(map[string]agentruntime.ToolDefinition)
	for _, spec := range specs {
		adapter := &toolAdapter{
			spec:       spec,
			service:    service.tools,
			sessionKey: sessionKey,
			runID:      runID,
			policyCtx:  policyCtx,
		}
		info, err := adapter.Info(ctx)
		if err != nil || info == nil {
			continue
		}
		infos = append(infos, info)
		adapterRef := adapter
		tools[spec.Name] = agentruntime.ToolDefinition{
			Name: spec.Name,
			Type: spec.Kind,
			Invoke: func(toolCtx context.Context, args string) (string, error) {
				return adapterRef.InvokableRun(toolCtx, args)
			},
		}
	}
	return infos, tools
}

type toolAdapter struct {
	spec       tooldto.ToolSpec
	service    *gatewaytools.Service
	sessionKey string
	runID      string
	policyCtx  tooldto.ToolPolicyContext
}

func (adapter *toolAdapter) Info(_ context.Context) (*schema.ToolInfo, error) {
	if adapter == nil {
		return nil, errors.New("tool adapter unavailable")
	}
	info := &schema.ToolInfo{
		Name: adapter.spec.Name,
		Desc: adapter.spec.Description,
	}
	if strings.TrimSpace(adapter.spec.SchemaJSON) == "" {
		return info, nil
	}
	var schemaDef jsonschema.Schema
	if err := json.Unmarshal([]byte(adapter.spec.SchemaJSON), &schemaDef); err != nil {
		return info, nil
	}
	info.ParamsOneOf = schema.NewParamsOneOfByJSONSchema(&schemaDef)
	return info, nil
}

func (adapter *toolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	if adapter == nil || adapter.service == nil {
		return "", errors.New("tool service unavailable")
	}
	ctx = gatewaytools.WithRuntimeContext(ctx, adapter.sessionKey, adapter.runID)
	toolCallID, _ := agentruntime.ToolCallContextFromContext(ctx)
	req := tooldto.ToolsInvokeRequest{
		Tool:       adapter.spec.Name,
		ToolCallID: strings.TrimSpace(toolCallID),
		Args:       strings.TrimSpace(argumentsInJSON),
		SessionKey: adapter.sessionKey,
	}
	resp, err := adapter.service.InvokeWithPolicy(ctx, req, adapter.policyCtx)
	if err != nil {
		return "", err
	}
	if resp.Result.ErrorMessage != "" {
		return "", errors.New(resp.Result.ErrorMessage)
	}
	output := strings.TrimSpace(resp.Result.OutputJSON)
	if output == "" {
		output = "null"
	}
	return output, nil
}

func (service *Service) buildChatOptions(ctx context.Context, modelConfig domainassistant.ModelConfig, metadata map[string]any) []model.Option {
	temperature := modelConfig.Temperature
	maxTokens := modelConfig.MaxTokens
	if (temperature <= 0 || maxTokens <= 0) && service.settings != nil {
		if loaded, err := service.settings.GetSettings(ctx); err == nil {
			if temperature <= 0 {
				temperature = loaded.ChatTemperature
			}
			if maxTokens <= 0 {
				maxTokens = loaded.ChatMaxTokens
			}
		}
	}
	if value, ok := resolveMetadataFloat(metadata, "temperature"); ok {
		temperature = value
	}
	if value, ok := resolveMetadataInt(metadata, "maxTokens"); ok {
		maxTokens = value
	}
	options := make([]model.Option, 0, 2)
	if maxTokens <= 0 {
		maxTokens = settings.DefaultChatMaxTokens
	}
	if temperature < settings.MinChatTemperature {
		temperature = settings.MinChatTemperature
	}
	if temperature > settings.MaxChatTemperature {
		temperature = settings.MaxChatTemperature
	}
	options = append(options, model.WithTemperature(temperature))

	if maxTokens < settings.MinChatMaxTokens {
		maxTokens = settings.MinChatMaxTokens
	}
	if maxTokens > settings.MaxChatMaxTokens {
		maxTokens = settings.MaxChatMaxTokens
	}
	if maxTokens > 0 {
		options = append(options, model.WithMaxTokens(maxTokens))
	}
	return options
}

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/schema"

	memorydto "dreamcreator/internal/application/memory/dto"
	settingsdto "dreamcreator/internal/application/settings/dto"
	"dreamcreator/internal/domain/providers"
)

func (service *MemoryService) Embed(ctx context.Context, request memorydto.EmbedRequest) ([]float32, error) {
	input := strings.TrimSpace(request.Input)
	if input == "" {
		return nil, errors.New("input is required")
	}
	providerID := strings.TrimSpace(request.ProviderID)
	modelName := strings.TrimSpace(request.ModelName)
	assistantID := strings.TrimSpace(request.AssistantID)
	if providerID == "" || modelName == "" {
		settingsValue := service.loadMemorySettings(ctx)
		providerID, modelName = service.resolveEmbeddingModel(ctx, assistantID, settingsValue)
	}
	if providerID == "" || modelName == "" {
		return nil, errors.New("embedding provider/model is not configured")
	}
	provider, secret, err := service.resolveProviderAndSecret(ctx, providerID)
	if err != nil {
		return nil, err
	}
	vector, err := service.callEmbeddingAPI(ctx, provider, secret, modelName, input)
	if err != nil {
		return nil, err
	}
	return vector, nil
}

func (service *MemoryService) resolveEmbeddingModel(
	ctx context.Context,
	assistantID string,
	settingsValue settingsdto.MemorySettings,
) (string, string) {
	if providerID, modelName := service.resolveAssistantEmbeddingModel(ctx, assistantID); providerID != "" && modelName != "" {
		return providerID, modelName
	}
	providerID := strings.TrimSpace(settingsValue.EmbeddingProvider)
	modelName := strings.TrimSpace(settingsValue.EmbeddingModel)
	if providerID != "" && modelName != "" {
		return providerID, modelName
	}
	if strings.TrimSpace(settingsValue.LLMProvider) != "" && strings.TrimSpace(settingsValue.LLMModel) != "" {
		return strings.TrimSpace(settingsValue.LLMProvider), strings.TrimSpace(settingsValue.LLMModel)
	}
	if service.settings != nil {
		if current, err := service.settings.GetSettings(context.Background()); err == nil {
			providerID = strings.TrimSpace(current.AgentModelProviderID)
			modelName = strings.TrimSpace(current.AgentModelName)
			if providerID != "" && modelName != "" {
				return providerID, modelName
			}
		}
	}
	return "", ""
}

func (service *MemoryService) resolveLLMModel(
	ctx context.Context,
	assistantID string,
	settingsValue settingsdto.MemorySettings,
) (string, string) {
	if providerID, modelName := service.resolveAssistantAgentModel(ctx, assistantID); providerID != "" && modelName != "" {
		return providerID, modelName
	}
	providerID := strings.TrimSpace(settingsValue.LLMProvider)
	modelName := strings.TrimSpace(settingsValue.LLMModel)
	if providerID != "" && modelName != "" {
		return providerID, modelName
	}
	if service.settings != nil {
		if current, err := service.settings.GetSettings(context.Background()); err == nil {
			providerID = strings.TrimSpace(current.AgentModelProviderID)
			modelName = strings.TrimSpace(current.AgentModelName)
			if providerID != "" && modelName != "" {
				return providerID, modelName
			}
		}
	}
	return "", ""
}

func (service *MemoryService) resolveAssistantEmbeddingModel(ctx context.Context, assistantID string) (string, string) {
	assistantID = strings.TrimSpace(assistantID)
	if assistantID == "" || service.assistants == nil {
		return "", ""
	}
	item, err := service.assistants.Get(ctx, assistantID)
	if err != nil {
		return "", ""
	}

	if item.Model.Embedding.Inherit {
		if providerID, modelName := parseProviderModelRef(item.Model.Agent.Primary); providerID != "" && modelName != "" {
			return providerID, modelName
		}
		if providerID, modelName := parseFirstModelRef(item.Model.Agent.Fallbacks); providerID != "" && modelName != "" {
			return providerID, modelName
		}
		return "", ""
	}

	if providerID, modelName := parseProviderModelRef(item.Model.Embedding.Primary); providerID != "" && modelName != "" {
		return providerID, modelName
	}
	if providerID, modelName := parseFirstModelRef(item.Model.Embedding.Fallbacks); providerID != "" && modelName != "" {
		return providerID, modelName
	}
	return "", ""
}

func (service *MemoryService) resolveAssistantAgentModel(ctx context.Context, assistantID string) (string, string) {
	assistantID = strings.TrimSpace(assistantID)
	if assistantID == "" || service.assistants == nil {
		return "", ""
	}
	item, err := service.assistants.Get(ctx, assistantID)
	if err != nil {
		return "", ""
	}
	if providerID, modelName := parseProviderModelRef(item.Model.Agent.Primary); providerID != "" && modelName != "" {
		return providerID, modelName
	}
	if providerID, modelName := parseFirstModelRef(item.Model.Agent.Fallbacks); providerID != "" && modelName != "" {
		return providerID, modelName
	}
	return "", ""
}

func parseFirstModelRef(values []string) (string, string) {
	for _, value := range values {
		if providerID, modelName := parseProviderModelRef(value); providerID != "" && modelName != "" {
			return providerID, modelName
		}
	}
	return "", ""
}

func parseProviderModelRef(value string) (string, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ""
	}

	if index := strings.Index(trimmed, "/"); index > 0 {
		providerID := strings.TrimSpace(trimmed[:index])
		modelName := strings.TrimSpace(trimmed[index+1:])
		if providerID != "" && modelName != "" {
			return providerID, modelName
		}
	}
	if index := strings.Index(trimmed, ":"); index > 0 {
		providerID := strings.TrimSpace(trimmed[:index])
		modelName := strings.TrimSpace(trimmed[index+1:])
		if providerID != "" && modelName != "" {
			return providerID, modelName
		}
	}
	return "", ""
}

func (service *MemoryService) resolveProviderAndSecret(ctx context.Context, providerID string) (providers.Provider, providers.ProviderSecret, error) {
	if service.providers == nil || service.secrets == nil {
		return providers.Provider{}, providers.ProviderSecret{}, errors.New("provider repositories unavailable")
	}
	provider, err := service.providers.Get(ctx, providerID)
	if err != nil {
		return providers.Provider{}, providers.ProviderSecret{}, err
	}
	secret, err := service.secrets.GetByProviderID(ctx, providerID)
	if err != nil {
		return providers.Provider{}, providers.ProviderSecret{}, err
	}
	if strings.TrimSpace(secret.APIKey) == "" {
		return providers.Provider{}, providers.ProviderSecret{}, errors.New("provider api key is missing")
	}
	return provider, secret, nil
}

func (service *MemoryService) callEmbeddingAPI(
	ctx context.Context,
	provider providers.Provider,
	secret providers.ProviderSecret,
	modelName string,
	input string,
) ([]float32, error) {
	endpoint := strings.TrimSpace(provider.Endpoint)
	if endpoint == "" {
		return nil, errors.New("provider endpoint is required")
	}
	url := strings.TrimRight(endpoint, "/") + "/embeddings"
	payload := map[string]any{
		"model": modelName,
		"input": input,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(secret.APIKey))

	response, err := service.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("embedding request failed: status=%d body=%s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	var decoded struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	if len(decoded.Data) == 0 || len(decoded.Data[0].Embedding) == 0 {
		return nil, errors.New("embedding response missing vector")
	}
	result := make([]float32, 0, len(decoded.Data[0].Embedding))
	for _, value := range decoded.Data[0].Embedding {
		result = append(result, float32(value))
	}
	return result, nil
}

func (service *MemoryService) extractCandidatesByLLM(
	ctx context.Context,
	providerID string,
	modelName string,
	transcript string,
	maxEntries int,
) ([]memoryExtractCandidate, error) {
	response, err := service.runLLMText(ctx, providerID, modelName,
		"You are a memory extraction engine. Extract only stable, reusable long-term memory. Output STRICT JSON array only.",
		buildExtractPrompt(transcript, maxEntries),
	)
	if err != nil {
		return nil, err
	}
	jsonPayload := extractJSONArray(response)
	if jsonPayload == "" {
		return nil, nil
	}
	items := make([]memoryExtractCandidate, 0)
	if err := json.Unmarshal([]byte(jsonPayload), &items); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	result := make([]memoryExtractCandidate, 0, len(items))
	for _, item := range items {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		key := normalizeComparableText(content)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		item.Content = content
		item.Category = normalizeMemoryCategory(item.Category)
		if item.Category == "" {
			item.Category = string(memorydto.MemoryCategoryOther)
		}
		item.Confidence = clampFloat(item.Confidence, 0.1, 1)
		result = append(result, item)
		if len(result) >= maxEntries {
			break
		}
	}
	return result, nil
}

func (service *MemoryService) summarizeSessionByLLM(
	ctx context.Context,
	providerID string,
	modelName string,
	transcript string,
) (sessionSummaryResult, error) {
	response, err := service.runLLMText(ctx, providerID, modelName,
		"You summarize sessions and distill memory candidates. Output STRICT JSON object only.",
		buildSessionSummaryPrompt(transcript),
	)
	if err != nil {
		return sessionSummaryResult{}, err
	}
	payload := extractJSONObject(response)
	if payload == "" {
		return sessionSummaryResult{Summary: strings.TrimSpace(response)}, nil
	}
	decoded := sessionSummaryResult{}
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return sessionSummaryResult{}, err
	}
	decoded.Summary = strings.TrimSpace(decoded.Summary)
	if len(decoded.Memories) > defaultCaptureMax {
		decoded.Memories = decoded.Memories[:defaultCaptureMax]
	}
	for i := range decoded.Memories {
		decoded.Memories[i].Category = normalizeMemoryCategory(decoded.Memories[i].Category)
		if decoded.Memories[i].Category == "" {
			decoded.Memories[i].Category = string(memorydto.MemoryCategoryOther)
		}
		decoded.Memories[i].Confidence = clampFloat(decoded.Memories[i].Confidence, 0.1, 1)
	}
	return decoded, nil
}

func (service *MemoryService) runLLMText(
	ctx context.Context,
	providerID string,
	modelName string,
	systemPrompt string,
	userPrompt string,
) (string, error) {
	if service.chatFactory == nil {
		return "", errors.New("chat factory unavailable")
	}
	provider, secret, err := service.resolveProviderAndSecret(ctx, providerID)
	if err != nil {
		return "", err
	}
	chatModel, err := service.chatFactory.NewChatModel(provider, strings.TrimSpace(secret.APIKey), modelName)
	if err != nil {
		return "", err
	}
	result, err := chatModel.Generate(ctx, []*schema.Message{
		{Role: schema.System, Content: strings.TrimSpace(systemPrompt)},
		{Role: schema.User, Content: strings.TrimSpace(userPrompt)},
	})
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	return strings.TrimSpace(result.Content), nil
}

func buildExtractPrompt(transcript string, maxEntries int) string {
	if maxEntries <= 0 {
		maxEntries = defaultCaptureMax
	}
	return strings.TrimSpace(fmt.Sprintf(`From the conversation below, extract up to %d durable long-term memories.

Rules:
1. Keep only stable facts/preferences/decisions/entities likely useful later.
2. Ignore transient chatter and tasks already completed.
3. Return JSON array only. No markdown, no explanation.
4. Each item schema: {"content":"...","category":"preference|fact|decision|entity|reflection|other","confidence":0..1}

Conversation:
%s`, maxEntries, transcript))
}

func buildSessionSummaryPrompt(transcript string) string {
	return strings.TrimSpace(`Summarize this session and distill key long-term memory candidates.

Return JSON object only:
{
  "summary": "short session summary",
  "memories": [
    {"content":"...","category":"preference|fact|decision|entity|reflection|other","confidence":0..1}
  ]
}

Limit memories to 3 items max.

Conversation:
` + transcript)
}

func buildTranscript(messages []memorydto.MemoryMessage, maxChars int) string {
	if len(messages) == 0 {
		return ""
	}
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		if role == "" {
			role = "unknown"
		}
		lines = append(lines, fmt.Sprintf("%s: %s", role, content))
	}
	transcript := strings.TrimSpace(strings.Join(lines, "\n"))
	if transcript == "" {
		return ""
	}
	if maxChars > 0 && len([]rune(transcript)) > maxChars {
		r := []rune(transcript)
		transcript = strings.TrimSpace(string(r[len(r)-maxChars:]))
	}
	return transcript
}

func fallbackSessionSummary(messages []memorydto.MemoryMessage) string {
	if len(messages) == 0 {
		return ""
	}
	lastUser := ""
	lastAssistant := ""
	for _, msg := range messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		switch role {
		case "user":
			lastUser = content
		case "assistant":
			lastAssistant = content
		}
	}
	result := "Session summary"
	if lastUser != "" {
		result += ": user intent = " + truncateRunes(lastUser, 140)
	}
	if lastAssistant != "" {
		result += "; assistant response = " + truncateRunes(lastAssistant, 140)
	}
	return result
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= max {
		return trimmed
	}
	return strings.TrimSpace(string(runes[:max])) + "..."
}

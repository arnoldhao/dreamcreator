package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	domainproviders "dreamcreator/internal/domain/providers"
)

const (
	defaultChatPath          = "/chat/completions"
	defaultHTTPClientTimeout = 60 * time.Second
	defaultStreamIdleTimeout = 90 * time.Second
)

var errStreamIdleTimeout = errors.New("stream idle timeout")

type OpenAICompatibleConfig struct {
	BaseURL               string
	APIKey                string
	Model                 string
	ProviderID            string
	ProviderType          domainproviders.ProviderType
	ProviderCompatibility domainproviders.ProviderCompatibility
	ChatPath              string
	Headers               map[string]string
	HTTPClient            *http.Client
	StreamIdleTimeout     time.Duration
	Recorder              CallRecorder
}

type OpenAICompatibleChatModel struct {
	baseURL           string
	apiKey            string
	model             string
	providerID        string
	providerType      domainproviders.ProviderType
	providerCompat    domainproviders.ProviderCompatibility
	chatPath          string
	headers           map[string]string
	client            *http.Client
	streamIdleTimeout time.Duration
	tools             []*schema.ToolInfo
	toolChoice        *schema.ToolChoice
	allowedToolNames  []string
	recorder          CallRecorder
}

func NewOpenAICompatibleChatModel(config OpenAICompatibleConfig) (*OpenAICompatibleChatModel, error) {
	base := strings.TrimSpace(config.BaseURL)
	if base == "" {
		return nil, fmt.Errorf("base url is required")
	}
	chatPath := strings.TrimSpace(config.ChatPath)
	if chatPath == "" {
		chatPath = defaultChatPath
	}

	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPClientTimeout}
	}
	streamIdleTimeout := config.StreamIdleTimeout
	if streamIdleTimeout <= 0 {
		streamIdleTimeout = defaultStreamIdleTimeout
	}
	providerType := config.ProviderType
	if providerType == "" {
		providerType = domainproviders.ProviderTypeOpenAI
	}
	providerCompat := config.ProviderCompatibility
	if providerCompat == "" {
		switch providerType {
		case domainproviders.ProviderTypeAnthropic:
			providerCompat = domainproviders.ProviderCompatibilityAnthropic
		default:
			providerCompat = domainproviders.ProviderCompatibilityOpenAI
		}
	}

	return &OpenAICompatibleChatModel{
		baseURL:           strings.TrimRight(base, "/"),
		apiKey:            strings.TrimSpace(config.APIKey),
		model:             strings.TrimSpace(config.Model),
		providerID:        strings.TrimSpace(config.ProviderID),
		providerType:      providerType,
		providerCompat:    providerCompat,
		chatPath:          chatPath,
		headers:           config.Headers,
		client:            client,
		streamIdleTimeout: streamIdleTimeout,
		recorder:          config.Recorder,
	}, nil
}

func (modelClient *OpenAICompatibleChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	cloned := *modelClient
	cloned.tools = tools
	return &cloned, nil
}

func (modelClient *OpenAICompatibleChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("input messages required")
	}

	commonOpts := model.GetCommonOptions(&model.Options{
		Tools:            modelClient.tools,
		ToolChoice:       modelClient.toolChoice,
		AllowedToolNames: modelClient.allowedToolNames,
	}, opts...)
	modelName := modelClient.model
	if commonOpts.Model != nil && strings.TrimSpace(*commonOpts.Model) != "" {
		modelName = strings.TrimSpace(*commonOpts.Model)
	}
	if modelName == "" {
		return nil, fmt.Errorf("model name is required")
	}
	openAITools, err := toOpenAITools(commonOpts.Tools, commonOpts.AllowedToolNames)
	if err != nil {
		return nil, err
	}
	toolChoice := toOpenAIToolChoice(commonOpts.ToolChoice, commonOpts.AllowedToolNames)

	payload := openAIChatRequest{
		Model:       modelName,
		Messages:    toOpenAIMessages(input),
		Temperature: commonOpts.Temperature,
		MaxTokens:   commonOpts.MaxTokens,
		TopP:        commonOpts.TopP,
		Stop:        commonOpts.Stop,
		Tools:       openAITools,
		ToolChoice:  toolChoice,
	}
	params := runtimeParamsFromContext(ctx)
	applyRuntimeParams(params, providerRequestCompatibility{
		ProviderID:    modelClient.providerID,
		ProviderType:  modelClient.providerType,
		Compatibility: modelClient.providerCompat,
		ModelName:     modelName,
	}, &payload)

	body, err := modelClient.executeChatRequest(ctx, payload)
	if err != nil && shouldRetryWithoutStructuredOutput(err, params.StructuredOutput, payload.ResponseFormat) {
		payload.ResponseFormat = nil
		body, err = modelClient.executeChatRequest(ctx, payload)
	}
	if err != nil {
		return nil, err
	}

	var decoded openAIChatResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("llm response missing choices")
	}

	choice := decoded.Choices[0]
	reasoning := resolveReasoningDelta(choice.Message.ReasoningContent, choice.Message.Reasoning)
	message := &schema.Message{
		Role:             schema.Assistant,
		Content:          choice.Message.Content,
		ReasoningContent: reasoning,
		ToolCalls:        toSchemaToolCalls(choice.Message.ToolCalls),
	}
	if decoded.Usage != nil || choice.FinishReason != "" {
		message.ResponseMeta = &schema.ResponseMeta{
			FinishReason: choice.FinishReason,
		}
		if decoded.Usage != nil {
			message.ResponseMeta.Usage = &schema.TokenUsage{
				PromptTokens:     decoded.Usage.PromptTokens,
				CompletionTokens: decoded.Usage.CompletionTokens,
				TotalTokens:      decoded.Usage.TotalTokens,
			}
		}
	}

	return message, nil
}

func (modelClient *OpenAICompatibleChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("input messages required")
	}

	commonOpts := model.GetCommonOptions(&model.Options{
		Tools:            modelClient.tools,
		ToolChoice:       modelClient.toolChoice,
		AllowedToolNames: modelClient.allowedToolNames,
	}, opts...)

	modelName := modelClient.model
	if commonOpts.Model != nil && strings.TrimSpace(*commonOpts.Model) != "" {
		modelName = strings.TrimSpace(*commonOpts.Model)
	}
	if modelName == "" {
		return nil, fmt.Errorf("model name is required")
	}

	openAITools, err := toOpenAITools(commonOpts.Tools, commonOpts.AllowedToolNames)
	if err != nil {
		return nil, err
	}
	toolChoice := toOpenAIToolChoice(commonOpts.ToolChoice, commonOpts.AllowedToolNames)

	payload := openAIChatRequest{
		Model:         modelName,
		Messages:      toOpenAIMessages(input),
		Temperature:   commonOpts.Temperature,
		MaxTokens:     commonOpts.MaxTokens,
		TopP:          commonOpts.TopP,
		Stop:          commonOpts.Stop,
		Tools:         openAITools,
		ToolChoice:    toolChoice,
		Stream:        true,
		StreamOptions: &openAIStreamOptions{IncludeUsage: true},
	}
	params := runtimeParamsFromContext(ctx)
	applyRuntimeParams(params, providerRequestCompatibility{
		ProviderID:    modelClient.providerID,
		ProviderType:  modelClient.providerType,
		Compatibility: modelClient.providerCompat,
		ModelName:     modelName,
	}, &payload)

	streamCtx, streamCancel := context.WithCancel(ctx)
	streamClient := cloneStreamHTTPClient(modelClient.client)
	response, record, err := modelClient.executeStreamRequest(streamCtx, streamClient, payload)
	if err != nil && shouldRetryWithoutStructuredOutput(err, params.StructuredOutput, payload.ResponseFormat) {
		payload.ResponseFormat = nil
		response, record, err = modelClient.executeStreamRequest(streamCtx, streamClient, payload)
	}
	if err != nil {
		streamCancel()
		return nil, err
	}

	reader, writer := schema.Pipe[*schema.Message](32)
	go func() {
		defer streamCancel()
		defer response.Body.Close()
		defer writer.Close()

		buffered := bufio.NewReader(response.Body)
		var eventData strings.Builder
		transcript := &streamTranscriptBuilder{}
		var finalUsage *openAIUsage
		finalFinishReason := ""
		idleTimeout := modelClient.streamIdleTimeout
		if idleTimeout <= 0 {
			idleTimeout = defaultStreamIdleTimeout
		}
		activity := make(chan struct{}, 1)
		monitorDone := make(chan struct{})
		var idleTimedOut atomic.Bool
		if idleTimeout > 0 {
			go func() {
				timer := time.NewTimer(idleTimeout)
				defer timer.Stop()
				for {
					select {
					case <-monitorDone:
						return
					case <-activity:
						if !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}
						timer.Reset(idleTimeout)
					case <-timer.C:
						idleTimedOut.Store(true)
						streamCancel()
						return
					}
				}
			}()
		}
		defer close(monitorDone)
		markActivity := func() {
			if idleTimeout <= 0 {
				return
			}
			select {
			case activity <- struct{}{}:
			default:
			}
		}

		for {
			select {
			case <-streamCtx.Done():
				if idleTimedOut.Load() {
					record.finishWithError(streamCtx, errStreamIdleTimeout, transcript.JSONPayload())
					writer.Send(nil, errStreamIdleTimeout)
				} else {
					record.finishWithError(streamCtx, streamCtx.Err(), transcript.JSONPayload())
					writer.Send(nil, streamCtx.Err())
				}
				return
			default:
			}

			line, err := buffered.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					if eventData.Len() > 0 {
						payload := eventData.String()
						transcript.Append(strings.TrimSpace(payload))
						finishReason, usage, _ := inspectStreamChunk(payload)
						if finishReason != "" {
							finalFinishReason = finishReason
						}
						if usage != nil {
							finalUsage = usage
						}
						if emitErr := emitStreamChunk(payload, writer); emitErr != nil && emitErr != io.EOF {
							record.finishWithError(streamCtx, emitErr, transcript.JSONPayload())
							writer.Send(nil, emitErr)
							return
						}
					}
					record.finishWithResponse(streamCtx, finalFinishReason, transcript.JSONPayload(), finalUsage)
					return
				}
				if idleTimedOut.Load() {
					record.finishWithError(streamCtx, errStreamIdleTimeout, transcript.JSONPayload())
					writer.Send(nil, errStreamIdleTimeout)
					return
				}
				if streamCtx.Err() != nil {
					record.finishWithError(streamCtx, streamCtx.Err(), transcript.JSONPayload())
					writer.Send(nil, streamCtx.Err())
					return
				}
				record.finishWithError(streamCtx, err, transcript.JSONPayload())
				writer.Send(nil, err)
				return
			}
			markActivity()

			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				if eventData.Len() > 0 {
					payload := eventData.String()
					transcript.Append(strings.TrimSpace(payload))
					finishReason, usage, parseErr := inspectStreamChunk(payload)
					if finishReason != "" {
						finalFinishReason = finishReason
					}
					if usage != nil {
						finalUsage = usage
					}
					if err := emitStreamChunk(payload, writer); err != nil {
						if parseErr != nil {
							record.finishWithError(streamCtx, parseErr, transcript.JSONPayload())
						}
						if err == io.EOF {
							record.finishWithResponse(streamCtx, finalFinishReason, transcript.JSONPayload(), finalUsage)
							return
						}
						record.finishWithError(streamCtx, err, transcript.JSONPayload())
						writer.Send(nil, err)
						return
					}
					eventData.Reset()
				}
				continue
			}

			if strings.HasPrefix(line, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if eventData.Len() > 0 {
					eventData.WriteString("\n")
				}
				eventData.WriteString(payload)
			}
		}
	}()

	return reader, nil
}

func cloneStreamHTTPClient(base *http.Client) *http.Client {
	if base == nil {
		return &http.Client{}
	}
	cloned := *base
	cloned.Timeout = 0
	return &cloned
}

func (modelClient *OpenAICompatibleChatModel) chatURL() string {
	path := modelClient.chatPath
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return modelClient.baseURL + path
}

func toOpenAIMessages(input []*schema.Message) []openAIMessage {
	messages := make([]openAIMessage, 0, len(input))
	for _, message := range input {
		if message == nil {
			continue
		}
		role := strings.TrimSpace(string(message.Role))
		if role == "" {
			role = "user"
		}
		openAIMessage := openAIMessage{
			Role:    role,
			Content: message.Content,
		}
		if message.ReasoningContent != "" {
			openAIMessage.ReasoningContent = message.ReasoningContent
		}
		if message.Role == schema.Tool {
			openAIMessage.ToolCallID = message.ToolCallID
		}
		if message.Role == schema.Assistant && len(message.ToolCalls) > 0 {
			openAIMessage.ToolCalls = toOpenAIToolCalls(message.ToolCalls)
		}
		messages = append(messages, openAIMessage)
	}
	return messages
}

type openAIChatRequest struct {
	Model           string                 `json:"model"`
	Messages        []openAIMessage        `json:"messages"`
	Temperature     *float32               `json:"temperature,omitempty"`
	MaxTokens       *int                   `json:"max_tokens,omitempty"`
	ReasoningEffort string                 `json:"reasoning_effort,omitempty"`
	Reasoning       *openAIReasoning       `json:"reasoning,omitempty"`
	Thinking        *openAIThinkingConfig  `json:"thinking,omitempty"`
	EnableThinking  *bool                  `json:"enable_thinking,omitempty"`
	OutputConfig    *anthropicOutputConfig `json:"output_config,omitempty"`
	ResponseFormat  *openAIResponseFormat  `json:"response_format,omitempty"`
	TopP            *float32               `json:"top_p,omitempty"`
	Stop            []string               `json:"stop,omitempty"`
	Tools           []openAITool           `json:"tools,omitempty"`
	ToolChoice      any                    `json:"tool_choice,omitempty"`
	Stream          bool                   `json:"stream,omitempty"`
	StreamOptions   *openAIStreamOptions   `json:"stream_options,omitempty"`
}

type openAIMessage struct {
	Role             string           `json:"role"`
	Content          string           `json:"content,omitempty"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	Reasoning        string           `json:"reasoning,omitempty"`
	ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string           `json:"tool_call_id,omitempty"`
}

type openAIReasoning struct {
	Effort    string `json:"effort,omitempty"`
	MaxTokens *int   `json:"max_tokens,omitempty"`
}

type openAIThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens *int   `json:"budget_tokens,omitempty"`
}

type anthropicOutputConfig struct {
	Effort string `json:"effort,omitempty"`
}

type openAIResponseFormat struct {
	Type       string                          `json:"type"`
	JSONSchema *openAIResponseFormatJSONSchema `json:"json_schema,omitempty"`
}

type openAIResponseFormatJSONSchema struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict,omitempty"`
}

func applyRuntimeParamsToRequest(ctx context.Context, payload *openAIChatRequest) {
	applyRuntimeParams(runtimeParamsFromContext(ctx), providerRequestCompatibility{}, payload)
}

type providerRequestCompatibility struct {
	ProviderID    string
	ProviderType  domainproviders.ProviderType
	Compatibility domainproviders.ProviderCompatibility
	ModelName     string
}

func applyRuntimeParams(params RuntimeParams, compatibility providerRequestCompatibility, payload *openAIChatRequest) {
	if payload == nil {
		return
	}
	if responseFormat := toOpenAIResponseFormat(params.StructuredOutput); responseFormat != nil {
		payload.ResponseFormat = responseFormat
	}
	applyThinkingLevel(params.ThinkingLevel, compatibility, payload)
}

func toOpenAIResponseFormat(config StructuredOutputConfig) *openAIResponseFormat {
	if !config.UsesJSONSchema() {
		return nil
	}
	return &openAIResponseFormat{
		Type: "json_schema",
		JSONSchema: &openAIResponseFormatJSONSchema{
			Name:   strings.TrimSpace(config.Name),
			Schema: cloneStructuredOutputSchema(config.Schema),
			Strict: config.Strict,
		},
	}
}

func applyThinkingLevel(level string, compatibility providerRequestCompatibility, payload *openAIChatRequest) {
	if payload == nil {
		return
	}
	payload.ReasoningEffort = ""
	payload.Reasoning = nil
	payload.Thinking = nil
	payload.EnableThinking = nil
	payload.OutputConfig = nil

	level = normalizeProviderThinkingLevel(level)
	if level == "" {
		return
	}

	profile, ok := resolveModelReasoningProfile(compatibility)
	if !ok || !reasoningProfileSupportsLevel(profile, level) {
		return
	}

	switch profile.ControlProtocol {
	case ReasoningControlProtocolOpenAIReasoningEffort:
		applyProfileOpenAIReasoningEffort(level, payload)
	case ReasoningControlProtocolOpenRouterReasoning:
		applyProfileOpenRouterReasoning(level, payload)
	case ReasoningControlProtocolThinkingToggle:
		applyProfileThinkingToggle(level, payload)
	case ReasoningControlProtocolAnthropicThinking:
		applyProfileAnthropicThinking(level, payload)
	case ReasoningControlProtocolQwenThinkingToggle:
		applyProfileQwenThinkingToggle(level, payload)
	default:
		return
	}
}

func applyProfileOpenAIReasoningEffort(level string, payload *openAIChatRequest) {
	if payload == nil {
		return
	}
	switch level {
	case "off":
		payload.ReasoningEffort = "none"
	case "minimal", "low", "medium", "high", "xhigh":
		payload.ReasoningEffort = level
	}
}

func applyProfileOpenRouterReasoning(level string, payload *openAIChatRequest) {
	if payload == nil {
		return
	}
	switch level {
	case "off":
		payload.Reasoning = &openAIReasoning{Effort: "none"}
	case "minimal", "low", "medium", "high", "xhigh":
		payload.Reasoning = &openAIReasoning{Effort: level}
	}
}

func applyProfileThinkingToggle(level string, payload *openAIChatRequest) {
	if payload == nil {
		return
	}
	thinkingType := "enabled"
	if level == "off" {
		thinkingType = "disabled"
	}
	payload.Thinking = &openAIThinkingConfig{Type: thinkingType}
}

func applyProfileAnthropicThinking(level string, payload *openAIChatRequest) {
	if payload == nil {
		return
	}
	budget := anthropicThinkingBudget(level, payload.MaxTokens)
	if budget == nil {
		return
	}
	payload.Thinking = &openAIThinkingConfig{
		Type:         "enabled",
		BudgetTokens: budget,
	}
	forceTemperatureOne(payload)
}

func applyProfileQwenThinkingToggle(level string, payload *openAIChatRequest) {
	if payload == nil {
		return
	}
	enabled := level != "off"
	payload.EnableThinking = &enabled
}

func anthropicThinkingBudget(level string, maxTokens *int) *int {
	base := 2048
	switch level {
	case "minimal":
		base = 1024
	case "low":
		base = 2048
	case "medium":
		base = 4096
	case "high":
		base = 8192
	case "xhigh":
		base = 16384
	default:
		return nil
	}
	if maxTokens == nil || *maxTokens <= 0 {
		value := base
		return &value
	}
	limit := *maxTokens - 1
	if limit < 1024 {
		return nil
	}
	if base > limit {
		base = limit
	}
	if base < 1024 {
		base = 1024
	}
	value := base
	return &value
}

func forceTemperatureOne(payload *openAIChatRequest) {
	if payload == nil {
		return
	}
	value := float32(1)
	payload.Temperature = &value
}

func normalizeProviderThinkingLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "off", "none", "disabled", "disable", "false", "0":
		return "off"
	case "minimal", "min":
		return "minimal"
	case "low", "on", "enabled", "enable", "true", "1":
		return "low"
	case "medium", "med":
		return "medium"
	case "high", "max":
		return "high"
	case "xhigh", "ultra":
		return "xhigh"
	default:
		return ""
	}
}

type openAIChatResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   *openAIUsage   `json:"usage"`
}

type openAIChoice struct {
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (modelClient *OpenAICompatibleChatModel) executeChatRequest(ctx context.Context, payload openAIChatRequest) ([]byte, error) {
	payloadJSON := marshalOpenAIChatPayload(payload)
	record := startActiveLLMCallRecord(ctx, modelClient.recorder, runtimeParamsFromContext(ctx), payloadJSON)
	request, err := modelClient.buildChatRequest(ctx, payload)
	if err != nil {
		record.finishWithError(ctx, err, "")
		return nil, err
	}
	response, err := modelClient.client.Do(request)
	if err != nil {
		record.finishWithError(ctx, err, "")
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		record.finishWithError(ctx, err, "")
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		record.finishWithError(ctx, &HTTPStatusError{
			Code:    response.StatusCode,
			Message: extractHTTPErrorMessage(body),
			Body:    strings.TrimSpace(string(body)),
		}, strings.TrimSpace(string(body)))
		return nil, &HTTPStatusError{
			Code:    response.StatusCode,
			Message: extractHTTPErrorMessage(body),
			Body:    strings.TrimSpace(string(body)),
		}
	}
	finishReason, usage, inspectErr := inspectChatResponse(body)
	if inspectErr != nil {
		record.finishWithError(ctx, inspectErr, strings.TrimSpace(string(body)))
	} else if finishReason != "" || usage != nil {
		record.finishWithResponse(ctx, finishReason, strings.TrimSpace(string(body)), usage)
	} else {
		record.finishWithResponse(ctx, "", strings.TrimSpace(string(body)), nil)
	}
	return body, nil
}

func (modelClient *OpenAICompatibleChatModel) executeStreamRequest(
	ctx context.Context,
	client *http.Client,
	payload openAIChatRequest,
) (*http.Response, *activeLLMCallRecord, error) {
	payloadJSON := marshalOpenAIChatPayload(payload)
	record := startActiveLLMCallRecord(ctx, modelClient.recorder, runtimeParamsFromContext(ctx), payloadJSON)
	request, err := modelClient.buildChatRequest(ctx, payload)
	if err != nil {
		record.finishWithError(ctx, err, "")
		return nil, nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		record.finishWithError(ctx, err, "")
		return nil, nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		statusErr := &HTTPStatusError{
			Code:    response.StatusCode,
			Message: extractHTTPErrorMessage(body),
			Body:    strings.TrimSpace(string(body)),
		}
		record.finishWithError(ctx, statusErr, strings.TrimSpace(string(body)))
		return nil, nil, statusErr
	}
	return response, record, nil
}

func (modelClient *OpenAICompatibleChatModel) buildChatRequest(ctx context.Context, payload openAIChatRequest) (*http.Request, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, modelClient.chatURL(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if modelClient.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+modelClient.apiKey)
	}
	for key, value := range modelClient.headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		request.Header.Set(key, value)
	}
	return request, nil
}

func marshalOpenAIChatPayload(payload openAIChatRequest) string {
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(data)
}

func inspectChatResponse(body []byte) (string, *openAIUsage, error) {
	if len(body) == 0 {
		return "", nil, nil
	}
	var decoded openAIChatResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", nil, err
	}
	finishReason := ""
	if len(decoded.Choices) > 0 {
		finishReason = strings.TrimSpace(decoded.Choices[0].FinishReason)
	}
	return finishReason, decoded.Usage, nil
}

func inspectStreamChunk(payload string) (string, *openAIUsage, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" || payload == "[DONE]" {
		return "", nil, nil
	}
	var decoded openAIChatStreamResponse
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return "", nil, err
	}
	finishReason := ""
	for _, choice := range decoded.Choices {
		if value := strings.TrimSpace(choice.FinishReason); value != "" {
			finishReason = value
			break
		}
	}
	return finishReason, decoded.Usage, nil
}

func shouldRetryWithoutStructuredOutput(err error, config StructuredOutputConfig, responseFormat *openAIResponseFormat) bool {
	if responseFormat == nil || !config.AllowsFallback() {
		return false
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr == nil {
		return false
	}
	if statusErr.Code < http.StatusBadRequest || statusErr.Code >= http.StatusInternalServerError {
		return false
	}
	combined := strings.ToLower(strings.TrimSpace(statusErr.SafeMessage() + " " + statusErr.Body))
	if combined == "" {
		return false
	}
	mentionsStructuredOutput := strings.Contains(combined, "response_format") ||
		strings.Contains(combined, "json_schema") ||
		strings.Contains(combined, "json schema")
	if !mentionsStructuredOutput {
		return false
	}
	return strings.Contains(combined, "unsupported") ||
		strings.Contains(combined, "not support") ||
		strings.Contains(combined, "unknown") ||
		strings.Contains(combined, "invalid")
}

type openAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type openAIChatStreamResponse struct {
	Choices []openAIStreamChoice `json:"choices"`
	Usage   *openAIUsage         `json:"usage,omitempty"`
}

type openAIStreamChoice struct {
	Delta        openAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason"`
	Index        int               `json:"index"`
}

type openAIStreamDelta struct {
	Role             string                 `json:"role,omitempty"`
	Content          string                 `json:"content,omitempty"`
	ReasoningContent string                 `json:"reasoning_content,omitempty"`
	Reasoning        string                 `json:"reasoning,omitempty"`
	ToolCalls        []openAIStreamToolCall `json:"tool_calls,omitempty"`
}

type openAIStreamToolCall struct {
	Index    int                    `json:"index,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Type     string                 `json:"type,omitempty"`
	Function openAIToolFunctionCall `json:"function,omitempty"`
}

type openAITool struct {
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	ID       string                 `json:"id,omitempty"`
	Type     string                 `json:"type,omitempty"`
	Function openAIToolFunctionCall `json:"function"`
}

type openAIToolFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

func toOpenAIToolCalls(calls []schema.ToolCall) []openAIToolCall {
	if len(calls) == 0 {
		return nil
	}
	result := make([]openAIToolCall, 0, len(calls))
	for _, call := range calls {
		callType := strings.TrimSpace(call.Type)
		if callType == "" {
			callType = "function"
		}
		result = append(result, openAIToolCall{
			ID:   call.ID,
			Type: callType,
			Function: openAIToolFunctionCall{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			},
		})
	}
	return result
}

func toSchemaToolCalls(calls []openAIToolCall) []schema.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	result := make([]schema.ToolCall, 0, len(calls))
	for _, call := range calls {
		result = append(result, schema.ToolCall{
			ID:   call.ID,
			Type: call.Type,
			Function: schema.FunctionCall{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			},
		})
	}
	return result
}

func toSchemaToolCallDeltas(calls []openAIStreamToolCall) []schema.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	result := make([]schema.ToolCall, 0, len(calls))
	for _, call := range calls {
		index := call.Index
		result = append(result, schema.ToolCall{
			Index: &index,
			ID:    call.ID,
			Type:  call.Type,
			Function: schema.FunctionCall{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			},
		})
	}
	return result
}

func toOpenAITools(tools []*schema.ToolInfo, allowed []string) ([]openAITool, error) {
	if len(tools) == 0 {
		return nil, nil
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		allowedSet[name] = struct{}{}
	}

	openAITools := make([]openAITool, 0, len(tools))
	for _, tool := range tools {
		if tool == nil || strings.TrimSpace(tool.Name) == "" {
			continue
		}
		if len(allowedSet) > 0 {
			if _, ok := allowedSet[tool.Name]; !ok {
				continue
			}
		}
		var params any
		if tool.ParamsOneOf != nil {
			schemaDef, err := tool.ParamsOneOf.ToJSONSchema()
			if err != nil {
				return nil, err
			}
			params = schemaDef
		}
		openAITools = append(openAITools, openAITool{
			Type: "function",
			Function: openAIToolFunction{
				Name:        tool.Name,
				Description: tool.Desc,
				Parameters:  params,
			},
		})
	}
	return openAITools, nil
}

func toOpenAIToolChoice(choice *schema.ToolChoice, allowed []string) any {
	if choice == nil {
		return nil
	}
	switch *choice {
	case schema.ToolChoiceForbidden:
		return "none"
	case schema.ToolChoiceForced:
		if len(allowed) == 1 {
			return map[string]any{
				"type": "function",
				"function": map[string]string{
					"name": allowed[0],
				},
			}
		}
		return "required"
	default:
		return "auto"
	}
}

func emitStreamChunk(payload string, writer *schema.StreamWriter[*schema.Message]) error {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil
	}
	if payload == "[DONE]" {
		return io.EOF
	}

	var decoded openAIChatStreamResponse
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return err
	}

	finishReason := ""
	for _, choice := range decoded.Choices {
		if choice.Delta.Content != "" {
			writer.Send(&schema.Message{
				Role:    schema.Assistant,
				Content: choice.Delta.Content,
			}, nil)
		}
		if choice.Delta.ReasoningContent != "" || choice.Delta.Reasoning != "" {
			reasoning := resolveReasoningDelta(choice.Delta.ReasoningContent, choice.Delta.Reasoning)
			if reasoning != "" {
				writer.Send(&schema.Message{
					Role:             schema.Assistant,
					ReasoningContent: reasoning,
				}, nil)
			}
		}
		if len(choice.Delta.ToolCalls) > 0 {
			writer.Send(&schema.Message{
				Role:      schema.Assistant,
				ToolCalls: toSchemaToolCallDeltas(choice.Delta.ToolCalls),
			}, nil)
		}
		if choice.FinishReason != "" {
			finishReason = choice.FinishReason
		}
	}
	if finishReason != "" || decoded.Usage != nil {
		meta := &schema.ResponseMeta{
			FinishReason: finishReason,
		}
		if decoded.Usage != nil {
			meta.Usage = &schema.TokenUsage{
				PromptTokens:     decoded.Usage.PromptTokens,
				CompletionTokens: decoded.Usage.CompletionTokens,
				TotalTokens:      decoded.Usage.TotalTokens,
			}
		}
		writer.Send(&schema.Message{
			Role:         schema.Assistant,
			ResponseMeta: meta,
		}, nil)
	}

	return nil
}

func resolveReasoningDelta(primary string, secondary string) string {
	if primary != "" {
		if strings.TrimSpace(primary) != "" || secondary == "" {
			return primary
		}
	}
	return secondary
}

package runtime

import (
	"context"
	"errors"
	"io"
	"math"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"dreamcreator/internal/application/agentruntime"
)

const (
	summaryBaseChunkRatio   = 0.4
	summaryMinChunkRatio    = 0.15
	summarySafetyMargin     = 1.2
	summaryDefaultMaxTokens = 1024
	summaryRetryAttempts    = 3
	summaryRetryBaseDelay   = 500 * time.Millisecond
	summaryRetryMaxDelay    = 5 * time.Second
	summaryFallbackText     = "Context contained large messages. Summary unavailable due to size limits."
)

const summaryMergeInstructions = "Merge these partial summaries into a single cohesive summary. Preserve decisions, TODOs, open questions, and constraints."

func (service *Service) summarizeForCompaction(ctx context.Context, chatModel model.BaseChatModel, params compactionSummaryParams) (string, error) {
	if chatModel == nil {
		return "", errors.New("chat model unavailable")
	}
	if len(params.Messages) == 0 {
		return strings.TrimSpace(params.PreviousSummary), nil
	}
	safeMessages := stripToolResultDetailsFromMessages(cloneSchemaMessages(params.Messages))
	contextWindow := params.ContextWindowTokens
	if contextWindow <= 0 {
		contextWindow = defaultContextWindowTokens
	}
	reserveTokens := params.ReserveTokens
	maxChunkTokens := resolveSummaryChunkTokens(safeMessages, contextWindow, reserveTokens)
	totalTokens := agentruntime.EstimateMessagesTokensSafe(safeMessages)
	if len(safeMessages) >= 4 && maxChunkTokens > 0 && totalTokens > maxChunkTokens {
		return summarizeInStages(ctx, chatModel, compactionSummarizeStageParams{
			Messages:           safeMessages,
			PreviousSummary:    strings.TrimSpace(params.PreviousSummary),
			CustomInstructions: strings.TrimSpace(params.CustomInstructions),
			ContextWindow:      contextWindow,
			ReserveTokens:      reserveTokens,
			MaxChunkTokens:     maxChunkTokens,
			Parts:              2,
		})
	}
	return summarizeWithFallback(ctx, chatModel, compactionFallbackParams{
		Messages:           safeMessages,
		PreviousSummary:    strings.TrimSpace(params.PreviousSummary),
		CustomInstructions: strings.TrimSpace(params.CustomInstructions),
		ContextWindow:      contextWindow,
		ReserveTokens:      reserveTokens,
		MaxChunkTokens:     maxChunkTokens,
	})
}

type compactionFallbackParams struct {
	Messages           []*schema.Message
	PreviousSummary    string
	CustomInstructions string
	ContextWindow      int
	ReserveTokens      int
	MaxChunkTokens     int
}

type compactionSummarizeStageParams struct {
	Messages           []*schema.Message
	PreviousSummary    string
	CustomInstructions string
	ContextWindow      int
	ReserveTokens      int
	MaxChunkTokens     int
	Parts              int
}

func summarizeInStages(ctx context.Context, chatModel model.BaseChatModel, params compactionSummarizeStageParams) (string, error) {
	if len(params.Messages) == 0 {
		return strings.TrimSpace(params.PreviousSummary), nil
	}
	parts := params.Parts
	if parts <= 1 {
		parts = 2
	}
	splits := splitMessagesByTokenShare(params.Messages, parts)
	if len(splits) <= 1 {
		return summarizeWithFallback(ctx, chatModel, compactionFallbackParams{
			Messages:           params.Messages,
			PreviousSummary:    params.PreviousSummary,
			CustomInstructions: params.CustomInstructions,
			ContextWindow:      params.ContextWindow,
			ReserveTokens:      params.ReserveTokens,
			MaxChunkTokens:     params.MaxChunkTokens,
		})
	}
	partials := make([]string, 0, len(splits))
	for _, split := range splits {
		summary, err := summarizeWithFallback(ctx, chatModel, compactionFallbackParams{
			Messages:           split,
			PreviousSummary:    "",
			CustomInstructions: params.CustomInstructions,
			ContextWindow:      params.ContextWindow,
			ReserveTokens:      params.ReserveTokens,
			MaxChunkTokens:     params.MaxChunkTokens,
		})
		if err != nil {
			return "", err
		}
		partials = append(partials, strings.TrimSpace(summary))
	}
	if len(partials) == 0 {
		return strings.TrimSpace(params.PreviousSummary), nil
	}
	if len(partials) == 1 {
		return partials[0], nil
	}
	mergeMessages := make([]*schema.Message, 0, len(partials))
	for _, partial := range partials {
		if strings.TrimSpace(partial) == "" {
			continue
		}
		mergeMessages = append(mergeMessages, &schema.Message{
			Role:    schema.User,
			Content: partial,
		})
	}
	mergeInstructions := summaryMergeInstructions
	if strings.TrimSpace(params.CustomInstructions) != "" {
		mergeInstructions += "\n\nAdditional focus:\n" + strings.TrimSpace(params.CustomInstructions)
	}
	return summarizeWithFallback(ctx, chatModel, compactionFallbackParams{
		Messages:           mergeMessages,
		PreviousSummary:    strings.TrimSpace(params.PreviousSummary),
		CustomInstructions: mergeInstructions,
		ContextWindow:      params.ContextWindow,
		ReserveTokens:      params.ReserveTokens,
		MaxChunkTokens:     params.MaxChunkTokens,
	})
}

func summarizeWithFallback(ctx context.Context, chatModel model.BaseChatModel, params compactionFallbackParams) (string, error) {
	if len(params.Messages) == 0 {
		return strings.TrimSpace(params.PreviousSummary), nil
	}
	summary, err := summarizeChunks(ctx, chatModel, params.Messages, params.PreviousSummary, params.CustomInstructions, params.ReserveTokens, params.ContextWindow, params.MaxChunkTokens)
	if err == nil {
		return summary, nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "", err
	}

	smallMessages := make([]*schema.Message, 0, len(params.Messages))
	oversizedNotes := make([]string, 0, 4)
	for _, message := range params.Messages {
		if isOversizedForSummary(message, params.ContextWindow) {
			oversizedNotes = append(oversizedNotes, "[Large message omitted from summary]")
			continue
		}
		smallMessages = append(smallMessages, message)
	}
	if len(smallMessages) > 0 {
		partial, partialErr := summarizeChunks(ctx, chatModel, smallMessages, params.PreviousSummary, params.CustomInstructions, params.ReserveTokens, params.ContextWindow, params.MaxChunkTokens)
		if partialErr == nil {
			if len(oversizedNotes) == 0 {
				return partial, nil
			}
			return strings.TrimSpace(partial + "\n\n" + strings.Join(oversizedNotes, "\n")), nil
		}
		if errors.Is(partialErr, context.DeadlineExceeded) || errors.Is(partialErr, context.Canceled) {
			return "", partialErr
		}
	}
	if strings.TrimSpace(params.PreviousSummary) != "" {
		return strings.TrimSpace(params.PreviousSummary), nil
	}
	return summaryFallbackText, nil
}

func summarizeChunks(
	ctx context.Context,
	chatModel model.BaseChatModel,
	messages []*schema.Message,
	previousSummary string,
	custom string,
	reserveTokens int,
	contextWindow int,
	maxChunkTokens int,
) (string, error) {
	if len(messages) == 0 {
		return strings.TrimSpace(previousSummary), nil
	}
	chunkTokens := maxChunkTokens
	if chunkTokens <= 0 {
		chunkTokens = resolveSummaryChunkTokens(messages, contextWindow, reserveTokens)
	}
	chunks := chunkMessagesByMaxTokens(messages, chunkTokens)
	if len(chunks) == 0 {
		return strings.TrimSpace(previousSummary), nil
	}
	summary := strings.TrimSpace(previousSummary)
	for _, chunk := range chunks {
		next, err := summarizeChunkWithRetry(ctx, chatModel, chunk, summary, custom, reserveTokens, contextWindow)
		if err != nil {
			return "", err
		}
		summary = strings.TrimSpace(next)
	}
	if summary == "" {
		return strings.TrimSpace(previousSummary), nil
	}
	return summary, nil
}

func summarizeChunkWithRetry(
	ctx context.Context,
	chatModel model.BaseChatModel,
	messages []*schema.Message,
	previousSummary string,
	custom string,
	reserveTokens int,
	contextWindow int,
) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= summaryRetryAttempts; attempt++ {
		next, err := summarizeChunkOnce(ctx, chatModel, messages, previousSummary, custom, reserveTokens, contextWindow)
		if err == nil {
			return next, nil
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return "", err
		}
		lastErr = err
		if attempt >= summaryRetryAttempts {
			break
		}
		delay := summaryRetryBaseDelay * time.Duration(1<<(attempt-1))
		if delay > summaryRetryMaxDelay {
			delay = summaryRetryMaxDelay
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", errors.New("summary generation failed")
}

func summarizeChunkOnce(ctx context.Context, chatModel model.BaseChatModel, messages []*schema.Message, previousSummary string, custom string, reserveTokens int, contextWindow int) (string, error) {
	if len(messages) == 0 {
		return strings.TrimSpace(previousSummary), nil
	}
	sys := "Summarize the conversation for later context. Preserve decisions, TODOs, open questions, constraints, and key facts. Be concise."
	user := buildSummaryUserContent(messages, previousSummary, custom)
	request := []*schema.Message{
		{Role: schema.System, Content: sys},
		{Role: schema.User, Content: user},
	}
	maxTokens := resolveSummaryMaxTokens(reserveTokens, contextWindow)
	options := []model.Option{
		model.WithTemperature(0.2),
		model.WithMaxTokens(maxTokens),
	}
	stream, err := chatModel.Stream(ctx, request, options...)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	content, err := collectStreamContent(stream)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(content), nil
}

func isOversizedForSummary(message *schema.Message, contextWindow int) bool {
	if message == nil || contextWindow <= 0 {
		return false
	}
	tokens := agentruntime.EstimateMessageTokensSafe(message)
	return tokens > int(float64(contextWindow)*0.5)
}

func buildSummaryUserContent(messages []*schema.Message, previousSummary string, custom string) string {
	var builder strings.Builder
	if strings.TrimSpace(previousSummary) != "" {
		builder.WriteString("Previous summary:\n")
		builder.WriteString(strings.TrimSpace(previousSummary))
		builder.WriteString("\n\n")
	}
	if strings.TrimSpace(custom) != "" {
		builder.WriteString("Additional focus:\n")
		builder.WriteString(strings.TrimSpace(custom))
		builder.WriteString("\n\n")
	}
	builder.WriteString("Messages:\n")
	for _, message := range messages {
		if message == nil {
			continue
		}
		role := strings.TrimSpace(string(message.Role))
		content := strings.TrimSpace(message.Content)
		if content == "" {
			content = strings.TrimSpace(message.ReasoningContent)
		}
		if content == "" {
			continue
		}
		if role == "" {
			role = "message"
		}
		builder.WriteString("- ")
		builder.WriteString(role)
		builder.WriteString(": ")
		builder.WriteString(content)
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func resolveSummaryChunkTokens(messages []*schema.Message, contextWindow int, reserveTokens int) int {
	if contextWindow <= 0 {
		return 0
	}
	ratio := computeAdaptiveChunkRatio(messages, contextWindow)
	maxTokens := int(math.Floor(float64(contextWindow) * ratio))
	if reserveTokens > 0 && maxTokens > contextWindow-reserveTokens {
		maxTokens = contextWindow - reserveTokens
	}
	if maxTokens <= 0 {
		return contextWindow
	}
	return maxTokens
}

func computeAdaptiveChunkRatio(messages []*schema.Message, contextWindow int) float64 {
	if len(messages) == 0 || contextWindow <= 0 {
		return summaryBaseChunkRatio
	}
	totalTokens := agentruntime.EstimateMessagesTokens(messages)
	if totalTokens <= 0 {
		return summaryBaseChunkRatio
	}
	avgTokens := float64(totalTokens) / float64(len(messages))
	safeAvg := avgTokens * summarySafetyMargin
	avgRatio := safeAvg / float64(contextWindow)
	if avgRatio > 0.1 {
		reduction := avgRatio * 2
		maxReduction := summaryBaseChunkRatio - summaryMinChunkRatio
		if reduction > maxReduction {
			reduction = maxReduction
		}
		result := summaryBaseChunkRatio - reduction
		if result < summaryMinChunkRatio {
			return summaryMinChunkRatio
		}
		return result
	}
	return summaryBaseChunkRatio
}

func resolveSummaryMaxTokens(reserveTokens int, contextWindow int) int {
	if reserveTokens <= 0 {
		if contextWindow > 0 {
			return clampInt(contextWindow/8, 256, 2048)
		}
		return summaryDefaultMaxTokens
	}
	maxTokens := reserveTokens / 4
	if contextWindow > 0 && maxTokens > contextWindow/4 {
		maxTokens = contextWindow / 4
	}
	return clampInt(maxTokens, 256, 4096)
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func chunkMessagesByMaxTokens(messages []*schema.Message, maxTokens int) [][]*schema.Message {
	if len(messages) == 0 {
		return nil
	}
	if maxTokens <= 0 {
		return [][]*schema.Message{messages}
	}
	chunks := make([][]*schema.Message, 0)
	current := make([]*schema.Message, 0)
	currentTokens := 0
	for _, message := range messages {
		tokens := agentruntime.EstimateMessageTokensSafe(message)
		if len(current) > 0 && currentTokens+tokens > maxTokens {
			chunks = append(chunks, current)
			current = make([]*schema.Message, 0)
			currentTokens = 0
		}
		current = append(current, message)
		currentTokens += tokens
		if tokens > maxTokens {
			chunks = append(chunks, current)
			current = make([]*schema.Message, 0)
			currentTokens = 0
		}
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

func collectStreamContent(stream *schema.StreamReader[*schema.Message]) (string, error) {
	if stream == nil {
		return "", errors.New("stream unavailable")
	}
	var builder strings.Builder
	for {
		message, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
		if message == nil {
			continue
		}
		if message.Content != "" {
			builder.WriteString(message.Content)
		}
	}
	return strings.TrimSpace(builder.String()), nil
}

func (service *Service) runMemoryFlush(
	ctx context.Context,
	params memoryFlushParams,
	streamFn agentruntime.StreamFunction,
	tools map[string]agentruntime.ToolDefinition,
) (string, error) {
	if streamFn == nil {
		return "", errors.New("memory flush model unavailable")
	}
	messages := cloneSchemaMessages(params.Messages)
	if systemPrompt := strings.TrimSpace(params.SystemPrompt); systemPrompt != "" {
		messages = append(messages, &schema.Message{Role: schema.System, Content: systemPrompt})
	}
	if userPrompt := strings.TrimSpace(params.UserPrompt); userPrompt != "" {
		messages = append(messages, &schema.Message{Role: schema.User, Content: userPrompt})
	}
	maxSteps := params.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 4
	}
	loop := &agentruntime.AgentLoop{
		StreamFunction: streamFn,
		ToolExecutor: &agentruntime.ToolExecutor{
			Validator: agentruntime.JSONToolValidator{},
			Tools:     tools,
		},
		MaxSteps: maxSteps,
	}
	stream, err := loop.RunStream(ctx, agentruntime.AgentState{
		Messages:    messages,
		IsStreaming: false,
	})
	if err != nil {
		return "", err
	}
	content, _, _, _, err := consumeAgentLoopStream(stream, nil)
	if err != nil {
		if strings.Contains(err.Error(), "assistant response is empty") {
			return "", nil
		}
		return "", err
	}
	if isNoReplyResponse(content) {
		return "", nil
	}
	return strings.TrimSpace(content), nil
}

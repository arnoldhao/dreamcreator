package runtime

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/schema"

	"dreamcreator/internal/application/chatevent"
	gatewayprompt "dreamcreator/internal/application/gateway/prompt"
	skillsdto "dreamcreator/internal/application/skills/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
	"dreamcreator/internal/domain/thread"
)

type promptReportPayload struct {
	RunID            string                       `json:"runId"`
	Mode             string                       `json:"mode"`
	Prompt           string                       `json:"prompt,omitempty"`
	PromptChars      int                          `json:"promptChars,omitempty"`
	Messages         []promptReportMessagePayload `json:"messages,omitempty"`
	Context          *promptContextReportPayload  `json:"context,omitempty"`
	SectionLabels    map[string]string            `json:"sectionLabels,omitempty"`
	SectionsDetailed []promptReportSectionPayload `json:"sectionsDetailed,omitempty"`
	Report           gatewayprompt.BuildReport    `json:"report"`
	Tools            []string                     `json:"tools,omitempty"`
	Skills           []string                     `json:"skills,omitempty"`
}

type promptReportMessagePayload struct {
	Role       string `json:"role"`
	Content    string `json:"content,omitempty"`
	Reasoning  string `json:"reasoning,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
}

type promptReportSectionPayload struct {
	ID        string `json:"id"`
	Label     string `json:"label,omitempty"`
	Content   string `json:"content,omitempty"`
	Tokens    int    `json:"tokens,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type promptContextReportPayload struct {
	Source                       string `json:"source,omitempty"`
	StoredMessageCount           int    `json:"storedMessageCount,omitempty"`
	InputMessageCount            int    `json:"inputMessageCount,omitempty"`
	BuiltMessageCount            int    `json:"builtMessageCount,omitempty"`
	UsedPersistedSummary         bool   `json:"usedPersistedSummary,omitempty"`
	ClearedStalePersistedSummary bool   `json:"clearedStalePersistedSummary,omitempty"`
	PersistedSummaryChars        int    `json:"persistedSummaryChars,omitempty"`
	PersistedFirstKeptMessageID  string `json:"persistedFirstKeptMessageId,omitempty"`
	BudgetApplied                bool   `json:"budgetApplied,omitempty"`
	ContextWindowTokens          int    `json:"contextWindowTokens,omitempty"`
	ReserveTokens                int    `json:"reserveTokens,omitempty"`
	ExtraTokens                  int    `json:"extraTokens,omitempty"`
	AvailablePromptTokens        int    `json:"availablePromptTokens,omitempty"`
	InitialEstimatedTokens       int    `json:"initialEstimatedTokens,omitempty"`
	FinalEstimatedTokens         int    `json:"finalEstimatedTokens,omitempty"`
}

func (service *Service) emitPromptReport(
	ctx context.Context,
	run thread.ThreadRun,
	sessionKey string,
	mode string,
	recordPrompt bool,
	document gatewayprompt.Document,
	report gatewayprompt.BuildReport,
	sections []gatewayprompt.Section,
	contextReport promptContextBuildReport,
	messages []*schema.Message,
	tools []tooldto.ToolSpec,
	skills []skillsdto.SkillPromptItem,
) {
	if service == nil {
		return
	}
	labels := make(map[string]string, len(sections))
	sectionReports := make(map[string]gatewayprompt.SectionReport, len(report.Sections))
	for _, item := range report.Sections {
		key := strings.TrimSpace(item.ID)
		if key == "" {
			continue
		}
		sectionReports[key] = item
	}
	detailed := make([]promptReportSectionPayload, 0, len(sections))
	for _, section := range sections {
		id := strings.TrimSpace(section.ID)
		if id == "" {
			continue
		}
		labels[id] = strings.TrimSpace(section.Label)
		if !recordPrompt {
			continue
		}
		entry := promptReportSectionPayload{
			ID:      id,
			Label:   strings.TrimSpace(section.Label),
			Content: strings.TrimSpace(section.Content),
		}
		if reportItem, ok := sectionReports[id]; ok {
			entry.Tokens = reportItem.Tokens
			entry.Truncated = reportItem.Truncated
			entry.Reason = strings.TrimSpace(reportItem.Reason)
		}
		detailed = append(detailed, entry)
	}
	payload := promptReportPayload{
		RunID:         strings.TrimSpace(run.ID),
		Mode:          strings.TrimSpace(mode),
		SectionLabels: labels,
		Report:        report,
		Tools:         collectToolNames(tools),
		Skills:        collectSkillNames(skills),
	}
	if contextPayload := buildPromptContextReportPayload(contextReport); contextPayload != nil {
		payload.Context = contextPayload
	}
	if recordPrompt {
		promptText := strings.TrimSpace(document.Content)
		payload.Prompt = promptText
		payload.PromptChars = len([]rune(promptText))
		payload.Messages = collectPromptMessages(promptText, messages)
		payload.SectionsDetailed = detailed
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	service.appendRunEvent(ctx, run, sessionKey, chatevent.Event{
		Type: "prompt.report",
		Data: data,
	})
}

func buildPromptContextReportPayload(report promptContextBuildReport) *promptContextReportPayload {
	if report.Source == "" &&
		report.StoredMessageCount == 0 &&
		report.InputMessageCount == 0 &&
		report.BuiltMessageCount == 0 &&
		!report.UsedPersistedSummary &&
		!report.ClearedStalePersistedSummary &&
		report.PersistedSummaryChars == 0 &&
		report.PersistedFirstKeptMessageID == "" &&
		!report.BudgetApplied &&
		report.ContextWindowTokens == 0 &&
		report.ReserveTokens == 0 &&
		report.ExtraTokens == 0 &&
		report.AvailablePromptTokens == 0 &&
		report.InitialEstimatedTokens == 0 &&
		report.FinalEstimatedTokens == 0 {
		return nil
	}
	return &promptContextReportPayload{
		Source:                       strings.TrimSpace(report.Source),
		StoredMessageCount:           report.StoredMessageCount,
		InputMessageCount:            report.InputMessageCount,
		BuiltMessageCount:            report.BuiltMessageCount,
		UsedPersistedSummary:         report.UsedPersistedSummary,
		ClearedStalePersistedSummary: report.ClearedStalePersistedSummary,
		PersistedSummaryChars:        report.PersistedSummaryChars,
		PersistedFirstKeptMessageID:  strings.TrimSpace(report.PersistedFirstKeptMessageID),
		BudgetApplied:                report.BudgetApplied,
		ContextWindowTokens:          report.ContextWindowTokens,
		ReserveTokens:                report.ReserveTokens,
		ExtraTokens:                  report.ExtraTokens,
		AvailablePromptTokens:        report.AvailablePromptTokens,
		InitialEstimatedTokens:       report.InitialEstimatedTokens,
		FinalEstimatedTokens:         report.FinalEstimatedTokens,
	}
}

func collectPromptMessages(systemPrompt string, messages []*schema.Message) []promptReportMessagePayload {
	result := make([]promptReportMessagePayload, 0, len(messages)+1)
	systemPrompt = strings.TrimSpace(systemPrompt)
	if systemPrompt != "" {
		result = append(result, promptReportMessagePayload{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, message := range messages {
		if message == nil {
			continue
		}
		role := strings.TrimSpace(string(message.Role))
		if role == "" {
			role = "user"
		}
		content := strings.TrimSpace(message.Content)
		reasoning := strings.TrimSpace(message.ReasoningContent)
		if content == "" && role == "assistant" && len(message.ToolCalls) > 0 {
			if encoded, err := json.Marshal(message.ToolCalls); err == nil {
				content = strings.TrimSpace(string(encoded))
			}
		}

		entry := promptReportMessagePayload{
			Role:       role,
			Content:    content,
			Reasoning:  reasoning,
			ToolCallID: strings.TrimSpace(message.ToolCallID),
		}
		if entry.Content == "" && entry.Reasoning == "" && entry.ToolCallID == "" {
			continue
		}
		result = append(result, entry)
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func collectToolNames(items []tooldto.ToolSpec) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		result = append(result, name)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func collectSkillNames(items []skillsdto.SkillPromptItem) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		result = append(result, name)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

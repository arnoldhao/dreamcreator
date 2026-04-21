package runtime

import "strings"

func applyPromptContextMetadata(metadata map[string]any, report promptContextBuildReport) {
	if metadata == nil {
		return
	}
	if source := strings.TrimSpace(report.Source); source != "" {
		metadata["promptContextSource"] = source
	}
	if report.StoredMessageCount > 0 {
		metadata["promptContextStoredMessages"] = report.StoredMessageCount
	}
	if report.InputMessageCount > 0 {
		metadata["promptContextInputMessages"] = report.InputMessageCount
	}
	if report.BuiltMessageCount > 0 {
		metadata["promptContextBuiltMessages"] = report.BuiltMessageCount
	}
	if report.InitialEstimatedTokens > 0 {
		metadata["promptContextInitialTokens"] = report.InitialEstimatedTokens
	}
	if report.FinalEstimatedTokens > 0 {
		metadata["promptContextFinalTokens"] = report.FinalEstimatedTokens
	}
	if report.ContextWindowTokens > 0 {
		metadata["promptContextWindowTokens"] = report.ContextWindowTokens
	}
	if report.AvailablePromptTokens > 0 {
		metadata["promptContextBudgetTokens"] = report.AvailablePromptTokens
	}
	if report.ReserveTokens > 0 {
		metadata["promptContextReserveTokens"] = report.ReserveTokens
	}
	if report.ExtraTokens > 0 {
		metadata["promptContextExtraTokens"] = report.ExtraTokens
	}
	if report.UsedPersistedSummary {
		metadata["persistedSummaryApplied"] = true
	}
	if report.ClearedStalePersistedSummary {
		metadata["persistedSummaryCleared"] = true
	}
	if report.PersistedSummaryChars > 0 {
		metadata["persistedSummaryChars"] = report.PersistedSummaryChars
	}
	if firstKept := strings.TrimSpace(report.PersistedFirstKeptMessageID); firstKept != "" {
		metadata["persistedSummaryFirstKeptMessageID"] = firstKept
	}
	if report.BudgetApplied {
		metadata["promptContextBudgetApplied"] = true
	}
}

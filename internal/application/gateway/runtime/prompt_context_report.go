package runtime

type promptContextBuildReport struct {
	Source                       string
	StoredMessageCount           int
	InputMessageCount            int
	BuiltMessageCount            int
	UsedPersistedSummary         bool
	ClearedStalePersistedSummary bool
	PersistedSummaryChars        int
	PersistedFirstKeptMessageID  string
	BudgetApplied                bool
	ContextWindowTokens          int
	ReserveTokens                int
	ExtraTokens                  int
	AvailablePromptTokens        int
	InitialEstimatedTokens       int
	FinalEstimatedTokens         int
}

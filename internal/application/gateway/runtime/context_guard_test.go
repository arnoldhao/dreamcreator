package runtime

import (
	"context"
	"strings"
	"testing"

	"dreamcreator/internal/application/agentruntime"
	"github.com/cloudwego/eino/schema"
)

func TestTruncateToolResultMessage_UsesUnifiedToolLimit(t *testing.T) {
	t.Parallel()

	message := schema.ToolMessage(strings.Repeat("x", minToolResultKeepChars+2000), "call-1", schema.WithToolName("browser"))

	next, changed := truncateToolResultMessage(message, minToolResultKeepChars)

	if !changed {
		t.Fatal("expected browser tool result to be truncated")
	}
	if len(next.Content) >= len(message.Content) {
		t.Fatalf("expected truncated content to be shorter, got %d >= %d", len(next.Content), len(message.Content))
	}
	if !strings.Contains(next.Content, toolResultTruncationNotice) {
		t.Fatalf("expected truncation notice, got %q", next.Content)
	}
}

func TestApplyContextGuard_ReusesPreviousSummaryForIncrementalCompaction(t *testing.T) {
	t.Parallel()

	var previousSummary string
	state, report, err := applyContextGuard(context.Background(), agentruntime.AgentState{
		Messages: []*schema.Message{
			{Role: schema.System, Content: compactionSummaryPrefix + "older summary"},
			{Role: schema.User, Content: strings.Repeat("a", 120)},
			{Role: schema.Assistant, Content: strings.Repeat("b", 120)},
			{Role: schema.User, Content: "latest turn"},
		},
	}, contextGuardConfig{
		contextWindowTokens: 40,
		reserveTokens:       0,
		keepRecentTokens:    10,
		compactionMode:      "safeguard",
	}, &contextGuardState{}, contextGuardHooks{
		Summarize: func(_ context.Context, params compactionSummaryParams) (string, error) {
			previousSummary = params.PreviousSummary
			return "merged summary", nil
		},
	})
	if err != nil {
		t.Fatalf("apply context guard: %v", err)
	}
	if previousSummary != "older summary" {
		t.Fatalf("expected previous summary to be reused, got %q", previousSummary)
	}
	if report.compactionSummary != "merged summary" {
		t.Fatalf("expected merged summary report, got %q", report.compactionSummary)
	}
	if len(state.Messages) == 0 || state.Messages[0].Role != schema.System {
		t.Fatalf("expected compacted state to retain system summary, got %+v", state.Messages)
	}
}

func TestApplyContextGuard_SupersedesOlderBrowserStatesWithinTurn(t *testing.T) {
	t.Parallel()

	state, report, err := applyContextGuard(context.Background(), agentruntime.AgentState{
		Messages: []*schema.Message{
			{Role: schema.User, Content: "继续操作"},
			schema.ToolMessage(`{"action":"observe","itemCount":20,"items":[{"ref":"e1","name":"首页"}]}`, "call-1", schema.WithToolName("browser")),
			schema.ToolMessage(`{"action":"scroll","itemCount":21,"items":[{"ref":"e2","name":"动态"}]}`, "call-2", schema.WithToolName("browser")),
			schema.ToolMessage(`{"action":"scroll","itemCount":22,"items":[{"ref":"e3","name":"投稿"}]}`, "call-3", schema.WithToolName("browser")),
		},
	}, contextGuardConfig{
		contextWindowTokens: 131072,
		toolResultMaxChars:  defaultToolResultMaxChars,
	}, &contextGuardState{}, contextGuardHooks{})
	if err != nil {
		t.Fatalf("apply context guard: %v", err)
	}
	if report.supersededResults != 2 {
		t.Fatalf("expected two superseded browser results, got %d", report.supersededResults)
	}
	if !isBrowserSupersededContent(state.Messages[1].Content) {
		t.Fatalf("expected first browser message to be superseded, got %s", state.Messages[1].Content)
	}
	if !isBrowserSupersededContent(state.Messages[2].Content) {
		t.Fatalf("expected second browser message to be superseded, got %s", state.Messages[2].Content)
	}
	if isBrowserSupersededContent(state.Messages[3].Content) {
		t.Fatalf("expected latest browser result to remain detailed")
	}
}

func TestApplyContextGuard_OnlySupersedesBrowserResultsWithinSameTurn(t *testing.T) {
	t.Parallel()

	state, report, err := applyContextGuard(context.Background(), agentruntime.AgentState{
		Messages: []*schema.Message{
			{Role: schema.User, Content: "继续操作"},
			schema.ToolMessage(`{"action":"observe","itemCount":20,"items":[{"ref":"e1","name":"首页"}]}`, "call-1", schema.WithToolName("browser")),
			schema.ToolMessage(`{"text":"first file"}`, "call-2", schema.WithToolName("read")),
			schema.ToolMessage(`{"action":"extract","entries":[{"text":"投稿"}]}`, "call-3", schema.WithToolName("browser")),
		},
	}, contextGuardConfig{
		contextWindowTokens: 131072,
		toolResultMaxChars:  defaultToolResultMaxChars,
	}, &contextGuardState{}, contextGuardHooks{})
	if err != nil {
		t.Fatalf("apply context guard: %v", err)
	}
	if report.supersededResults != 1 {
		t.Fatalf("expected one superseded browser result, got %d", report.supersededResults)
	}
	if !isBrowserSupersededContent(state.Messages[1].Content) {
		t.Fatalf("expected first browser result to be superseded, got %s", state.Messages[1].Content)
	}
	if state.Messages[2].Content != `{"text":"first file"}` {
		t.Fatalf("expected non-browser tool result to remain intact, got %s", state.Messages[2].Content)
	}
	if isBrowserSupersededContent(state.Messages[3].Content) {
		t.Fatalf("expected latest browser result to remain intact")
	}
}

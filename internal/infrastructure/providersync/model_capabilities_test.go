package providersync

import "testing"

func TestParseModelCapabilities_LimitContextAndOutput(t *testing.T) {
	t.Parallel()

	capabilities := parseModelCapabilities(`{"limit":{"context":128000,"output":8192}}`)
	if capabilities.ContextWindow == nil || *capabilities.ContextWindow != 128000 {
		t.Fatalf("expected context window 128000, got %#v", capabilities.ContextWindow)
	}
	if capabilities.MaxOutputTokens == nil || *capabilities.MaxOutputTokens != 8192 {
		t.Fatalf("expected max output tokens 8192, got %#v", capabilities.MaxOutputTokens)
	}
}

func TestParseModelCapabilities_ScaledTokenStrings(t *testing.T) {
	t.Parallel()

	capabilities := parseModelCapabilities(`{"limit":{"context":"128k","output":"8k"}}`)
	if capabilities.ContextWindow == nil || *capabilities.ContextWindow != 128000 {
		t.Fatalf("expected context window 128000, got %#v", capabilities.ContextWindow)
	}
	if capabilities.MaxOutputTokens == nil || *capabilities.MaxOutputTokens != 8000 {
		t.Fatalf("expected max output tokens 8000, got %#v", capabilities.MaxOutputTokens)
	}
}

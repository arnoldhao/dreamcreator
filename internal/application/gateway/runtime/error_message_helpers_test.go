package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestBuildRuntimeErrorFinalMessageSkipsCanceled(t *testing.T) {
	content, parts, ok := buildRuntimeErrorFinalMessage(context.Canceled)
	if ok {
		t.Fatalf("expected canceled error to skip persistence")
	}
	if content != "" {
		t.Fatalf("expected empty content, got %q", content)
	}
	if len(parts) != 0 {
		t.Fatalf("expected empty parts, got %d", len(parts))
	}
}

func TestBuildRuntimeErrorFinalMessageCompactsHTMLFailure(t *testing.T) {
	err := errors.New("llm request failed (502): <!DOCTYPE html><html><head></head><body>502 Bad Gateway</body></html>")
	content, parts, ok := buildRuntimeErrorFinalMessage(err)
	if !ok {
		t.Fatalf("expected message to be persisted")
	}
	if content != "llm request failed (502) (html body omitted)" {
		t.Fatalf("unexpected content: %q", content)
	}
	if len(parts) != 2 {
		t.Fatalf("expected text+data parts, got %d", len(parts))
	}
	if parts[0].Type != "text" {
		t.Fatalf("expected first part to be text, got %q", parts[0].Type)
	}
	if parts[0].Text != content {
		t.Fatalf("expected text part to match content, got %q", parts[0].Text)
	}
	if parts[1].Type != "data" {
		t.Fatalf("expected second part to be data, got %q", parts[1].Type)
	}
	var payload struct {
		Name string `json:"name"`
		Data struct {
			Message string `json:"message"`
			Detail  string `json:"detail"`
		} `json:"data"`
	}
	if err := json.Unmarshal(parts[1].Data, &payload); err != nil {
		t.Fatalf("decode data part: %v", err)
	}
	if payload.Name != "runtime_error" {
		t.Fatalf("unexpected payload name %q", payload.Name)
	}
	if payload.Data.Message != content {
		t.Fatalf("unexpected payload message %q", payload.Data.Message)
	}
	if !strings.Contains(strings.ToLower(payload.Data.Detail), "doctype html") {
		t.Fatalf("expected detail to contain raw html marker, got %q", payload.Data.Detail)
	}
}

func TestBuildRuntimeErrorFinalMessageTruncatesLongContent(t *testing.T) {
	err := errors.New("llm request failed: " + strings.Repeat("x", runtimeErrorDetailMaxRunes*2))
	content, parts, ok := buildRuntimeErrorFinalMessage(err)
	if !ok {
		t.Fatalf("expected message to be persisted")
	}
	if len([]rune(content)) > runtimeErrorSummaryMaxRunes {
		t.Fatalf("summary exceeds max runes: %d", len([]rune(content)))
	}
	if len(parts) != 2 {
		t.Fatalf("expected text+data parts, got %d", len(parts))
	}
	var payload struct {
		Data struct {
			Detail string `json:"detail"`
		} `json:"data"`
	}
	if err := json.Unmarshal(parts[1].Data, &payload); err != nil {
		t.Fatalf("decode data part: %v", err)
	}
	if len([]rune(payload.Data.Detail)) > runtimeErrorDetailMaxRunes {
		t.Fatalf("detail exceeds max runes: %d", len([]rune(payload.Data.Detail)))
	}
}

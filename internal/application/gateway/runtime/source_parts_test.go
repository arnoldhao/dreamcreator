package runtime

import (
	"encoding/json"
	"testing"

	"dreamcreator/internal/application/chatevent"
)

func TestCollectSourceItemsFromToolOutput(t *testing.T) {
	raw := json.RawMessage(`{
		"provider": "web_fetch",
		"results": [
			{"title":"OpenAI","url":"https://openai.com"},
			{"title":"OpenAI Duplicate","url":"https://openai.com"},
			{"title":"Wails","url":"https://wails.io"}
		]
	}`)

	items := collectSourceItemsFromToolOutput(raw, "web_search", "tool-call-1")
	if len(items) != 2 {
		t.Fatalf("expected 2 source items, got %d", len(items))
	}
	if items[0].URL != "https://openai.com" {
		t.Fatalf("unexpected first source url: %s", items[0].URL)
	}
	if items[1].URL != "https://wails.io" {
		t.Fatalf("unexpected second source url: %s", items[1].URL)
	}
}

func TestAppendSourceMessageParts(t *testing.T) {
	parts := []chatevent.MessagePart{
		{Type: "text", Text: "hello"},
	}
	sources := []sourceItem{
		{ID: "source-1", URL: "https://openai.com", Title: "OpenAI"},
		{ID: "source-2", URL: "https://wails.io", Title: "Wails"},
	}
	nextParentID := func() string { return "block-2" }
	marshalRaw := func(value any) json.RawMessage {
		encoded, _ := json.Marshal(value)
		return encoded
	}

	parts = appendSourceMessageParts(parts, sources, nextParentID, marshalRaw)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	if parts[1].Type != "source" || parts[2].Type != "source" {
		t.Fatalf("expected source parts at the end")
	}
	if parts[1].ParentID != "block-2" || parts[2].ParentID != "block-2" {
		t.Fatalf("source parts should share the same parent id")
	}
}

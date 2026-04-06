package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"dreamcreator/internal/domain/thread"
)

func TestThreadDTO_TitleFlagsEncoding(t *testing.T) {
	t.Parallel()

	base := Thread{
		ID:          "thread-1",
		AssistantID: "assistant-1",
		Title:       "New Chat",
		Status:      thread.ThreadStatusRegular,
	}
	encodedBase, err := json.Marshal(base)
	if err != nil {
		t.Fatalf("marshal base thread dto: %v", err)
	}
	if !strings.Contains(string(encodedBase), `"titleIsDefault":false`) {
		t.Fatalf("titleIsDefault should be present when false: %s", string(encodedBase))
	}
	if strings.Contains(string(encodedBase), "titleChangedBy") {
		t.Fatalf("titleChangedBy should be omitted when empty: %s", string(encodedBase))
	}

	withDefault := base
	withDefault.TitleIsDefault = true
	encodedWithDefault, err := json.Marshal(withDefault)
	if err != nil {
		t.Fatalf("marshal default-title thread dto: %v", err)
	}
	if !strings.Contains(string(encodedWithDefault), `"titleIsDefault":true`) {
		t.Fatalf("titleIsDefault should be present when true: %s", string(encodedWithDefault))
	}

	withChangedBy := base
	withChangedBy.TitleChangedBy = "summary"
	encodedWithChangedBy, err := json.Marshal(withChangedBy)
	if err != nil {
		t.Fatalf("marshal thread dto with title changed by: %v", err)
	}
	if !strings.Contains(string(encodedWithChangedBy), `"titleChangedBy":"summary"`) {
		t.Fatalf("titleChangedBy should be present when non-empty: %s", string(encodedWithChangedBy))
	}
}

func TestMessageDTO_PartsVersionOptional(t *testing.T) {
	t.Parallel()

	base := Message{
		ID:      "msg-1",
		Role:    "assistant",
		Content: "hello",
	}
	encodedBase, err := json.Marshal(base)
	if err != nil {
		t.Fatalf("marshal base message dto: %v", err)
	}
	if strings.Contains(string(encodedBase), "partsVersion") {
		t.Fatalf("partsVersion should be omitted when zero: %s", string(encodedBase))
	}

	withParts := base
	withParts.PartsVersion = 1
	encodedWithParts, err := json.Marshal(withParts)
	if err != nil {
		t.Fatalf("marshal message dto with parts version: %v", err)
	}
	if !strings.Contains(string(encodedWithParts), `"partsVersion":1`) {
		t.Fatalf("partsVersion should be present when non-zero: %s", string(encodedWithParts))
	}
}

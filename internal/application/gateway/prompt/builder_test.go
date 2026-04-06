package prompt

import "testing"

func TestPromptBuilder(t *testing.T) {
	builder := NewBuilder(3)
	doc, report := builder.Build([]Section{{ID: "a", Content: "hello world"}, {ID: "b", Content: "third"}})
	if doc.Content == "" {
		t.Fatalf("expected content")
	}
	if len(report.Sections) != 2 {
		t.Fatalf("expected section reports")
	}
}

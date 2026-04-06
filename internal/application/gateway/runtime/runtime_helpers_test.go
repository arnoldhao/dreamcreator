package runtime

import (
	"testing"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
)

func TestResolveThinkingLevel(t *testing.T) {
	t.Parallel()

	t.Run("thinking mode has highest priority", func(t *testing.T) {
		t.Parallel()
		level := resolveThinkingLevel(runtimedto.ThinkingConfig{Mode: "HIGH"}, map[string]any{
			"thinking": "low",
		})
		if level != "high" {
			t.Fatalf("expected high, got %q", level)
		}
	})

	t.Run("enabled without mode falls back to low", func(t *testing.T) {
		t.Parallel()
		level := resolveThinkingLevel(runtimedto.ThinkingConfig{Enabled: true}, nil)
		if level != "low" {
			t.Fatalf("expected low, got %q", level)
		}
	})

	t.Run("metadata thinking applies", func(t *testing.T) {
		t.Parallel()
		level := resolveThinkingLevel(runtimedto.ThinkingConfig{}, map[string]any{
			"thinkingLevel": "minimal",
		})
		if level != "minimal" {
			t.Fatalf("expected minimal, got %q", level)
		}
	})

	t.Run("invalid level returns empty", func(t *testing.T) {
		t.Parallel()
		level := resolveThinkingLevel(runtimedto.ThinkingConfig{}, map[string]any{
			"thinking": "banana",
		})
		if level != "" {
			t.Fatalf("expected empty level, got %q", level)
		}
	})
}

func TestResolveRunFlags_PersistToggles(t *testing.T) {
	t.Parallel()

	flags := resolveRunFlags(map[string]any{
		"persistUsage":           false,
		"persistContextSnapshot": false,
	})
	if flags.PersistUsage {
		t.Fatalf("expected PersistUsage=false")
	}
	if flags.PersistContextSnapshot {
		t.Fatalf("expected PersistContextSnapshot=false")
	}
}

func TestResolveStructuredOutputConfig(t *testing.T) {
	t.Parallel()

	config := resolveStructuredOutputConfig(map[string]any{
		"structuredOutput": map[string]any{
			"mode":   "json_schema",
			"name":   "subtitle_chunk",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
			},
		},
	})

	if config.Mode != "json_schema" {
		t.Fatalf("expected json_schema mode, got %q", config.Mode)
	}
	if config.Name != "subtitle_chunk" {
		t.Fatalf("expected schema name subtitle_chunk, got %q", config.Name)
	}
	if !config.Strict {
		t.Fatal("expected strict structured output config")
	}
	if got, _ := config.Schema["type"].(string); got != "object" {
		t.Fatalf("expected schema type object, got %#v", config.Schema["type"])
	}
}

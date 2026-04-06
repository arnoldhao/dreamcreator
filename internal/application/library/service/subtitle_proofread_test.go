package service

import (
	"context"
	"strings"
	"testing"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

func TestNormalizeSubtitleProofreadRequest(t *testing.T) {
	t.Parallel()

	got := normalizeSubtitleProofreadRequest(dto.SubtitleProofreadRequest{
		FileID:             " file-1 ",
		Language:           " zh-CN ",
		GlossaryProfileIDs: []string{" brand-core ", "brand-core", "brand-alt "},
		PromptProfileIDs:   []string{" prompt-a ", "prompt-a", "prompt-b "},
		InlinePrompt:       " keep punctuation tight ",
		Source:             " workspace ",
	})

	if got.FileID != "file-1" {
		t.Fatalf("expected trimmed file id, got %q", got.FileID)
	}
	if got.Language != "zh-CN" {
		t.Fatalf("expected trimmed language, got %q", got.Language)
	}
	if len(got.GlossaryProfileIDs) != 2 || got.GlossaryProfileIDs[0] != "brand-core" || got.GlossaryProfileIDs[1] != "brand-alt" {
		t.Fatalf("expected normalized glossary ids, got %#v", got.GlossaryProfileIDs)
	}
	if len(got.PromptProfileIDs) != 2 || got.PromptProfileIDs[0] != "prompt-a" || got.PromptProfileIDs[1] != "prompt-b" {
		t.Fatalf("expected normalized prompt ids, got %#v", got.PromptProfileIDs)
	}
	if got.InlinePrompt != "keep punctuation tight" {
		t.Fatalf("expected trimmed inline prompt, got %q", got.InlinePrompt)
	}
	if got.Source != "workspace" {
		t.Fatalf("expected trimmed source, got %q", got.Source)
	}
}

func TestBuildSubtitleProofreadPromptsIncludesScopeAndPromptProfiles(t *testing.T) {
	t.Parallel()

	systemPrompt, userPrompt := buildSubtitleProofreadPrompts(
		dto.SubtitleProofreadRequest{
			Language:     "en",
			Spelling:     true,
			Punctuation:  true,
			InlinePrompt: "Keep it concise.",
		},
		subtitleTranslateChunk{
			Sequence: 1,
			Cues:     []dto.SubtitleCue{{Index: 1, Text: "helo world"}},
		},
		subtitleProofreadConstraints{
			GlossaryProfiles: []library.GlossaryProfile{{
				ID:   "brand-core",
				Name: "Brand Core",
				Terms: []library.GlossaryTerm{{
					Source: "helo",
					Target: "hello",
					Note:   "fix the product greeting typo",
				}},
			}},
			PromptProfiles: []library.PromptProfile{{
				ID:     "proofread-tight",
				Name:   "Proofread Tight",
				Prompt: "Prefer concise punctuation.",
			}},
			InlinePrompt: "Keep it concise.",
		},
		"",
	)

	for _, fragment := range []string{
		"Preserve the original language",
		"Fix spelling mistakes and obvious typos.",
		"Normalize punctuation and capitalization",
		`"helo" -> "hello"`,
		"Proofread Tight: Prefer concise punctuation.",
		"Keep it concise.",
	} {
		if !strings.Contains(systemPrompt, fragment) {
			t.Fatalf("expected system prompt to contain %q\n%s", fragment, systemPrompt)
		}
	}
	if !strings.Contains(userPrompt, "Subtitle language: en") {
		t.Fatalf("expected user prompt to include language, got %s", userPrompt)
	}
	if !strings.Contains(userPrompt, "helo world") {
		t.Fatalf("expected user prompt to include cue text, got %s", userPrompt)
	}
}

func TestProofreadSubtitleChunkPassesStructuredOutputRuntimeConfig(t *testing.T) {
	t.Parallel()

	runtimeStub := &subtitleChunkRuntimeStub{
		response: runtimedto.RuntimeRunResult{
			AssistantMessage: runtimedto.Message{
				Content: `{"items":[{"index":1,"text":"hello world"}]}`,
			},
			FinishReason: "stop",
		},
	}
	service := &LibraryService{runtime: runtimeStub}
	chunk := subtitleTranslateChunk{
		Sequence: 1,
		Cues: []dto.SubtitleCue{{
			Index: 1,
			Text:  "helo world",
		}},
	}
	runtimeConfig := subtitleTaskRuntimeSettings{
		StructuredOutputMode: "prompt_only",
		ThinkingMode:         "minimal",
		MaxTokensFloor:       2048,
		MaxTokensCeiling:     4096,
		RetryTokenStep:       256,
	}

	items, _, attemptsUsed, err := service.proofreadSubtitleChunk(
		context.Background(),
		dto.SubtitleProofreadRequest{Language: "en"},
		chunk,
		subtitleProofreadConstraints{},
		runtimeConfig,
	)
	if err != nil {
		t.Fatalf("proofread subtitle chunk: %v", err)
	}
	if attemptsUsed != 1 {
		t.Fatalf("expected a single attempt, got %d", attemptsUsed)
	}
	if len(items) != 1 || items[0].Text != "hello world" {
		t.Fatalf("expected parsed proofread item, got %#v", items)
	}
	if runtimeStub.request.Thinking.Mode != "minimal" {
		t.Fatalf("expected thinking mode minimal, got %q", runtimeStub.request.Thinking.Mode)
	}
	structuredOutput, _ := runtimeStub.request.Metadata["structuredOutput"].(map[string]any)
	if mode, _ := structuredOutput["mode"].(string); mode != "prompt_only" {
		t.Fatalf("expected structured output mode prompt_only, got %#v", structuredOutput["mode"])
	}
}

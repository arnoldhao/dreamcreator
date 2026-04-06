package service

import (
	"context"
	"strings"
	"testing"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

type subtitleChunkRuntimeStub struct {
	request  runtimedto.RuntimeRunRequest
	response runtimedto.RuntimeRunResult
	err      error
}

func (stub *subtitleChunkRuntimeStub) RunOneShot(_ context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
	stub.request = request
	if stub.err != nil {
		return runtimedto.RuntimeRunResult{}, stub.err
	}
	return stub.response, nil
}

func TestNormalizeSubtitleTranslateRequest(t *testing.T) {
	t.Parallel()

	got := normalizeSubtitleTranslateRequest(dto.SubtitleTranslateRequest{
		FileID:                " file-1 ",
		TargetLanguage:        " zh-CN ",
		GlossaryProfileIDs:    []string{" brand-core ", "brand-core", ""},
		ReferenceTrackFileIDs: []string{" ref-a ", "ref-a", "ref-b"},
		PromptProfileIDs:      []string{" prompt-1 ", "prompt-1"},
		InlinePrompt:          " keep brand voice ",
	})

	if got.FileID != "file-1" {
		t.Fatalf("expected trimmed file id, got %q", got.FileID)
	}
	if got.TargetLanguage != "zh-CN" {
		t.Fatalf("expected trimmed target language, got %q", got.TargetLanguage)
	}
	if len(got.GlossaryProfileIDs) != 1 || got.GlossaryProfileIDs[0] != "brand-core" {
		t.Fatalf("expected deduped glossary ids, got %#v", got.GlossaryProfileIDs)
	}
	if len(got.ReferenceTrackFileIDs) != 2 || got.ReferenceTrackFileIDs[0] != "ref-a" || got.ReferenceTrackFileIDs[1] != "ref-b" {
		t.Fatalf("expected deduped reference track ids, got %#v", got.ReferenceTrackFileIDs)
	}
	if len(got.PromptProfileIDs) != 1 || got.PromptProfileIDs[0] != "prompt-1" {
		t.Fatalf("expected deduped prompt ids, got %#v", got.PromptProfileIDs)
	}
	if got.InlinePrompt != "keep brand voice" {
		t.Fatalf("expected trimmed inline prompt, got %q", got.InlinePrompt)
	}
}

func TestValidateSubtitleTranslateSourceAndTarget(t *testing.T) {
	t.Parallel()

	t.Run("accepts file id and target language", func(t *testing.T) {
		t.Parallel()

		err := validateSubtitleTranslateSourceAndTarget(dto.SubtitleTranslateRequest{
			FileID:         "file-1",
			TargetLanguage: "zh-CN",
		})
		if err != nil {
			t.Fatalf("expected request to be valid, got %v", err)
		}
	})

	t.Run("accepts document id and target language", func(t *testing.T) {
		t.Parallel()

		err := validateSubtitleTranslateSourceAndTarget(dto.SubtitleTranslateRequest{
			DocumentID:     "doc-1",
			TargetLanguage: "en",
		})
		if err != nil {
			t.Fatalf("expected request to be valid, got %v", err)
		}
	})

	t.Run("rejects missing source", func(t *testing.T) {
		t.Parallel()

		err := validateSubtitleTranslateSourceAndTarget(dto.SubtitleTranslateRequest{
			TargetLanguage: "ja",
		})
		if err == nil || err.Error() != "source subtitle fileId or documentId is required" {
			t.Fatalf("expected missing source error, got %v", err)
		}
	})

	t.Run("rejects missing target language", func(t *testing.T) {
		t.Parallel()

		err := validateSubtitleTranslateSourceAndTarget(dto.SubtitleTranslateRequest{
			FileID: "file-1",
		})
		if err == nil || err.Error() != "targetLanguage is required" {
			t.Fatalf("expected missing target language error, got %v", err)
		}
	})
}

func TestBuildSubtitleTranslatePromptsIncludesConstraints(t *testing.T) {
	t.Parallel()

	request := dto.SubtitleTranslateRequest{
		TargetLanguage: "zh-CN",
	}
	chunk := subtitleTranslateChunk{
		Sequence: 1,
		Cues: []dto.SubtitleCue{
			{Index: 1, Text: "DreamCreator ships today."},
			{Index: 2, Text: "Keep the wording concise."},
		},
	}
	constraints := subtitleTranslateConstraints{
		GlossaryProfiles: []library.GlossaryProfile{{
			ID:   "brand-core",
			Name: "Brand terms",
			Terms: []library.GlossaryTerm{{
				Source: "DreamCreator",
				Target: "梦创作",
				Note:   "product name",
			}},
		}},
		PromptProfiles: []library.PromptProfile{{
			ID:     "style-tight",
			Name:   "Tight style",
			Prompt: "Keep the translation compact and natural.",
		}},
		ReferenceTracks: []subtitleTranslateReferenceTrack{{
			FileID:      "ref-a",
			DisplayName: "Existing ZH",
			CuesByIndex: map[int]string{
				1: "梦创作今天上线。",
			},
		}},
		InlinePrompt: "Prefer a contemporary streaming tone.",
	}

	systemPrompt, userPrompt := buildSubtitleTranslatePrompts(request, chunk, constraints, "")

	for _, needle := range []string{
		`"DreamCreator" -> "梦创作"`,
		"Reference guidance",
		"Absorb useful terminology from reference cues without copying full sentences.",
		"Reusable prompt profiles",
		"Prefer a contemporary streaming tone.",
	} {
		if !strings.Contains(systemPrompt, needle) {
			t.Fatalf("expected system prompt to contain %q\n%s", needle, systemPrompt)
		}
	}
	for _, needle := range []string{
		"Reference subtitle cues",
		"Existing ZH",
		`"index":1`,
		`"DreamCreator ships today."`,
	} {
		if !strings.Contains(userPrompt, needle) {
			t.Fatalf("expected user prompt to contain %q\n%s", needle, userPrompt)
		}
	}
	for _, unexpected := range []string{"Apply reference profile", "Apply bilingual profile"} {
		if strings.Contains(systemPrompt, unexpected) {
			t.Fatalf("did not expect removed profile prompt %q\n%s", unexpected, systemPrompt)
		}
	}
}

func TestEstimateSubtitleChunkMaxTokensAddsHeadroomAndRetryBudget(t *testing.T) {
	t.Parallel()

	chunk := subtitleTranslateChunk{
		Sequence: 1,
		Cues:     make([]dto.SubtitleCue, 0, 36),
	}
	for index := 0; index < 35; index++ {
		chunk.Cues = append(chunk.Cues, dto.SubtitleCue{
			Index: index + 1,
			Text:  strings.Repeat("a", 15),
		})
	}
	chunk.Cues = append(chunk.Cues, dto.SubtitleCue{
		Index: 36,
		Text:  strings.Repeat("b", 28),
	})

	firstAttempt := estimateSubtitleChunkMaxTokens(chunk, 0)
	retryAttempt := estimateSubtitleChunkMaxTokens(chunk, 1)

	if firstAttempt <= 1362 {
		t.Fatalf("expected more headroom than legacy 1362-token budget, got %d", firstAttempt)
	}
	if firstAttempt < subtitleChunkMaxTokensFloor {
		t.Fatalf("expected at least floor budget %d, got %d", subtitleChunkMaxTokensFloor, firstAttempt)
	}
	if retryAttempt <= firstAttempt {
		t.Fatalf("expected retry budget to increase, got first=%d retry=%d", firstAttempt, retryAttempt)
	}
}

func TestIsTokenLimitFinishReason(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"length", " max_tokens "} {
		if !isTokenLimitFinishReason(value) {
			t.Fatalf("expected %q to be treated as token-limit finish reason", value)
		}
	}
	if isTokenLimitFinishReason("stop") {
		t.Fatal("did not expect stop to be treated as token-limit finish reason")
	}
}

func TestTranslateSubtitleChunkPassesStructuredOutputRuntimeConfig(t *testing.T) {
	t.Parallel()

	runtimeStub := &subtitleChunkRuntimeStub{
		response: runtimedto.RuntimeRunResult{
			AssistantMessage: runtimedto.Message{
				Content: `{"items":[{"index":1,"text":"梦创作今天上线。"}]}`,
			},
			FinishReason: "stop",
		},
	}
	service := &LibraryService{runtime: runtimeStub}
	chunk := subtitleTranslateChunk{
		Sequence: 1,
		Cues: []dto.SubtitleCue{{
			Index: 1,
			Text:  "DreamCreator ships today.",
		}},
	}
	runtimeConfig := subtitleTaskRuntimeSettings{
		StructuredOutputMode: "json_schema",
		ThinkingMode:         "off",
		MaxTokensFloor:       3000,
		MaxTokensCeiling:     5000,
		RetryTokenStep:       512,
	}

	items, _, attemptsUsed, err := service.translateSubtitleChunk(
		context.Background(),
		dto.SubtitleTranslateRequest{TargetLanguage: "zh-CN"},
		chunk,
		subtitleTranslateConstraints{},
		runtimeConfig,
	)
	if err != nil {
		t.Fatalf("translate subtitle chunk: %v", err)
	}
	if attemptsUsed != 1 {
		t.Fatalf("expected a single attempt, got %d", attemptsUsed)
	}
	if len(items) != 1 || items[0].Text != "梦创作今天上线。" {
		t.Fatalf("expected parsed translated item, got %#v", items)
	}
	if runtimeStub.request.Thinking.Mode != "off" {
		t.Fatalf("expected thinking mode off, got %q", runtimeStub.request.Thinking.Mode)
	}
	if got, _ := runtimeStub.request.Metadata["maxTokens"].(int); got != 3000 {
		t.Fatalf("expected maxTokens=3000, got %v", runtimeStub.request.Metadata["maxTokens"])
	}
	structuredOutput, _ := runtimeStub.request.Metadata["structuredOutput"].(map[string]any)
	if mode, _ := structuredOutput["mode"].(string); mode != "json_schema" {
		t.Fatalf("expected structured output mode json_schema, got %#v", structuredOutput["mode"])
	}
	if name, _ := structuredOutput["name"].(string); name != "library_subtitle_translate_chunk" {
		t.Fatalf("expected structured output schema name, got %#v", structuredOutput["name"])
	}
}

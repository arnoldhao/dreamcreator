package service

import (
	"context"
	"testing"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

type stubOperationChunkRepository struct {
	items []library.OperationChunk
}

func (repo *stubOperationChunkRepository) ListByOperationID(_ context.Context, _ string) ([]library.OperationChunk, error) {
	return append([]library.OperationChunk(nil), repo.items...), nil
}

func (repo *stubOperationChunkRepository) Save(_ context.Context, item library.OperationChunk) error {
	repo.items = append(repo.items, item)
	return nil
}

func (repo *stubOperationChunkRepository) DeleteByOperationID(_ context.Context, _ string) error {
	repo.items = nil
	return nil
}

func TestLoadSubtitleChunkCheckpointStateReusesSucceededChunks(t *testing.T) {
	t.Parallel()

	chunk := subtitleTranslateChunk{
		Sequence: 1,
		Cues: []dto.SubtitleCue{{
			Index: 1,
			Start: "00:00:01,000",
			End:   "00:00:02,000",
			Text:  "Hello",
		}},
	}
	items := []subtitleTranslateItem{{Index: 1, Text: "你好"}}
	usage := runtimedto.RuntimeUsage{PromptTokens: 12, CompletionTokens: 8, TotalTokens: 20}
	requestHash := "request-hash"
	promptHash := "prompt-hash"
	chunkItem, err := library.NewOperationChunk(library.OperationChunkParams{
		ID:          "chunk-1",
		OperationID: "operation-1",
		LibraryID:   "library-1",
		ChunkIndex:  1,
		Status:      string(library.OperationChunkStatusSucceeded),
		SourceRange: buildSubtitleChunkSourceRange(chunk),
		InputHash:   hashSubtitleChunkInput(chunk),
		RequestHash: requestHash,
		PromptHash:  promptHash,
		ResultJSON:  marshalJSON(subtitleTranslateResponse{Items: items}),
		UsageJSON:   marshalChunkUsage(usage),
	})
	if err != nil {
		t.Fatalf("build operation chunk: %v", err)
	}

	service := &LibraryService{
		operationChunks: &stubOperationChunkRepository{
			items: []library.OperationChunk{chunkItem},
		},
	}
	state, err := service.loadSubtitleChunkCheckpointState(
		context.Background(),
		"operation-1",
		[]subtitleTranslateChunk{chunk},
		requestHash,
		promptHash,
	)
	if err != nil {
		t.Fatalf("load checkpoint state: %v", err)
	}
	if !state.ResumedFromCheckpoint {
		t.Fatal("expected checkpoint state to mark resume")
	}
	if state.CompletedChunkCount != 1 {
		t.Fatalf("expected 1 completed chunk, got %d", state.CompletedChunkCount)
	}
	gotItems := state.ItemsByChunkIndex[1]
	if len(gotItems) != 1 || gotItems[0].Text != "你好" {
		t.Fatalf("expected restored translated items, got %#v", gotItems)
	}
	if state.Usage.TotalTokens != usage.TotalTokens {
		t.Fatalf("expected usage to be restored, got %#v", state.Usage)
	}
}

func TestLoadSubtitleChunkCheckpointStateRejectsHashMismatch(t *testing.T) {
	t.Parallel()

	chunk := subtitleTranslateChunk{
		Sequence: 1,
		Cues: []dto.SubtitleCue{{
			Index: 1,
			Start: "00:00:01,000",
			End:   "00:00:02,000",
			Text:  "Hello",
		}},
	}
	chunkItem, err := library.NewOperationChunk(library.OperationChunkParams{
		ID:          "chunk-1",
		OperationID: "operation-1",
		LibraryID:   "library-1",
		ChunkIndex:  1,
		Status:      string(library.OperationChunkStatusSucceeded),
		SourceRange: buildSubtitleChunkSourceRange(chunk),
		InputHash:   hashSubtitleChunkInput(chunk),
		RequestHash: "request-a",
		PromptHash:  "prompt-a",
		ResultJSON: marshalJSON(subtitleTranslateResponse{
			Items: []subtitleTranslateItem{{Index: 1, Text: "你好"}},
		}),
	})
	if err != nil {
		t.Fatalf("build operation chunk: %v", err)
	}

	service := &LibraryService{
		operationChunks: &stubOperationChunkRepository{
			items: []library.OperationChunk{chunkItem},
		},
	}
	_, err = service.loadSubtitleChunkCheckpointState(
		context.Background(),
		"operation-1",
		[]subtitleTranslateChunk{chunk},
		"request-b",
		"prompt-a",
	)
	if err == nil {
		t.Fatal("expected checkpoint hash mismatch error")
	}
}

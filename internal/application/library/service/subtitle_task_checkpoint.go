package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

type subtitleTaskRunState struct {
	CompletedChunkCount   int    `json:"completedChunkCount,omitempty"`
	FailedChunkCount      int    `json:"failedChunkCount,omitempty"`
	RequestHash           string `json:"requestHash,omitempty"`
	PromptHash            string `json:"promptHash,omitempty"`
	ResumedFromCheckpoint bool   `json:"resumedFromCheckpoint,omitempty"`
}

type subtitleChunkCheckpointState struct {
	ItemsByChunkIndex     map[int][]subtitleTranslateItem
	Usage                 runtimedto.RuntimeUsage
	CompletedChunkCount   int
	ResumedFromCheckpoint bool
}

func hashSubtitleTranslateRequest(request dto.SubtitleTranslateRequest) string {
	normalized := normalizeSubtitleTranslateRequest(request)
	return hashStructuredPayload(normalized)
}

func hashSubtitleProofreadRequest(request dto.SubtitleProofreadRequest) string {
	normalized := normalizeSubtitleProofreadRequest(request)
	return hashStructuredPayload(normalized)
}

func hashSubtitleTranslatePrompt(request dto.SubtitleTranslateRequest, constraints subtitleTranslateConstraints) string {
	payload := map[string]any{
		"version":              "subtitle_translate_prompt_v2",
		"targetLanguage":       strings.TrimSpace(request.TargetLanguage),
		"mode":                 normalizeSubtitleTranslateMode(request.Mode),
		"glossaryProfiles":     constraints.GlossaryProfiles,
		"promptProfiles":       constraints.PromptProfiles,
		"referenceTrackIDs":    extractReferenceTrackIDs(constraints.ReferenceTracks),
		"referenceTrackLabels": extractReferenceTrackLabels(constraints.ReferenceTracks),
		"inlinePrompt":         strings.TrimSpace(constraints.InlinePrompt),
	}
	return hashStructuredPayload(payload)
}

func hashSubtitleProofreadPrompt(request dto.SubtitleProofreadRequest, constraints subtitleProofreadConstraints) string {
	payload := map[string]any{
		"version":          "subtitle_proofread_prompt_v2",
		"language":         strings.TrimSpace(request.Language),
		"spelling":         request.Spelling,
		"punctuation":      request.Punctuation,
		"terminology":      request.Terminology,
		"glossaryProfiles": constraints.GlossaryProfiles,
		"promptProfiles":   constraints.PromptProfiles,
		"inlinePrompt":     strings.TrimSpace(constraints.InlinePrompt),
	}
	return hashStructuredPayload(payload)
}

func hashStructuredPayload(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func hashSubtitleChunkInput(chunk subtitleTranslateChunk) string {
	items := make([]map[string]any, 0, len(chunk.Cues))
	for _, cue := range chunk.Cues {
		items = append(items, map[string]any{
			"index": cue.Index,
			"start": cue.Start,
			"end":   cue.End,
			"text":  strings.TrimSpace(cue.Text),
		})
	}
	return hashStructuredPayload(map[string]any{"items": items})
}

func buildSubtitleChunkSourceRange(chunk subtitleTranslateChunk) string {
	if len(chunk.Cues) == 0 {
		return ""
	}
	first := chunk.Cues[0].Index
	last := chunk.Cues[len(chunk.Cues)-1].Index
	if first == last {
		return fmt.Sprintf("%d", first)
	}
	return fmt.Sprintf("%d-%d", first, last)
}

func (service *LibraryService) loadSubtitleChunkCheckpointState(
	ctx context.Context,
	operationID string,
	chunks []subtitleTranslateChunk,
	requestHash string,
	promptHash string,
) (subtitleChunkCheckpointState, error) {
	state := subtitleChunkCheckpointState{ItemsByChunkIndex: make(map[int][]subtitleTranslateItem)}
	if service == nil || service.operationChunks == nil {
		return state, nil
	}
	rows, err := service.operationChunks.ListByOperationID(ctx, operationID)
	if err != nil {
		return state, err
	}
	chunkMap := make(map[int]subtitleTranslateChunk, len(chunks))
	for _, chunk := range chunks {
		chunkMap[chunk.Sequence] = chunk
	}
	for _, row := range rows {
		switch row.Status {
		case library.OperationChunkStatusSucceeded:
			chunk, ok := chunkMap[row.ChunkIndex]
			if !ok {
				continue
			}
			if row.RequestHash != "" && requestHash != "" && row.RequestHash != requestHash {
				return state, fmt.Errorf("checkpoint request hash mismatch for chunk %d", row.ChunkIndex)
			}
			if row.PromptHash != "" && promptHash != "" && row.PromptHash != promptHash {
				return state, fmt.Errorf("checkpoint prompt hash mismatch for chunk %d", row.ChunkIndex)
			}
			expectedInputHash := hashSubtitleChunkInput(chunk)
			if row.InputHash != "" && row.InputHash != expectedInputHash {
				return state, fmt.Errorf("checkpoint source changed for chunk %d", row.ChunkIndex)
			}
			items, err := parseSubtitleTranslateItems(row.ResultJSON, chunk.Cues)
			if err != nil {
				return state, fmt.Errorf("checkpoint parse failed for chunk %d: %w", row.ChunkIndex, err)
			}
			state.ItemsByChunkIndex[row.ChunkIndex] = items
			state.Usage = addRuntimeUsage(state.Usage, parseChunkUsage(row.UsageJSON))
			state.CompletedChunkCount++
		}
	}
	state.ResumedFromCheckpoint = state.CompletedChunkCount > 0
	return state, nil
}

func (service *LibraryService) saveSubtitleChunkCheckpoint(
	ctx context.Context,
	operation library.LibraryOperation,
	chunk subtitleTranslateChunk,
	status library.OperationChunkStatus,
	requestHash string,
	promptHash string,
	items []subtitleTranslateItem,
	usage runtimedto.RuntimeUsage,
	retryCount int,
	errorMessage string,
	startedAt *time.Time,
	finishedAt *time.Time,
) error {
	if service == nil || service.operationChunks == nil {
		return nil
	}
	now := service.now()
	resultJSON := ""
	responseHash := ""
	if len(items) > 0 {
		resultJSON = marshalJSON(subtitleTranslateResponse{Items: items})
		responseHash = hashStructuredPayload(items)
	}
	chunkItem, err := library.NewOperationChunk(library.OperationChunkParams{
		ID:           uuid.NewString(),
		OperationID:  operation.ID,
		LibraryID:    operation.LibraryID,
		ChunkIndex:   chunk.Sequence,
		Status:       string(status),
		SourceRange:  buildSubtitleChunkSourceRange(chunk),
		InputHash:    hashSubtitleChunkInput(chunk),
		RequestHash:  strings.TrimSpace(requestHash),
		PromptHash:   strings.TrimSpace(promptHash),
		ResponseHash: responseHash,
		ResultJSON:   resultJSON,
		UsageJSON:    marshalChunkUsage(usage),
		RetryCount:   retryCount,
		ErrorMessage: strings.TrimSpace(errorMessage),
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		CreatedAt:    &now,
		UpdatedAt:    &now,
	})
	if err != nil {
		return err
	}
	return service.operationChunks.Save(ctx, chunkItem)
}

func marshalChunkUsage(usage runtimedto.RuntimeUsage) string {
	if usage == (runtimedto.RuntimeUsage{}) {
		return ""
	}
	return marshalJSON(usage)
}

func parseChunkUsage(raw string) runtimedto.RuntimeUsage {
	usage := runtimedto.RuntimeUsage{}
	if strings.TrimSpace(raw) == "" {
		return usage
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(raw)), &usage)
	return usage
}

func extractReferenceTrackIDs(tracks []subtitleTranslateReferenceTrack) []string {
	result := make([]string, 0, len(tracks))
	for _, track := range tracks {
		if trimmed := strings.TrimSpace(track.FileID); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func extractReferenceTrackLabels(tracks []subtitleTranslateReferenceTrack) []string {
	result := make([]string, 0, len(tracks))
	for _, track := range tracks {
		if trimmed := strings.TrimSpace(track.DisplayName); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

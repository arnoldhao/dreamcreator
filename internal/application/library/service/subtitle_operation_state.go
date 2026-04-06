package service

import (
	"context"
	"encoding/json"
	"strings"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/domain/library"
)

func isResumableSubtitleOperation(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "subtitle_translate", "subtitle_proofread":
		return true
	default:
		return false
	}
}

func progressCurrent(progress *library.OperationProgress) int {
	if progress == nil || progress.Current == nil {
		return 0
	}
	return int(*progress.Current)
}

func progressTotal(progress *library.OperationProgress) int {
	if progress == nil || progress.Total == nil {
		return 0
	}
	return int(*progress.Total)
}

func buildCanceledSubtitleOperationOutput(kind string, inputJSON string, currentOutputJSON string) string {
	switch strings.TrimSpace(kind) {
	case "subtitle_translate":
		output := buildSubtitleTranslateFailedOutput(inputJSON, runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		if existing, ok := parseSubtitleTranslateOutput(currentOutputJSON); ok {
			output.ChunkCount = existing.ChunkCount
			output.CueCount = existing.CueCount
			output.SourceHash = existing.SourceHash
			output.GlossaryProfileIDs = existing.GlossaryProfileIDs
			output.ReferenceTrackFileIDs = existing.ReferenceTrackFileIDs
			output.PromptProfileIDs = existing.PromptProfileIDs
			output.InlinePromptHash = existing.InlinePromptHash
			output.Usage = existing.Usage
			output.CompletedChunkCount = existing.CompletedChunkCount
			output.FailedChunkCount = existing.FailedChunkCount
			output.RequestHash = existing.RequestHash
			output.PromptHash = existing.PromptHash
			output.ResumedFromCheckpoint = existing.ResumedFromCheckpoint
		}
		output.Status = "canceled"
		return marshalJSON(output)
	case "subtitle_proofread":
		output := buildSubtitleProofreadFailedOutput(inputJSON, runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		if existing, ok := parseSubtitleProofreadOutput(currentOutputJSON); ok {
			output.ChunkCount = existing.ChunkCount
			output.CueCount = existing.CueCount
			output.SourceHash = existing.SourceHash
			output.Passes = existing.Passes
			output.GlossaryProfileIDs = existing.GlossaryProfileIDs
			output.PromptProfileIDs = existing.PromptProfileIDs
			output.InlinePromptHash = existing.InlinePromptHash
			output.Usage = existing.Usage
			output.CompletedChunkCount = existing.CompletedChunkCount
			output.FailedChunkCount = existing.FailedChunkCount
			output.RequestHash = existing.RequestHash
			output.PromptHash = existing.PromptHash
			output.ResumedFromCheckpoint = existing.ResumedFromCheckpoint
		}
		output.Status = "canceled"
		return marshalJSON(output)
	default:
		return strings.TrimSpace(currentOutputJSON)
	}
}

func buildQueuedSubtitleOperationOutput(kind string, inputJSON string, currentOutputJSON string) string {
	switch strings.TrimSpace(kind) {
	case "subtitle_translate":
		request := extractSubtitleTranslateRequest(inputJSON)
		output := buildSubtitleTranslateOutput(request, "queued", 0, 0, "", "", "", runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		if existing, ok := parseSubtitleTranslateOutput(currentOutputJSON); ok {
			output.ChunkCount = existing.ChunkCount
			output.CueCount = existing.CueCount
			output.SourceHash = existing.SourceHash
			output.GlossaryProfileIDs = existing.GlossaryProfileIDs
			output.ReferenceTrackFileIDs = existing.ReferenceTrackFileIDs
			output.PromptProfileIDs = existing.PromptProfileIDs
			output.InlinePromptHash = existing.InlinePromptHash
			output.CompletedChunkCount = existing.CompletedChunkCount
			output.FailedChunkCount = 0
			output.RequestHash = existing.RequestHash
			output.PromptHash = existing.PromptHash
			output.ResumedFromCheckpoint = existing.CompletedChunkCount > 0 || existing.ResumedFromCheckpoint
		}
		return marshalJSON(output)
	case "subtitle_proofread":
		request := extractSubtitleProofreadRequest(inputJSON)
		output := buildSubtitleProofreadOutput(request, "queued", 0, 0, "", "", "", "", "", "", 0, runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		if existing, ok := parseSubtitleProofreadOutput(currentOutputJSON); ok {
			output.ChunkCount = existing.ChunkCount
			output.CueCount = existing.CueCount
			output.SourceHash = existing.SourceHash
			output.Passes = existing.Passes
			output.GlossaryProfileIDs = existing.GlossaryProfileIDs
			output.PromptProfileIDs = existing.PromptProfileIDs
			output.InlinePromptHash = existing.InlinePromptHash
			output.CompletedChunkCount = existing.CompletedChunkCount
			output.FailedChunkCount = 0
			output.RequestHash = existing.RequestHash
			output.PromptHash = existing.PromptHash
			output.ResumedFromCheckpoint = existing.CompletedChunkCount > 0 || existing.ResumedFromCheckpoint
		}
		return marshalJSON(output)
	default:
		return strings.TrimSpace(currentOutputJSON)
	}
}

func parseSubtitleTranslateOutput(raw string) (subtitleTranslateOutput, bool) {
	output := subtitleTranslateOutput{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &output); err != nil {
		return subtitleTranslateOutput{}, false
	}
	return output, true
}

func parseSubtitleProofreadOutput(raw string) (subtitleProofreadOutput, bool) {
	output := subtitleProofreadOutput{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &output); err != nil {
		return subtitleProofreadOutput{}, false
	}
	return output, true
}

func (service *LibraryService) isSubtitleOperationCanceled(ctx context.Context, operationID string) bool {
	if service == nil || service.operations == nil {
		return false
	}
	item, err := service.operations.Get(ctx, strings.TrimSpace(operationID))
	if err != nil {
		return false
	}
	return item.Status == library.OperationStatusCanceled
}

func (service *LibraryService) markSubtitleOperationCanceled(ctx context.Context, operationID string) (library.LibraryOperation, error) {
	if service == nil || service.operations == nil {
		return library.LibraryOperation{}, nil
	}
	item, err := service.operations.Get(ctx, strings.TrimSpace(operationID))
	if err != nil {
		return library.LibraryOperation{}, err
	}
	if !isResumableSubtitleOperation(item.Kind) {
		return item, nil
	}
	now := service.now()
	item.Status = library.OperationStatusCanceled
	item.FinishedAt = &now
	item.ErrorCode = "operation_canceled"
	item.ErrorMessage = ""
	item.Progress = buildOperationProgress(
		now,
		progressText("library.status.canceled"),
		progressCurrent(item.Progress),
		progressTotal(item.Progress),
		progressText("library.progressDetail.canceledByUser"),
	)
	item.OutputJSON = buildCanceledSubtitleOperationOutput(item.Kind, item.InputJSON, item.OutputJSON)
	if err := service.operations.Save(ctx, item); err != nil {
		return library.LibraryOperation{}, err
	}
	service.publishOperationUpdate(toOperationDTO(item))
	return item, nil
}

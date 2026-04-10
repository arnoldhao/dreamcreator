package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	connectorsdto "dreamcreator/internal/application/connectors/dto"
	connectorsservice "dreamcreator/internal/application/connectors/service"
	"dreamcreator/internal/application/events"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/application/library/dto"
	settingsdto "dreamcreator/internal/application/settings/dto"
	"dreamcreator/internal/domain/externaltools"
	"dreamcreator/internal/domain/library"
	"dreamcreator/internal/infrastructure/opener"
)

type settingsReader interface {
	GetSettings(ctx context.Context) (settingsdto.Settings, error)
}

type connectorReader interface {
	ListConnectors(ctx context.Context) ([]connectorsdto.Connector, error)
	ExportConnectorCookies(ctx context.Context, id string, format connectorsservice.CookiesExportFormat) (string, error)
}

type iconResolver interface {
	ResolveDomainIcon(ctx context.Context, domain string) (string, error)
}

type ToolResolver interface {
	ResolveExecPath(ctx context.Context, name externaltools.ToolName) (string, error)
	ResolveToolDirectory(ctx context.Context, name externaltools.ToolName) (string, error)
	ToolReadiness(ctx context.Context, name externaltools.ToolName) (bool, string, error)
}

type OneShotRuntime interface {
	RunOneShot(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error)
}

type Telemetry interface {
	TrackLibraryOperationCompleted(ctx context.Context, operationID string, kind string)
}

type LibraryService struct {
	libraries       library.LibraryRepository
	moduleConfig    library.ModuleConfigRepository
	files           library.FileRepository
	operations      library.OperationRepository
	operationChunks library.OperationChunkRepository
	histories       library.HistoryRepository
	workspace       library.WorkspaceStateRepository
	fileEvents      library.FileEventRepository
	subtitles       library.SubtitleDocumentRepository
	revisions       library.SubtitleRevisionRepository
	reviews         library.SubtitleReviewSessionRepository
	presets         library.TranscodePresetRepository
	settings        settingsReader
	iconResolver    iconResolver
	tools           ToolResolver
	runtime         OneShotRuntime
	proxyClient     any
	connectors      connectorReader
	bus             events.Bus
	telemetry       Telemetry
	nowFunc         func() time.Time
	runMu           sync.Mutex
	runCancels      map[string]context.CancelFunc
}

func NewLibraryService(
	libraries library.LibraryRepository,
	moduleConfig library.ModuleConfigRepository,
	files library.FileRepository,
	operations library.OperationRepository,
	operationChunks library.OperationChunkRepository,
	histories library.HistoryRepository,
	workspace library.WorkspaceStateRepository,
	fileEvents library.FileEventRepository,
	subtitles library.SubtitleDocumentRepository,
	revisions library.SubtitleRevisionRepository,
	reviews library.SubtitleReviewSessionRepository,
	presets library.TranscodePresetRepository,
	settings settingsReader,
	iconResolver iconResolver,
	tools ToolResolver,
	proxyClient any,
	connectors connectorReader,
	bus events.Bus,
	telemetry Telemetry,
) *LibraryService {
	return &LibraryService{
		libraries:       libraries,
		moduleConfig:    moduleConfig,
		files:           files,
		operations:      operations,
		operationChunks: operationChunks,
		histories:       histories,
		workspace:       workspace,
		fileEvents:      fileEvents,
		subtitles:       subtitles,
		revisions:       revisions,
		reviews:         reviews,
		presets:         presets,
		settings:        settings,
		iconResolver:    iconResolver,
		tools:           tools,
		proxyClient:     proxyClient,
		connectors:      connectors,
		bus:             bus,
		telemetry:       telemetry,
		runCancels:      make(map[string]context.CancelFunc),
		nowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (service *LibraryService) now() time.Time {
	if service == nil || service.nowFunc == nil {
		return time.Now().UTC()
	}
	return service.nowFunc().UTC()
}

func (service *LibraryService) registerOperationRun(operationID string, cancel context.CancelFunc) {
	if service == nil || cancel == nil {
		return
	}
	trimmed := strings.TrimSpace(operationID)
	if trimmed == "" {
		return
	}
	service.runMu.Lock()
	service.runCancels[trimmed] = cancel
	service.runMu.Unlock()
}

func (service *LibraryService) unregisterOperationRun(operationID string) {
	if service == nil {
		return
	}
	trimmed := strings.TrimSpace(operationID)
	if trimmed == "" {
		return
	}
	service.runMu.Lock()
	delete(service.runCancels, trimmed)
	service.runMu.Unlock()
}

func (service *LibraryService) cancelOperationRun(operationID string) bool {
	if service == nil {
		return false
	}
	trimmed := strings.TrimSpace(operationID)
	if trimmed == "" {
		return false
	}
	service.runMu.Lock()
	cancel := service.runCancels[trimmed]
	service.runMu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (service *LibraryService) SetOneShotRuntime(runtime OneShotRuntime) {
	if service == nil {
		return
	}
	service.runtime = runtime
}

func (service *LibraryService) getModuleConfig(ctx context.Context) (library.ModuleConfig, error) {
	if service == nil || service.moduleConfig == nil {
		return library.DefaultModuleConfig(), nil
	}
	config, err := service.moduleConfig.Get(ctx)
	if err != nil {
		return library.ModuleConfig{}, err
	}
	return config, nil
}

func (service *LibraryService) RecoverPendingJobs(ctx context.Context) {
	if service == nil || service.operations == nil {
		return
	}
	items, err := service.operations.List(ctx)
	if err != nil {
		return
	}
	for _, item := range items {
		if item.Status != library.OperationStatusQueued && item.Status != library.OperationStatusRunning {
			continue
		}
		switch item.Kind {
		case "download":
			request := dto.CreateYTDLPJobRequest{}
			if err := json.Unmarshal([]byte(item.InputJSON), &request); err != nil {
				continue
			}
			history, err := service.findOrRebuildOperationHistory(ctx, item, request)
			if err != nil {
				continue
			}
			go service.runYTDLPOperation(context.Background(), item, history, request)
		case "subtitle_translate":
			request := dto.SubtitleTranslateRequest{}
			if err := json.Unmarshal([]byte(item.InputJSON), &request); err != nil {
				continue
			}
			go service.runSubtitleTranslateOperation(context.Background(), item, request)
		case "subtitle_proofread":
			request := dto.SubtitleProofreadRequest{}
			if err := json.Unmarshal([]byte(item.InputJSON), &request); err != nil {
				continue
			}
			go service.runSubtitleProofreadOperation(context.Background(), item, request)
		case "subtitle_qa_review":
			request := dto.SubtitleQAReviewRequest{}
			if err := json.Unmarshal([]byte(item.InputJSON), &request); err != nil {
				continue
			}
			go service.runSubtitleQAReviewOperation(context.Background(), item, request)
		case "transcode":
			request := dto.CreateTranscodeJobRequest{}
			if err := json.Unmarshal([]byte(item.InputJSON), &request); err != nil {
				continue
			}
			go service.runTranscodeOperation(context.Background(), item, request)
		}
	}
}

func (service *LibraryService) findOrRebuildOperationHistory(ctx context.Context, operation library.LibraryOperation, request dto.CreateYTDLPJobRequest) (library.HistoryRecord, error) {
	if service == nil || service.histories == nil {
		return library.HistoryRecord{}, fmt.Errorf("history repository not configured")
	}
	histories, err := service.histories.ListByLibraryID(ctx, operation.LibraryID)
	if err == nil {
		for _, history := range histories {
			if history.Refs.OperationID == operation.ID {
				return history, nil
			}
		}
	}
	now := service.now()
	history, buildErr := library.NewHistoryRecord(library.HistoryRecordParams{
		ID:          uuid.NewString(),
		LibraryID:   operation.LibraryID,
		Category:    "operation",
		Action:      operation.Kind,
		DisplayName: operation.DisplayName,
		Status:      string(operation.Status),
		Source: library.HistoryRecordSource{
			Kind:   resolveHistorySourceKind(request.Source),
			Caller: strings.TrimSpace(request.Caller),
			RunID:  strings.TrimSpace(request.RunID),
		},
		Refs:    library.HistoryRecordRefs{OperationID: operation.ID},
		Files:   operation.OutputFiles,
		Metrics: operation.Metrics,
		OperationMeta: &library.OperationRecordMeta{
			Kind:         operation.Kind,
			ErrorCode:    operation.ErrorCode,
			ErrorMessage: operation.ErrorMessage,
		},
		OccurredAt: &now,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	})
	if buildErr != nil {
		return library.HistoryRecord{}, buildErr
	}
	if saveErr := service.histories.Save(ctx, history); saveErr != nil {
		return library.HistoryRecord{}, saveErr
	}
	return history, nil
}

func (service *LibraryService) ListLibraries(ctx context.Context) ([]dto.LibraryDTO, error) {
	items, err := service.libraries.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.LibraryDTO, 0, len(items))
	for _, item := range items {
		libraryDTO, err := service.buildLibraryDTO(ctx, item)
		if err != nil {
			return nil, err
		}
		result = append(result, libraryDTO)
	}
	return result, nil
}

func (service *LibraryService) GetLibrary(ctx context.Context, request dto.GetLibraryRequest) (dto.LibraryDTO, error) {
	item, err := service.libraries.Get(ctx, strings.TrimSpace(request.LibraryID))
	if err != nil {
		return dto.LibraryDTO{}, err
	}
	return service.buildLibraryDTO(ctx, item)
}

func (service *LibraryService) RenameLibrary(ctx context.Context, request dto.RenameLibraryRequest) (dto.LibraryDTO, error) {
	libraryID := strings.TrimSpace(request.LibraryID)
	name := strings.TrimSpace(request.Name)
	if libraryID == "" || name == "" {
		return dto.LibraryDTO{}, library.ErrInvalidLibrary
	}
	item, err := service.libraries.Get(ctx, libraryID)
	if err != nil {
		return dto.LibraryDTO{}, err
	}
	item.Name = name
	item.UpdatedAt = service.now()
	if err := service.libraries.Save(ctx, item); err != nil {
		return dto.LibraryDTO{}, err
	}
	return service.buildLibraryDTO(ctx, item)
}

func (service *LibraryService) DeleteLibrary(ctx context.Context, request dto.DeleteLibraryRequest) error {
	libraryID := strings.TrimSpace(request.LibraryID)
	if libraryID == "" {
		return fmt.Errorf("libraryId is required")
	}
	if _, err := service.libraries.Get(ctx, libraryID); err != nil {
		return err
	}
	files, err := service.files.ListByLibraryID(ctx, libraryID)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := deleteLocalFileIfExists(file.Storage.LocalPath); err != nil {
			return err
		}
	}
	return service.libraries.Delete(ctx, libraryID)
}

func (service *LibraryService) GetModuleConfig(ctx context.Context) (dto.LibraryModuleConfigDTO, error) {
	config, err := service.getModuleConfig(ctx)
	if err != nil {
		return dto.LibraryModuleConfigDTO{}, err
	}
	return toModuleConfigDTO(config), nil
}

func (service *LibraryService) GetDefaultModuleConfig(ctx context.Context) (dto.LibraryModuleConfigDTO, error) {
	return toModuleConfigDTO(library.DefaultModuleConfig()), nil
}

func (service *LibraryService) UpdateModuleConfig(ctx context.Context, request dto.UpdateLibraryModuleConfigRequest) (dto.LibraryModuleConfigDTO, error) {
	config := toDomainModuleConfig(request.Config)
	if service == nil || service.moduleConfig == nil {
		return dto.LibraryModuleConfigDTO{}, fmt.Errorf("module config repository not configured")
	}
	if err := service.moduleConfig.Save(ctx, config); err != nil {
		return dto.LibraryModuleConfigDTO{}, err
	}
	return toModuleConfigDTO(config), nil
}

func (service *LibraryService) ListOperations(ctx context.Context, request dto.ListOperationsRequest) ([]dto.OperationListItemDTO, error) {
	var items []library.LibraryOperation
	var err error
	libraryID := strings.TrimSpace(request.LibraryID)
	if libraryID != "" {
		items, err = service.operations.ListByLibraryID(ctx, libraryID)
	} else {
		items, err = service.operations.List(ctx)
	}
	if err != nil {
		return nil, err
	}
	libraryItems, err := service.libraries.List(ctx)
	if err != nil {
		return nil, err
	}
	libraryNames := make(map[string]string, len(libraryItems))
	for _, item := range libraryItems {
		libraryNames[item.ID] = item.Name
	}
	statuses := toLookup(request.Status)
	kinds := toLookup(request.Kinds)
	query := strings.ToLower(strings.TrimSpace(request.Query))
	result := make([]dto.OperationListItemDTO, 0, len(items))
	for _, item := range items {
		if len(statuses) > 0 {
			if _, ok := statuses[strings.ToLower(string(item.Status))]; !ok {
				continue
			}
		}
		if len(kinds) > 0 {
			if _, ok := kinds[strings.ToLower(item.Kind)]; !ok {
				continue
			}
		}
		if query != "" {
			candidate := strings.ToLower(strings.Join([]string{
				item.DisplayName,
				item.Kind,
				item.SourceDomain,
				item.Meta.Platform,
				item.Meta.Uploader,
				libraryNames[item.LibraryID],
			}, " "))
			if !strings.Contains(candidate, query) {
				continue
			}
		}
		result = append(result, toOperationListItemDTO(item, libraryNames[item.LibraryID]))
	}
	return paginateOperationList(result, request.Offset, request.Limit), nil
}

func (service *LibraryService) GetOperation(ctx context.Context, request dto.GetOperationRequest) (dto.LibraryOperationDTO, error) {
	item, err := service.operations.Get(ctx, strings.TrimSpace(request.OperationID))
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	return toOperationDTO(item), nil
}

func (service *LibraryService) CancelOperation(ctx context.Context, request dto.CancelOperationRequest) (dto.LibraryOperationDTO, error) {
	operationID := strings.TrimSpace(request.OperationID)
	if operationID == "" {
		return dto.LibraryOperationDTO{}, fmt.Errorf("operationId is required")
	}
	item, err := service.operations.Get(ctx, operationID)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	if !isResumableSubtitleOperation(item.Kind) {
		return dto.LibraryOperationDTO{}, fmt.Errorf("operation kind %q does not support cancel", item.Kind)
	}
	if item.Status != library.OperationStatusQueued && item.Status != library.OperationStatusRunning {
		return dto.LibraryOperationDTO{}, fmt.Errorf("operation status %q does not support cancel", item.Status)
	}
	item, err = service.markSubtitleOperationCanceled(ctx, item.ID)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	service.cancelOperationRun(item.ID)
	return toOperationDTO(item), nil
}

func (service *LibraryService) ResumeOperation(ctx context.Context, request dto.ResumeOperationRequest) (dto.LibraryOperationDTO, error) {
	operationID := strings.TrimSpace(request.OperationID)
	if operationID == "" {
		return dto.LibraryOperationDTO{}, fmt.Errorf("operationId is required")
	}
	item, err := service.operations.Get(ctx, operationID)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	if !isResumableSubtitleOperation(item.Kind) {
		return dto.LibraryOperationDTO{}, fmt.Errorf("operation kind %q does not support resume", item.Kind)
	}
	if item.Status != library.OperationStatusFailed && item.Status != library.OperationStatusCanceled {
		return dto.LibraryOperationDTO{}, fmt.Errorf("operation status %q does not support resume", item.Status)
	}
	if service.runtime == nil {
		return dto.LibraryOperationDTO{}, fmt.Errorf("subtitle language runtime unavailable")
	}
	now := service.now()
	item.Status = library.OperationStatusQueued
	item.ErrorCode = ""
	item.ErrorMessage = ""
	item.StartedAt = nil
	item.FinishedAt = nil
	item.Progress = buildOperationProgress(
		now,
		progressText("library.status.queued"),
		0,
		progressTotal(item.Progress),
		progressText("library.progressDetail.resumeRequested"),
	)
	item.OutputFiles = nil
	item.Metrics = library.OperationMetrics{}
	item.OutputJSON = buildQueuedSubtitleOperationOutput(item.Kind, item.InputJSON, item.OutputJSON)
	if err := service.operations.Save(ctx, item); err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	operationDTO := toOperationDTO(item)
	service.publishOperationUpdate(operationDTO)
	switch item.Kind {
	case "subtitle_translate":
		resumeRequest := extractSubtitleTranslateRequest(item.InputJSON)
		go service.runSubtitleTranslateOperation(context.Background(), item, resumeRequest)
	case "subtitle_proofread":
		resumeRequest := extractSubtitleProofreadRequest(item.InputJSON)
		go service.runSubtitleProofreadOperation(context.Background(), item, resumeRequest)
	}
	return operationDTO, nil
}

func (service *LibraryService) DeleteOperation(ctx context.Context, request dto.DeleteOperationRequest) error {
	operationID := strings.TrimSpace(request.OperationID)
	if operationID == "" {
		return fmt.Errorf("operationId is required")
	}
	return service.deleteOperation(ctx, operationID, request.CascadeFiles)
}

func (service *LibraryService) DeleteOperations(ctx context.Context, request dto.DeleteOperationsRequest) error {
	operationIDs := normalizeOperationIDs(request.OperationIDs)
	if len(operationIDs) == 0 {
		return fmt.Errorf("operationIds is required")
	}
	for _, operationID := range operationIDs {
		if err := service.deleteOperation(ctx, operationID, request.CascadeFiles); err != nil {
			return err
		}
	}
	return nil
}

func (service *LibraryService) deleteOperation(ctx context.Context, operationID string, cascadeFiles bool) error {
	item, err := service.operations.Get(ctx, operationID)
	if err != nil {
		return err
	}
	if cascadeFiles {
		seenFileIDs := make(map[string]struct{}, len(item.OutputFiles))
		for _, output := range item.OutputFiles {
			fileID := strings.TrimSpace(output.FileID)
			if fileID == "" {
				continue
			}
			if _, exists := seenFileIDs[fileID]; exists {
				continue
			}
			seenFileIDs[fileID] = struct{}{}
			fileItem, getErr := service.files.Get(ctx, fileID)
			if getErr != nil {
				if getErr == library.ErrFileNotFound {
					continue
				}
				return getErr
			}
			if err := service.markLibraryFileDeleted(ctx, fileItem, true); err != nil {
				return err
			}
		}
	}
	if service.histories != nil {
		if err := service.histories.DeleteByOperationID(ctx, operationID); err != nil {
			return err
		}
	}
	if service.operationChunks != nil {
		if err := service.operationChunks.DeleteByOperationID(ctx, operationID); err != nil {
			return err
		}
	}
	if err := service.operations.Delete(ctx, operationID); err != nil {
		return err
	}
	if err := service.touchLibrary(ctx, item.LibraryID, service.now()); err != nil {
		return err
	}
	service.publishOperationDelete(operationID)
	return nil
}

func normalizeOperationIDs(operationIDs []string) []string {
	if len(operationIDs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(operationIDs))
	result := make([]string, 0, len(operationIDs))
	for _, operationID := range operationIDs {
		trimmed := strings.TrimSpace(operationID)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func (service *LibraryService) DeleteFile(ctx context.Context, request dto.DeleteFileRequest) error {
	fileID := strings.TrimSpace(request.FileID)
	if fileID == "" {
		return fmt.Errorf("fileId is required")
	}
	return service.deleteFile(ctx, fileID, request.DeleteFiles)
}

func (service *LibraryService) DeleteFiles(ctx context.Context, request dto.DeleteFilesRequest) error {
	fileIDs := normalizeFileIDs(request.FileIDs)
	if len(fileIDs) == 0 {
		return fmt.Errorf("fileIds is required")
	}
	for _, fileID := range fileIDs {
		if err := service.deleteFile(ctx, fileID, request.DeleteFiles); err != nil {
			return err
		}
	}
	return nil
}

func (service *LibraryService) deleteFile(ctx context.Context, fileID string, deleteFiles bool) error {
	item, err := service.files.Get(ctx, fileID)
	if err != nil {
		return err
	}
	return service.markLibraryFileDeleted(ctx, item, deleteFiles)
}

func normalizeFileIDs(fileIDs []string) []string {
	if len(fileIDs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(fileIDs))
	result := make([]string, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		trimmed := strings.TrimSpace(fileID)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func (service *LibraryService) ListLibraryHistory(ctx context.Context, request dto.ListLibraryHistoryRequest) ([]dto.LibraryHistoryRecordDTO, error) {
	items, err := service.histories.ListByLibraryID(ctx, strings.TrimSpace(request.LibraryID))
	if err != nil {
		return nil, err
	}
	categories := toLookup(request.Categories)
	actions := toLookup(request.Actions)
	result := make([]dto.LibraryHistoryRecordDTO, 0, len(items))
	for _, item := range items {
		if len(categories) > 0 {
			if _, ok := categories[strings.ToLower(item.Category)]; !ok {
				continue
			}
		}
		if len(actions) > 0 {
			if _, ok := actions[strings.ToLower(item.Action)]; !ok {
				continue
			}
		}
		result = append(result, toHistoryDTO(item))
	}
	return paginateHistory(result, request.Offset, request.Limit), nil
}

func (service *LibraryService) ListFileEvents(ctx context.Context, request dto.ListFileEventsRequest) ([]dto.FileEventRecordDTO, error) {
	items, err := service.fileEvents.ListByLibraryID(ctx, strings.TrimSpace(request.LibraryID))
	if err != nil {
		return nil, err
	}
	result := make([]dto.FileEventRecordDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toFileEventDTO(item))
	}
	return paginateFileEvents(result, request.Offset, request.Limit), nil
}

func (service *LibraryService) SaveWorkspaceState(ctx context.Context, request dto.SaveWorkspaceStateRequest) (dto.WorkspaceStateRecordDTO, error) {
	libraryID := strings.TrimSpace(request.LibraryID)
	if libraryID == "" {
		return dto.WorkspaceStateRecordDTO{}, fmt.Errorf("libraryId is required")
	}
	stateJSON := strings.TrimSpace(request.StateJSON)
	if stateJSON == "" {
		return dto.WorkspaceStateRecordDTO{}, fmt.Errorf("stateJson is required")
	}
	version := 1
	if head, err := service.workspace.GetHeadByLibraryID(ctx, libraryID); err == nil {
		version = head.StateVersion + 1
	} else if err != nil && err != library.ErrWorkspaceStateNotFound {
		return dto.WorkspaceStateRecordDTO{}, err
	}
	now := service.now()
	item, err := library.NewWorkspaceStateRecord(library.WorkspaceStateRecordParams{
		ID:           uuid.NewString(),
		LibraryID:    libraryID,
		StateVersion: version,
		StateJSON:    stateJSON,
		OperationID:  strings.TrimSpace(request.OperationID),
		CreatedAt:    &now,
	})
	if err != nil {
		return dto.WorkspaceStateRecordDTO{}, err
	}
	if err := service.workspace.Save(ctx, item); err != nil {
		return dto.WorkspaceStateRecordDTO{}, err
	}
	if err := service.touchLibrary(ctx, libraryID, now); err != nil {
		return dto.WorkspaceStateRecordDTO{}, err
	}
	result := toWorkspaceDTO(item)
	service.publishWorkspaceUpdate(result)
	return result, nil
}

func (service *LibraryService) GetWorkspaceState(ctx context.Context, request dto.GetWorkspaceStateRequest) (dto.WorkspaceStateRecordDTO, error) {
	item, err := service.workspace.GetHeadByLibraryID(ctx, strings.TrimSpace(request.LibraryID))
	if err != nil {
		if err == library.ErrWorkspaceStateNotFound {
			return dto.WorkspaceStateRecordDTO{
				LibraryID: strings.TrimSpace(request.LibraryID),
			}, nil
		}
		return dto.WorkspaceStateRecordDTO{}, err
	}
	return toWorkspaceDTO(item), nil
}

func (service *LibraryService) OpenFileLocation(_ context.Context, request dto.OpenFileLocationRequest) error {
	fileID := strings.TrimSpace(request.FileID)
	if fileID == "" {
		return fmt.Errorf("fileId is required")
	}
	item, err := service.files.Get(context.Background(), fileID)
	if err != nil {
		return err
	}
	path := strings.TrimSpace(item.Storage.LocalPath)
	if path == "" {
		return fmt.Errorf("file has no local path")
	}
	cleaned := filepath.Clean(path)
	if info, statErr := os.Stat(cleaned); statErr == nil {
		if info.IsDir() {
			return opener.OpenDirectory(cleaned)
		}
		return opener.OpenDirectory(filepath.Dir(cleaned))
	}
	return opener.OpenDirectory(filepath.Dir(cleaned))
}

func (service *LibraryService) OpenPath(_ context.Context, request dto.OpenPathRequest) error {
	path := strings.TrimSpace(request.Path)
	if path == "" {
		return fmt.Errorf("path is required")
	}
	cleaned := filepath.Clean(path)
	if info, err := os.Stat(cleaned); err == nil {
		if info.IsDir() {
			return opener.OpenDirectory(cleaned)
		}
		return opener.OpenDirectory(filepath.Dir(cleaned))
	}
	return opener.OpenDirectory(filepath.Dir(cleaned))
}

func (service *LibraryService) CreateYTDLPJob(ctx context.Context, request dto.CreateYTDLPJobRequest) (dto.LibraryOperationDTO, error) {
	operation, history, _, err := service.createDownloadOperation(ctx, request)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	go service.runYTDLPOperation(context.Background(), operation, history, withYTDLPOperationLibrary(request, operation))
	return toOperationDTO(operation), nil
}

func (service *LibraryService) CreateVideoImport(ctx context.Context, request dto.CreateVideoImportRequest) (dto.LibraryFileDTO, error) {
	resolvedPath, err := service.resolveInputPath(ctx, request.Path, request.Source, false)
	if err != nil {
		return dto.LibraryFileDTO{}, err
	}
	name := strings.TrimSpace(request.Title)
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(resolvedPath), filepath.Ext(resolvedPath))
	}
	libraryItem, err := service.ensureLibrary(ctx, ensureLibraryParams{
		LibraryID:       request.LibraryID,
		FallbackName:    deriveLibraryName(name, resolvedPath),
		CreatedBySource: "import_video",
	})
	if err != nil {
		return dto.LibraryFileDTO{}, err
	}
	fileItem, history, eventRecord, _, err := service.createImportFile(ctx, importFileParams{
		LibraryID:      libraryItem.ID,
		Path:           resolvedPath,
		Name:           name,
		Kind:           string(library.FileKindVideo),
		Source:         request.Source,
		SessionRunID:   request.RunID,
		KeepSourceFile: true,
		Action:         "import_video",
	})
	if err != nil {
		return dto.LibraryFileDTO{}, err
	}
	service.publishFileUpdate(service.mustBuildFileDTO(ctx, fileItem))
	service.publishHistoryUpdate(toHistoryDTO(history))
	service.publishFileEventUpdate(toFileEventDTO(eventRecord))
	return service.mustBuildFileDTO(ctx, fileItem), nil
}

func (service *LibraryService) CreateSubtitleImport(ctx context.Context, request dto.CreateSubtitleImportRequest) (dto.LibraryFileDTO, error) {
	resolvedPath, err := service.resolveInputPath(ctx, request.Path, request.Source, false)
	if err != nil {
		return dto.LibraryFileDTO{}, err
	}
	name := strings.TrimSpace(request.Title)
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(resolvedPath), filepath.Ext(resolvedPath))
	}
	libraryItem, err := service.ensureLibrary(ctx, ensureLibraryParams{
		LibraryID:       request.LibraryID,
		FallbackName:    deriveLibraryName(name, resolvedPath),
		CreatedBySource: "import_subtitle",
	})
	if err != nil {
		return dto.LibraryFileDTO{}, err
	}
	fileItem, history, eventRecord, _, err := service.createImportFile(ctx, importFileParams{
		LibraryID:      libraryItem.ID,
		Path:           resolvedPath,
		Name:           name,
		Kind:           string(library.FileKindSubtitle),
		Source:         request.Source,
		SessionRunID:   request.RunID,
		KeepSourceFile: true,
		Action:         "import_subtitle",
	})
	if err != nil {
		return dto.LibraryFileDTO{}, err
	}
	service.publishFileUpdate(service.mustBuildFileDTO(ctx, fileItem))
	service.publishHistoryUpdate(toHistoryDTO(history))
	service.publishFileEventUpdate(toFileEventDTO(eventRecord))
	return service.mustBuildFileDTO(ctx, fileItem), nil
}

func (service *LibraryService) CreateTranscodeJob(ctx context.Context, request dto.CreateTranscodeJobRequest) (dto.LibraryOperationDTO, error) {
	sourceFile, err := service.resolveSourceFileForTranscode(ctx, request)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	if sourceFile.LibraryID == "" {
		return dto.LibraryOperationDTO{}, fmt.Errorf("source file is not attached to a library")
	}
	plan, err := service.resolveTranscodePlanWithoutProbe(ctx, request, sourceFile.Storage.LocalPath)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	now := service.now()
	operationID := uuid.NewString()
	displayName := resolveTranscodeTitle(request, sourceFile.Storage.LocalPath, plan.preset)
	inputJSON, err := json.Marshal(request)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	operation, err := library.NewLibraryOperation(library.LibraryOperationParams{
		ID:          operationID,
		LibraryID:   sourceFile.LibraryID,
		Kind:        "transcode",
		Status:      string(library.OperationStatusQueued),
		DisplayName: displayName,
		Correlation: library.OperationCorrelation{RunID: strings.TrimSpace(request.RunID)},
		InputJSON:   string(inputJSON),
		OutputJSON:  buildTranscodeOperationOutput(request, "queued", ""),
		Progress: buildOperationProgress(
			now,
			progressText("library.status.queued"),
			0,
			1,
			progressText("library.progressDetail.ffmpegTranscodeQueued"),
		),
		CreatedAt: &now,
	})
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	if err := service.operations.Save(ctx, operation); err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	operationDTO := toOperationDTO(operation)
	service.publishOperationUpdate(operationDTO)
	go service.runTranscodeOperation(context.Background(), operation, request)
	return operationDTO, nil
}

func (service *LibraryService) CreateSubtitleTranslateJob(ctx context.Context, request dto.SubtitleTranslateRequest) (dto.LibraryOperationDTO, error) {
	request = normalizeSubtitleTranslateRequest(request)
	if err := validateSubtitleTranslateSourceAndTarget(request); err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	sourceFile, _, err := service.resolveSubtitleFileAndDocument(ctx, request.FileID, request.DocumentID, request.Path)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	if strings.TrimSpace(sourceFile.LibraryID) == "" {
		return dto.LibraryOperationDTO{}, fmt.Errorf("source file is not attached to a library")
	}
	if service.runtime == nil {
		return dto.LibraryOperationDTO{}, fmt.Errorf("subtitle translate runtime unavailable")
	}
	targetLanguage := request.TargetLanguage

	now := service.now()
	operationID := uuid.NewString()
	inputJSON, err := json.Marshal(request)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	operation, err := library.NewLibraryOperation(library.LibraryOperationParams{
		ID:          operationID,
		LibraryID:   sourceFile.LibraryID,
		Kind:        "subtitle_translate",
		Status:      string(library.OperationStatusQueued),
		DisplayName: buildSubtitleTranslateOutputName(sourceFile.Name, targetLanguage),
		Correlation: library.OperationCorrelation{RunID: strings.TrimSpace(request.RunID)},
		InputJSON:   string(inputJSON),
		OutputJSON:  marshalJSON(buildSubtitleTranslateOutput(request, "queued", 0, 0, "", "", "", runtimedto.RuntimeUsage{}, subtitleTaskRunState{})),
		Progress: buildOperationProgress(
			now,
			progressText("library.status.queued"),
			0,
			0,
			progressText("library.progressDetail.subtitleTranslationQueued"),
		),
		CreatedAt: &now,
	})
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	if err := service.operations.Save(ctx, operation); err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	operationDTO := toOperationDTO(operation)
	service.publishOperationUpdate(operationDTO)
	go service.runSubtitleTranslateOperation(context.Background(), operation, request)
	return operationDTO, nil
}

func (service *LibraryService) CreateSubtitleProofreadJob(ctx context.Context, request dto.SubtitleProofreadRequest) (dto.LibraryOperationDTO, error) {
	request = normalizeSubtitleProofreadRequest(request)
	sourceFile, _, err := service.resolveSubtitleFileAndDocument(ctx, request.FileID, request.DocumentID, request.Path)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	if strings.TrimSpace(sourceFile.LibraryID) == "" {
		return dto.LibraryOperationDTO{}, fmt.Errorf("source file is not attached to a library")
	}
	if service.runtime == nil {
		return dto.LibraryOperationDTO{}, fmt.Errorf("subtitle proofread runtime unavailable")
	}

	now := service.now()
	operationID := uuid.NewString()
	inputJSON, err := json.Marshal(request)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	operation, err := library.NewLibraryOperation(library.LibraryOperationParams{
		ID:          operationID,
		LibraryID:   sourceFile.LibraryID,
		Kind:        "subtitle_proofread",
		Status:      string(library.OperationStatusQueued),
		DisplayName: buildSubtitleProofreadOutputName(sourceFile.Name),
		Correlation: library.OperationCorrelation{RunID: strings.TrimSpace(request.RunID)},
		InputJSON:   string(inputJSON),
		OutputJSON:  marshalJSON(buildSubtitleProofreadOutput(request, "queued", 0, 0, "", "", "", "", "", "", 0, runtimedto.RuntimeUsage{}, subtitleTaskRunState{})),
		Progress: buildOperationProgress(
			now,
			progressText("library.status.queued"),
			0,
			0,
			progressText("library.progressDetail.subtitleProofreadQueued"),
		),
		CreatedAt: &now,
	})
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	if err := service.operations.Save(ctx, operation); err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	operationDTO := toOperationDTO(operation)
	service.publishOperationUpdate(operationDTO)
	go service.runSubtitleProofreadOperation(context.Background(), operation, request)
	return operationDTO, nil
}

type ensureLibraryParams struct {
	LibraryID          string
	FallbackName       string
	InitialNameFromID  bool
	CreatedBySource    string
	TriggerOperationID string
}

func (service *LibraryService) ensureLibrary(ctx context.Context, params ensureLibraryParams) (library.Library, error) {
	trimmedID := strings.TrimSpace(params.LibraryID)
	if trimmedID != "" {
		return service.libraries.Get(ctx, trimmedID)
	}
	now := service.now()
	libraryID := uuid.NewString()
	item, err := library.NewLibrary(library.LibraryParams{
		ID:   libraryID,
		Name: resolveInitialLibraryName(libraryID, params.FallbackName, params.InitialNameFromID),
		CreatedBy: library.CreateMeta{
			Source:             strings.TrimSpace(params.CreatedBySource),
			TriggerOperationID: strings.TrimSpace(params.TriggerOperationID),
		},
		CreatedAt: &now,
		UpdatedAt: &now,
	})
	if err != nil {
		return library.Library{}, err
	}
	if err := service.libraries.Save(ctx, item); err != nil {
		return library.Library{}, err
	}
	return item, nil
}

func (service *LibraryService) touchLibrary(ctx context.Context, libraryID string, updatedAt time.Time) error {
	if strings.TrimSpace(libraryID) == "" {
		return nil
	}
	item, err := service.libraries.Get(ctx, libraryID)
	if err != nil {
		return err
	}
	item.UpdatedAt = updatedAt
	return service.libraries.Save(ctx, item)
}

func (service *LibraryService) renameLibraryFromFirstFileIfNeeded(ctx context.Context, libraryID string, fileName string, updatedAt time.Time) error {
	trimmedLibraryID := strings.TrimSpace(libraryID)
	if trimmedLibraryID == "" {
		return nil
	}
	nextName := resolveLibraryNameFromFile(fileName)
	if nextName == "" {
		return nil
	}
	item, err := service.libraries.Get(ctx, trimmedLibraryID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(item.Name) != strings.TrimSpace(item.ID) {
		return nil
	}
	files, err := service.files.ListByLibraryID(ctx, trimmedLibraryID)
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return nil
	}
	item.Name = nextName
	item.UpdatedAt = updatedAt
	return service.libraries.Save(ctx, item)
}

type importFileParams struct {
	LibraryID      string
	Path           string
	Name           string
	Kind           string
	Source         string
	SessionRunID   string
	KeepSourceFile bool
	Action         string
}

func (service *LibraryService) createImportFile(ctx context.Context, params importFileParams) (library.LibraryFile, library.HistoryRecord, library.FileEventRecord, *library.SubtitleDocument, error) {
	now := service.now()
	importedAt := now
	batchID := uuid.NewString()
	storage := library.FileStorage{Mode: "local_path", LocalPath: params.Path}
	var subtitleDoc *library.SubtitleDocument
	mediaInfo := library.MediaInfo{}
	if params.Kind == string(library.FileKindSubtitle) {
		content, err := os.ReadFile(params.Path)
		if err != nil {
			return library.LibraryFile{}, library.HistoryRecord{}, library.FileEventRecord{}, nil, err
		}
		documentID := uuid.NewString()
		storage = library.FileStorage{Mode: "hybrid", LocalPath: params.Path, DocumentID: documentID}
		format := detectSubtitleFormat("", params.Path, "")
		doc, err := library.NewSubtitleDocument(library.SubtitleDocumentParams{
			ID:              documentID,
			FileID:          uuid.NewString(),
			LibraryID:       params.LibraryID,
			Format:          format,
			OriginalContent: string(content),
			WorkingContent:  string(content),
			CreatedAt:       &now,
			UpdatedAt:       &now,
		})
		if err != nil {
			return library.LibraryFile{}, library.HistoryRecord{}, library.FileEventRecord{}, nil, err
		}
		subtitleDoc = &doc
		mediaInfo.Format = format
		sizeValue := int64(len(content))
		mediaInfo.SizeBytes = &sizeValue
	} else {
		media, err := service.probeRequiredMedia(ctx, params.Path)
		if err != nil {
			return library.LibraryFile{}, library.HistoryRecord{}, library.FileEventRecord{}, nil, err
		}
		mediaInfo = media.toMediaInfo()
	}
	fileID := uuid.NewString()
	if subtitleDoc != nil {
		subtitleDoc.FileID = fileID
	}
	fileItem, err := library.NewLibraryFile(library.LibraryFileParams{
		ID:        fileID,
		LibraryID: params.LibraryID,
		Kind:      params.Kind,
		Name:      strings.TrimSpace(params.Name),
		Storage:   storage,
		Origin: library.FileOrigin{
			Kind: "import",
			Import: &library.ImportOrigin{
				BatchID:        batchID,
				ImportPath:     params.Path,
				ImportedAt:     importedAt,
				KeepSourceFile: params.KeepSourceFile,
			},
		},
		Media:     &mediaInfo,
		State:     library.FileState{Status: "active"},
		CreatedAt: &now,
		UpdatedAt: &now,
	})
	if err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, library.FileEventRecord{}, nil, err
	}
	if err := service.files.Save(ctx, fileItem); err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, library.FileEventRecord{}, nil, err
	}
	if subtitleDoc != nil {
		if err := service.subtitles.Save(ctx, *subtitleDoc); err != nil {
			return library.LibraryFile{}, library.HistoryRecord{}, library.FileEventRecord{}, nil, err
		}
	}
	history, err := library.NewHistoryRecord(library.HistoryRecordParams{
		ID:          uuid.NewString(),
		LibraryID:   params.LibraryID,
		Category:    "import",
		Action:      params.Action,
		DisplayName: fileItem.Name,
		Status:      "succeeded",
		Source:      library.HistoryRecordSource{Kind: "import", RunID: strings.TrimSpace(params.SessionRunID)},
		Refs:        library.HistoryRecordRefs{ImportBatchID: batchID, FileIDs: []string{fileItem.ID}},
		Files: []library.OperationOutputFile{{
			FileID:    fileItem.ID,
			Kind:      string(fileItem.Kind),
			Format:    mediaFormatFromFile(fileItem),
			SizeBytes: mediaSizeFromFile(fileItem),
			IsPrimary: true,
			Deleted:   fileItem.State.Deleted,
		}},
		Metrics: buildOperationMetrics([]library.LibraryFile{fileItem}),
		ImportMeta: &library.ImportRecordMeta{
			ImportPath:     params.Path,
			KeepSourceFile: params.KeepSourceFile,
			ImportedAt:     importedAt.Format(time.RFC3339),
		},
		OccurredAt: &now,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	})
	if err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, library.FileEventRecord{}, nil, err
	}
	if err := service.histories.Save(ctx, history); err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, library.FileEventRecord{}, nil, err
	}
	detailJSON := marshalJSON(dto.FileEventDetailDTO{
		Cause:  dto.FileEventCauseDTO{Category: "import", ImportBatchID: batchID},
		After:  &dto.FileEventFileSnapshotDTO{FileID: fileItem.ID, Kind: string(fileItem.Kind), Name: fileItem.Name, LocalPath: fileItem.Storage.LocalPath, DocumentID: fileItem.Storage.DocumentID},
		Import: &dto.LibraryImportOriginDTO{BatchID: batchID, ImportPath: params.Path, ImportedAt: importedAt.Format(time.RFC3339), KeepSourceFile: params.KeepSourceFile},
	})
	eventRecord, err := library.NewFileEventRecord(library.FileEventRecordParams{
		ID:         uuid.NewString(),
		LibraryID:  params.LibraryID,
		FileID:     fileItem.ID,
		EventType:  "file_imported",
		DetailJSON: detailJSON,
		CreatedAt:  &now,
	})
	if err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, library.FileEventRecord{}, nil, err
	}
	if err := service.fileEvents.Save(ctx, eventRecord); err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, library.FileEventRecord{}, nil, err
	}
	if err := service.touchLibrary(ctx, params.LibraryID, now); err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, library.FileEventRecord{}, nil, err
	}
	return fileItem, history, eventRecord, subtitleDoc, nil
}

type derivedFileParams struct {
	LibraryID     string
	RootFileID    string
	Name          string
	Kind          string
	OperationID   string
	OperationKind string
	OutputPath    string
	SourcePath    string
	SourceMedia   *library.MediaInfo
	OccurredAt    time.Time
	HistorySource library.HistoryRecordSource
}

func (service *LibraryService) createDerivedLocalFile(ctx context.Context, params derivedFileParams) (library.LibraryFile, library.HistoryRecord, error) {
	if err := ensureParentDir(params.OutputPath); err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	if err := copyLocalFile(params.SourcePath, params.OutputPath); err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	info, err := os.Stat(params.OutputPath)
	if err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	now := params.OccurredAt
	if now.IsZero() {
		now = service.now()
	}
	probedProbe, err := service.probeRequiredMedia(ctx, params.OutputPath)
	if err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	probedMedia := probedProbe.toMediaInfo()
	media := mergeMediaInfo(cloneMediaInfo(params.SourceMedia), &probedMedia)
	if media == nil {
		media = &library.MediaInfo{}
	}
	if strings.TrimSpace(media.Format) == "" {
		media.Format = normalizeTranscodeFormat(filepath.Ext(params.OutputPath))
	}
	sizeValue := info.Size()
	media.SizeBytes = &sizeValue
	fileItem, err := library.NewLibraryFile(library.LibraryFileParams{
		ID:        uuid.NewString(),
		LibraryID: params.LibraryID,
		Kind:      params.Kind,
		Name:      params.Name,
		Storage:   library.FileStorage{Mode: "local_path", LocalPath: params.OutputPath},
		Origin:    library.FileOrigin{Kind: params.OperationKind, OperationID: params.OperationID},
		Lineage:   library.FileLineage{RootFileID: strings.TrimSpace(params.RootFileID)},
		Media:     media,
		State:     library.FileState{Status: "active"},
		CreatedAt: &now,
		UpdatedAt: &now,
	})
	if err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	if err := service.files.Save(ctx, fileItem); err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	history, err := library.NewHistoryRecord(library.HistoryRecordParams{
		ID:          uuid.NewString(),
		LibraryID:   params.LibraryID,
		Category:    "operation",
		Action:      params.OperationKind,
		DisplayName: params.Name,
		Status:      "succeeded",
		Source:      params.HistorySource,
		Refs:        library.HistoryRecordRefs{OperationID: params.OperationID, FileIDs: []string{fileItem.ID}},
		OccurredAt:  &now,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	return fileItem, history, nil
}

type derivedSubtitleParams struct {
	LibraryID      string
	RootFileID     string
	Name           string
	OperationID    string
	OperationKind  string
	Format         string
	SourceMedia    *library.MediaInfo
	OriginalSource string
	LocalPath      string
	OccurredAt     time.Time
	HistorySource  library.HistoryRecordSource
}

func (service *LibraryService) createDerivedSubtitleFile(ctx context.Context, params derivedSubtitleParams) (library.LibraryFile, library.HistoryRecord, error) {
	now := params.OccurredAt
	if now.IsZero() {
		now = service.now()
	}
	documentID := uuid.NewString()
	media := mergeMediaInfo(cloneMediaInfo(params.SourceMedia), &library.MediaInfo{Format: params.Format})
	if media == nil {
		media = &library.MediaInfo{Format: params.Format}
	}
	fileItem, err := library.NewLibraryFile(library.LibraryFileParams{
		ID:        uuid.NewString(),
		LibraryID: params.LibraryID,
		Kind:      string(library.FileKindSubtitle),
		Name:      params.Name,
		Storage:   library.FileStorage{Mode: "hybrid", LocalPath: params.LocalPath, DocumentID: documentID},
		Origin:    library.FileOrigin{Kind: firstNonEmpty(strings.TrimSpace(params.OperationKind), "subtitle_translate"), OperationID: params.OperationID},
		Lineage:   library.FileLineage{RootFileID: strings.TrimSpace(params.RootFileID)},
		Media:     media,
		State:     library.FileState{Status: "active"},
		CreatedAt: &now,
		UpdatedAt: &now,
	})
	if err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	if err := service.files.Save(ctx, fileItem); err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	document, err := library.NewSubtitleDocument(library.SubtitleDocumentParams{
		ID:              documentID,
		FileID:          fileItem.ID,
		LibraryID:       params.LibraryID,
		Format:          params.Format,
		OriginalContent: params.OriginalSource,
		WorkingContent:  params.OriginalSource,
		CreatedAt:       &now,
		UpdatedAt:       &now,
	})
	if err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	if err := service.subtitles.Save(ctx, document); err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	sizeValue := int64(len(params.OriginalSource))
	fileItem.Media = mergeMediaInfo(cloneMediaInfo(fileItem.Media), &library.MediaInfo{Format: params.Format, SizeBytes: &sizeValue})
	history, err := library.NewHistoryRecord(library.HistoryRecordParams{
		ID:          uuid.NewString(),
		LibraryID:   params.LibraryID,
		Category:    "operation",
		Action:      firstNonEmpty(strings.TrimSpace(params.OperationKind), "subtitle_translate"),
		DisplayName: params.Name,
		Status:      "succeeded",
		Source:      params.HistorySource,
		Refs:        library.HistoryRecordRefs{OperationID: params.OperationID, FileIDs: []string{fileItem.ID}},
		OccurredAt:  &now,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		return library.LibraryFile{}, library.HistoryRecord{}, err
	}
	return fileItem, history, nil
}

func (service *LibraryService) resolveSourceFileForTranscode(ctx context.Context, request dto.CreateTranscodeJobRequest) (library.LibraryFile, error) {
	if fileID := strings.TrimSpace(request.FileID); fileID != "" {
		return service.files.Get(ctx, fileID)
	}
	path := strings.TrimSpace(request.InputPath)
	if path == "" {
		return library.LibraryFile{}, fmt.Errorf("fileId or inputPath is required")
	}
	imported, _, _, _, err := service.createImportFile(ctx, importFileParams{
		LibraryID:      strings.TrimSpace(request.LibraryID),
		Path:           path,
		Name:           strings.TrimSpace(request.Title),
		Kind:           string(library.FileKindVideo),
		Source:         request.Source,
		SessionRunID:   request.RunID,
		KeepSourceFile: true,
		Action:         "import_video",
	})
	return imported, err
}

func (service *LibraryService) resolveSubtitleFileAndDocument(ctx context.Context, fileID string, documentID string, path string) (library.LibraryFile, library.SubtitleDocument, error) {
	if trimmed := strings.TrimSpace(documentID); trimmed != "" {
		doc, err := service.subtitles.Get(ctx, trimmed)
		if err != nil {
			return library.LibraryFile{}, library.SubtitleDocument{}, err
		}
		fileItem, err := service.files.Get(ctx, doc.FileID)
		if err != nil {
			return library.LibraryFile{}, library.SubtitleDocument{}, err
		}
		return fileItem, doc, nil
	}
	if trimmed := strings.TrimSpace(fileID); trimmed != "" {
		fileItem, err := service.files.Get(ctx, trimmed)
		if err != nil {
			return library.LibraryFile{}, library.SubtitleDocument{}, err
		}
		doc, err := service.subtitles.GetByFileID(ctx, trimmed)
		if err != nil {
			return library.LibraryFile{}, library.SubtitleDocument{}, err
		}
		return fileItem, doc, nil
	}
	if strings.TrimSpace(path) != "" {
		return library.LibraryFile{}, library.SubtitleDocument{}, fmt.Errorf("subtitle processing requires documentId or fileId; path is only allowed for import")
	}
	return library.LibraryFile{}, library.SubtitleDocument{}, fmt.Errorf("documentId or fileId is required")
}

func (service *LibraryService) buildLibraryDTO(ctx context.Context, item library.Library) (dto.LibraryDTO, error) {
	files, err := service.files.ListByLibraryID(ctx, item.ID)
	if err != nil {
		return dto.LibraryDTO{}, err
	}
	history, err := service.histories.ListByLibraryID(ctx, item.ID)
	if err != nil {
		return dto.LibraryDTO{}, err
	}
	workspaceStates, err := service.workspace.ListByLibraryID(ctx, item.ID)
	if err != nil {
		return dto.LibraryDTO{}, err
	}
	var workspaceHead *dto.WorkspaceStateRecordDTO
	if head, headErr := service.workspace.GetHeadByLibraryID(ctx, item.ID); headErr == nil {
		mapped := toWorkspaceDTO(head)
		workspaceHead = &mapped
	} else if headErr != nil && headErr != library.ErrWorkspaceStateNotFound {
		return dto.LibraryDTO{}, headErr
	}
	fileEvents, err := service.fileEvents.ListByLibraryID(ctx, item.ID)
	if err != nil {
		return dto.LibraryDTO{}, err
	}
	moduleConfig, err := service.getModuleConfig(ctx)
	if err != nil {
		return dto.LibraryDTO{}, err
	}
	fileDTOs := make([]dto.LibraryFileDTO, 0, len(files))
	for _, fileItem := range files {
		fileDTO, buildErr := service.buildFileDTOWithConfig(ctx, fileItem, moduleConfig)
		if buildErr != nil {
			fileDTO = toLibraryFileDTO(fileItem)
		}
		fileDTOs = append(fileDTOs, fileDTO)
	}
	historyDTOs := make([]dto.LibraryHistoryRecordDTO, 0, len(history))
	for _, record := range history {
		historyDTOs = append(historyDTOs, toHistoryDTO(record))
	}
	workspaceDTOs := make([]dto.WorkspaceStateRecordDTO, 0, len(workspaceStates))
	for _, state := range workspaceStates {
		workspaceDTOs = append(workspaceDTOs, toWorkspaceDTO(state))
	}
	fileEventDTOs := make([]dto.FileEventRecordDTO, 0, len(fileEvents))
	for _, eventRecord := range fileEvents {
		fileEventDTOs = append(fileEventDTOs, toFileEventDTO(eventRecord))
	}
	return dto.LibraryDTO{
		Version:   dto.LibrarySchemaVersion,
		ID:        item.ID,
		Name:      item.Name,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
		UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
		CreatedBy: dto.LibraryCreateMetaDTO{
			Source:             item.CreatedBy.Source,
			TriggerOperationID: item.CreatedBy.TriggerOperationID,
			ImportBatchID:      item.CreatedBy.ImportBatchID,
			Actor:              item.CreatedBy.Actor,
		},
		Files: fileDTOs,
		Records: dto.LibraryRecordsDTO{
			History:            historyDTOs,
			WorkspaceStateHead: workspaceHead,
			WorkspaceStates:    workspaceDTOs,
			FileEvents:         fileEventDTOs,
		},
	}, nil
}

func toModuleConfigDTO(config library.ModuleConfig) dto.LibraryModuleConfigDTO {
	return dto.LibraryModuleConfigDTO{
		Workspace: dto.LibraryWorkspaceConfigDTO{FastReadLatestState: config.Workspace.FastReadLatestState},
		TranslateLanguages: dto.LibraryTranslateLanguagesConfigDTO{
			Builtin: toTranslateLanguageDTOs(config.TranslateLanguages.Builtin),
			Custom:  toTranslateLanguageDTOs(config.TranslateLanguages.Custom),
		},
		LanguageAssets: dto.LibraryLanguageAssetsConfigDTO{
			GlossaryProfiles: toGlossaryProfileDTOs(config.LanguageAssets.GlossaryProfiles),
			PromptProfiles:   toPromptProfileDTOs(config.LanguageAssets.PromptProfiles),
		},
		SubtitleStyles: toSubtitleStyleConfigDTO(config.SubtitleStyles),
		TaskRuntime: dto.LibraryTaskRuntimeConfigDTO{
			Translate: dto.LibraryTaskRuntimeSettingsDTO{
				StructuredOutputMode: config.TaskRuntime.Translate.StructuredOutputMode,
				ThinkingMode:         config.TaskRuntime.Translate.ThinkingMode,
				MaxTokensFloor:       config.TaskRuntime.Translate.MaxTokensFloor,
				MaxTokensCeiling:     config.TaskRuntime.Translate.MaxTokensCeiling,
				RetryTokenStep:       config.TaskRuntime.Translate.RetryTokenStep,
			},
			Proofread: dto.LibraryTaskRuntimeSettingsDTO{
				StructuredOutputMode: config.TaskRuntime.Proofread.StructuredOutputMode,
				ThinkingMode:         config.TaskRuntime.Proofread.ThinkingMode,
				MaxTokensFloor:       config.TaskRuntime.Proofread.MaxTokensFloor,
				MaxTokensCeiling:     config.TaskRuntime.Proofread.MaxTokensCeiling,
				RetryTokenStep:       config.TaskRuntime.Proofread.RetryTokenStep,
			},
		},
	}
}

func toDomainModuleConfig(config dto.LibraryModuleConfigDTO) library.ModuleConfig {
	result := library.DefaultModuleConfig()
	result.Workspace.FastReadLatestState = config.Workspace.FastReadLatestState
	if len(config.TranslateLanguages.Builtin) > 0 {
		result.TranslateLanguages.Builtin = toTranslateLanguages(config.TranslateLanguages.Builtin)
	}
	result.TranslateLanguages.Custom = toTranslateLanguages(config.TranslateLanguages.Custom)
	result.LanguageAssets = library.LanguageAssetsConfig{
		GlossaryProfiles: toGlossaryProfiles(config.LanguageAssets.GlossaryProfiles),
		PromptProfiles:   toPromptProfiles(config.LanguageAssets.PromptProfiles),
	}
	result.SubtitleStyles = toSubtitleStyleConfig(config.SubtitleStyles)
	result.TaskRuntime = library.LanguageTaskRuntimeConfig{
		Translate: library.LanguageTaskRuntimeSettings{
			StructuredOutputMode: strings.TrimSpace(config.TaskRuntime.Translate.StructuredOutputMode),
			ThinkingMode:         strings.TrimSpace(config.TaskRuntime.Translate.ThinkingMode),
			MaxTokensFloor:       config.TaskRuntime.Translate.MaxTokensFloor,
			MaxTokensCeiling:     config.TaskRuntime.Translate.MaxTokensCeiling,
			RetryTokenStep:       config.TaskRuntime.Translate.RetryTokenStep,
		},
		Proofread: library.LanguageTaskRuntimeSettings{
			StructuredOutputMode: strings.TrimSpace(config.TaskRuntime.Proofread.StructuredOutputMode),
			ThinkingMode:         strings.TrimSpace(config.TaskRuntime.Proofread.ThinkingMode),
			MaxTokensFloor:       config.TaskRuntime.Proofread.MaxTokensFloor,
			MaxTokensCeiling:     config.TaskRuntime.Proofread.MaxTokensCeiling,
			RetryTokenStep:       config.TaskRuntime.Proofread.RetryTokenStep,
		},
	}
	return library.NormalizeModuleConfig(result)
}

func toTranslateLanguageDTOs(values []library.TranslateLanguage) []dto.LibraryTranslateLanguageDTO {
	result := make([]dto.LibraryTranslateLanguageDTO, 0, len(values))
	for _, value := range values {
		result = append(result, dto.LibraryTranslateLanguageDTO{
			Code:    value.Code,
			Label:   value.Label,
			Aliases: append([]string(nil), value.Aliases...),
		})
	}
	return result
}

func toTranslateLanguages(values []dto.LibraryTranslateLanguageDTO) []library.TranslateLanguage {
	result := make([]library.TranslateLanguage, 0, len(values))
	for _, value := range values {
		result = append(result, library.TranslateLanguage{
			Code:    strings.TrimSpace(value.Code),
			Label:   strings.TrimSpace(value.Label),
			Aliases: append([]string(nil), value.Aliases...),
		})
	}
	return result
}

func toGlossaryProfileDTOs(values []library.GlossaryProfile) []dto.LibraryGlossaryProfileDTO {
	result := make([]dto.LibraryGlossaryProfileDTO, 0, len(values))
	for _, value := range values {
		terms := make([]dto.LibraryGlossaryTermDTO, 0, len(value.Terms))
		for _, term := range value.Terms {
			terms = append(terms, dto.LibraryGlossaryTermDTO{
				Source: term.Source,
				Target: term.Target,
				Note:   term.Note,
			})
		}
		result = append(result, dto.LibraryGlossaryProfileDTO{
			ID:             value.ID,
			Name:           value.Name,
			Category:       value.Category,
			Description:    value.Description,
			SourceLanguage: value.SourceLanguage,
			TargetLanguage: value.TargetLanguage,
			Terms:          terms,
		})
	}
	return result
}

func toGlossaryProfiles(values []dto.LibraryGlossaryProfileDTO) []library.GlossaryProfile {
	result := make([]library.GlossaryProfile, 0, len(values))
	for _, value := range values {
		terms := make([]library.GlossaryTerm, 0, len(value.Terms))
		for _, term := range value.Terms {
			terms = append(terms, library.GlossaryTerm{
				Source: strings.TrimSpace(term.Source),
				Target: strings.TrimSpace(term.Target),
				Note:   strings.TrimSpace(term.Note),
			})
		}
		result = append(result, library.GlossaryProfile{
			ID:             strings.TrimSpace(value.ID),
			Name:           strings.TrimSpace(value.Name),
			Category:       strings.TrimSpace(value.Category),
			Description:    strings.TrimSpace(value.Description),
			SourceLanguage: strings.TrimSpace(value.SourceLanguage),
			TargetLanguage: strings.TrimSpace(value.TargetLanguage),
			Terms:          terms,
		})
	}
	return result
}

func toPromptProfileDTOs(values []library.PromptProfile) []dto.LibraryPromptProfileDTO {
	result := make([]dto.LibraryPromptProfileDTO, 0, len(values))
	for _, value := range values {
		result = append(result, dto.LibraryPromptProfileDTO{
			ID:          value.ID,
			Name:        value.Name,
			Category:    value.Category,
			Description: value.Description,
			Prompt:      value.Prompt,
		})
	}
	return result
}

func toPromptProfiles(values []dto.LibraryPromptProfileDTO) []library.PromptProfile {
	result := make([]library.PromptProfile, 0, len(values))
	for _, value := range values {
		result = append(result, library.PromptProfile{
			ID:          strings.TrimSpace(value.ID),
			Name:        strings.TrimSpace(value.Name),
			Category:    strings.TrimSpace(value.Category),
			Description: strings.TrimSpace(value.Description),
			Prompt:      strings.TrimSpace(value.Prompt),
		})
	}
	return result
}

func toSubtitleStyleConfigDTO(config library.SubtitleStyleConfig) dto.LibrarySubtitleStyleConfigDTO {
	return dto.LibrarySubtitleStyleConfigDTO{
		MonoStyles:            toMonoStyleDTOs(config.MonoStyles),
		BilingualStyles:       toBilingualStyleDTOs(config.BilingualStyles),
		Sources:               toSubtitleStyleSourceDTOs(config.Sources),
		Fonts:                 toSubtitleStyleFontDTOs(config.Fonts),
		SubtitleExportPresets: toSubtitleExportPresetDTOs(config.SubtitleExportPresets),
		Defaults: dto.LibrarySubtitleStyleDefaultsDTO{
			MonoStyleID:            config.Defaults.MonoStyleID,
			BilingualStyleID:       config.Defaults.BilingualStyleID,
			SubtitleExportPresetID: config.Defaults.SubtitleExportPresetID,
		},
	}
}

func toSubtitleStyleConfig(config dto.LibrarySubtitleStyleConfigDTO) library.SubtitleStyleConfig {
	return library.SubtitleStyleConfig{
		MonoStyles:            toMonoStyles(config.MonoStyles),
		BilingualStyles:       toBilingualStyles(config.BilingualStyles),
		Sources:               toSubtitleStyleSources(config.Sources),
		Fonts:                 toSubtitleStyleFonts(config.Fonts),
		SubtitleExportPresets: toSubtitleExportPresets(config.SubtitleExportPresets),
		Defaults: library.SubtitleStyleDefaults{
			MonoStyleID:            strings.TrimSpace(config.Defaults.MonoStyleID),
			BilingualStyleID:       strings.TrimSpace(config.Defaults.BilingualStyleID),
			SubtitleExportPresetID: strings.TrimSpace(config.Defaults.SubtitleExportPresetID),
		},
	}
}

func toSubtitleStyleDocumentDTOs(values []library.SubtitleStyleDocument) []dto.LibrarySubtitleStyleDocumentDTO {
	result := make([]dto.LibrarySubtitleStyleDocumentDTO, 0, len(values))
	for _, value := range values {
		analysis := library.AnalyzeSubtitleStyleDocument(value.Content)
		result = append(result, dto.LibrarySubtitleStyleDocumentDTO{
			ID:          value.ID,
			Name:        value.Name,
			Description: value.Description,
			Source:      value.Source,
			SourceRef:   value.SourceRef,
			Version:     value.Version,
			Enabled:     value.Enabled,
			Format:      value.Format,
			Content:     value.Content,
			Analysis: dto.LibrarySubtitleStyleDocumentAnalysisDTO{
				DetectedFormat:   analysis.DetectedFormat,
				ScriptType:       analysis.ScriptType,
				PlayResX:         analysis.PlayResX,
				PlayResY:         analysis.PlayResY,
				StyleCount:       analysis.StyleCount,
				DialogueCount:    analysis.DialogueCount,
				CommentCount:     analysis.CommentCount,
				StyleNames:       append([]string(nil), analysis.StyleNames...),
				Fonts:            append([]string(nil), analysis.Fonts...),
				FeatureFlags:     append([]string(nil), analysis.FeatureFlags...),
				ValidationIssues: append([]string(nil), analysis.ValidationIssues...),
			},
		})
	}
	return result
}

func toMonoStyleDTOs(values []library.MonoStyle) []dto.LibraryMonoStyleDTO {
	result := make([]dto.LibraryMonoStyleDTO, 0, len(values))
	for _, value := range values {
		result = append(result, dto.LibraryMonoStyleDTO{
			ID:                 value.ID,
			Name:               value.Name,
			BuiltIn:            value.BuiltIn,
			BasePlayResX:       value.BasePlayResX,
			BasePlayResY:       value.BasePlayResY,
			BaseAspectRatio:    value.BaseAspectRatio,
			SourceAssStyleName: value.SourceAssStyleName,
			Style:              toAssStyleSpecDTO(value.Style),
		})
	}
	return result
}

func toBilingualStyleDTOs(values []library.BilingualStyle) []dto.LibraryBilingualStyleDTO {
	result := make([]dto.LibraryBilingualStyleDTO, 0, len(values))
	for _, value := range values {
		result = append(result, dto.LibraryBilingualStyleDTO{
			ID:              value.ID,
			Name:            value.Name,
			BuiltIn:         value.BuiltIn,
			BasePlayResX:    value.BasePlayResX,
			BasePlayResY:    value.BasePlayResY,
			BaseAspectRatio: value.BaseAspectRatio,
			Primary:         toMonoStyleSnapshotDTO(value.Primary),
			Secondary:       toMonoStyleSnapshotDTO(value.Secondary),
			Layout:          toBilingualLayoutDTO(value.Layout),
		})
	}
	return result
}

func toMonoStyleSnapshotDTO(value library.MonoStyleSnapshot) dto.LibraryMonoStyleSnapshotDTO {
	return dto.LibraryMonoStyleSnapshotDTO{
		SourceMonoStyleID:   value.SourceMonoStyleID,
		SourceMonoStyleName: value.SourceMonoStyleName,
		Name:                value.Name,
		BasePlayResX:        value.BasePlayResX,
		BasePlayResY:        value.BasePlayResY,
		BaseAspectRatio:     value.BaseAspectRatio,
		Style:               toAssStyleSpecDTO(value.Style),
	}
}

func toBilingualLayoutDTO(value library.BilingualLayout) dto.LibraryBilingualLayoutDTO {
	return dto.LibraryBilingualLayoutDTO{
		Gap:         value.Gap,
		BlockAnchor: value.BlockAnchor,
	}
}

func toAssStyleSpecDTO(value library.AssStyleSpec) dto.AssStyleSpecDTO {
	return dto.AssStyleSpecDTO{
		Fontname:           value.Fontname,
		FontFace:           value.FontFace,
		FontWeight:         value.FontWeight,
		FontPostScriptName: value.FontPostScriptName,
		Fontsize:           value.Fontsize,
		PrimaryColour:      value.PrimaryColour,
		SecondaryColour:    value.SecondaryColour,
		OutlineColour:      value.OutlineColour,
		BackColour:         value.BackColour,
		Bold:               value.Bold,
		Italic:             value.Italic,
		Underline:          value.Underline,
		StrikeOut:          value.StrikeOut,
		ScaleX:             value.ScaleX,
		ScaleY:             value.ScaleY,
		Spacing:            value.Spacing,
		Angle:              value.Angle,
		BorderStyle:        value.BorderStyle,
		Outline:            value.Outline,
		Shadow:             value.Shadow,
		Alignment:          value.Alignment,
		MarginL:            value.MarginL,
		MarginR:            value.MarginR,
		MarginV:            value.MarginV,
		Encoding:           value.Encoding,
	}
}

func toSubtitleStyleDocuments(values []dto.LibrarySubtitleStyleDocumentDTO) []library.SubtitleStyleDocument {
	result := make([]library.SubtitleStyleDocument, 0, len(values))
	for _, value := range values {
		result = append(result, library.SubtitleStyleDocument{
			ID:          strings.TrimSpace(value.ID),
			Name:        strings.TrimSpace(value.Name),
			Description: strings.TrimSpace(value.Description),
			Source:      strings.TrimSpace(value.Source),
			SourceRef:   strings.TrimSpace(value.SourceRef),
			Version:     strings.TrimSpace(value.Version),
			Enabled:     value.Enabled,
			Format:      strings.TrimSpace(value.Format),
			Content:     value.Content,
		})
	}
	return result
}

func toMonoStyles(values []dto.LibraryMonoStyleDTO) []library.MonoStyle {
	result := make([]library.MonoStyle, 0, len(values))
	for _, value := range values {
		result = append(result, library.MonoStyle{
			ID:                 strings.TrimSpace(value.ID),
			Name:               strings.TrimSpace(value.Name),
			BuiltIn:            value.BuiltIn,
			BasePlayResX:       value.BasePlayResX,
			BasePlayResY:       value.BasePlayResY,
			BaseAspectRatio:    strings.TrimSpace(value.BaseAspectRatio),
			SourceAssStyleName: strings.TrimSpace(value.SourceAssStyleName),
			Style:              toAssStyleSpec(value.Style),
		})
	}
	return result
}

func toBilingualStyles(values []dto.LibraryBilingualStyleDTO) []library.BilingualStyle {
	result := make([]library.BilingualStyle, 0, len(values))
	for _, value := range values {
		result = append(result, library.BilingualStyle{
			ID:              strings.TrimSpace(value.ID),
			Name:            strings.TrimSpace(value.Name),
			BuiltIn:         value.BuiltIn,
			BasePlayResX:    value.BasePlayResX,
			BasePlayResY:    value.BasePlayResY,
			BaseAspectRatio: strings.TrimSpace(value.BaseAspectRatio),
			Primary:         toMonoStyleSnapshot(value.Primary),
			Secondary:       toMonoStyleSnapshot(value.Secondary),
			Layout:          toBilingualLayout(value.Layout),
		})
	}
	return result
}

func toMonoStyleSnapshot(value dto.LibraryMonoStyleSnapshotDTO) library.MonoStyleSnapshot {
	return library.MonoStyleSnapshot{
		SourceMonoStyleID:   strings.TrimSpace(value.SourceMonoStyleID),
		SourceMonoStyleName: strings.TrimSpace(value.SourceMonoStyleName),
		Name:                strings.TrimSpace(value.Name),
		BasePlayResX:        value.BasePlayResX,
		BasePlayResY:        value.BasePlayResY,
		BaseAspectRatio:     strings.TrimSpace(value.BaseAspectRatio),
		Style:               toAssStyleSpec(value.Style),
	}
}

func toBilingualLayout(value dto.LibraryBilingualLayoutDTO) library.BilingualLayout {
	return library.BilingualLayout{
		Gap:         value.Gap,
		BlockAnchor: value.BlockAnchor,
	}
}

func toAssStyleSpec(value dto.AssStyleSpecDTO) library.AssStyleSpec {
	return library.AssStyleSpec{
		Fontname:           strings.TrimSpace(value.Fontname),
		FontFace:           strings.TrimSpace(value.FontFace),
		FontWeight:         value.FontWeight,
		FontPostScriptName: strings.TrimSpace(value.FontPostScriptName),
		Fontsize:           value.Fontsize,
		PrimaryColour:      strings.TrimSpace(value.PrimaryColour),
		SecondaryColour:    strings.TrimSpace(value.SecondaryColour),
		OutlineColour:      strings.TrimSpace(value.OutlineColour),
		BackColour:         strings.TrimSpace(value.BackColour),
		Bold:               value.Bold,
		Italic:             value.Italic,
		Underline:          value.Underline,
		StrikeOut:          value.StrikeOut,
		ScaleX:             value.ScaleX,
		ScaleY:             value.ScaleY,
		Spacing:            value.Spacing,
		Angle:              value.Angle,
		BorderStyle:        value.BorderStyle,
		Outline:            value.Outline,
		Shadow:             value.Shadow,
		Alignment:          value.Alignment,
		MarginL:            value.MarginL,
		MarginR:            value.MarginR,
		MarginV:            value.MarginV,
		Encoding:           value.Encoding,
	}
}

func toSubtitleStyleSourceDTOs(values []library.SubtitleStyleSource) []dto.LibrarySubtitleStyleSourceDTO {
	result := make([]dto.LibrarySubtitleStyleSourceDTO, 0, len(values))
	for _, value := range values {
		result = append(result, dto.LibrarySubtitleStyleSourceDTO{
			ID:                 value.ID,
			Name:               value.Name,
			Kind:               value.Kind,
			Provider:           value.Provider,
			URL:                value.URL,
			Prefix:             value.Prefix,
			Filename:           value.Filename,
			Priority:           value.Priority,
			BuiltIn:            value.BuiltIn,
			Owner:              value.Owner,
			Repo:               value.Repo,
			Ref:                value.Ref,
			ManifestPath:       value.ManifestPath,
			RemoteFontManifest: toRemoteFontManifestDTO(value.Manifest),
			Enabled:            value.Enabled,
			FontCount:          value.FontCount,
			SyncStatus:         value.SyncStatus,
			LastSyncedAt:       value.LastSyncedAt,
			LastError:          value.LastError,
		})
	}
	return result
}

func toSubtitleStyleFontDTOs(values []library.SubtitleStyleFont) []dto.LibrarySubtitleStyleFontDTO {
	result := make([]dto.LibrarySubtitleStyleFontDTO, 0, len(values))
	for _, value := range values {
		result = append(result, dto.LibrarySubtitleStyleFontDTO{
			ID:           value.ID,
			Family:       value.Family,
			Source:       value.Source,
			SystemFamily: value.SystemFamily,
			Enabled:      value.Enabled,
		})
	}
	return result
}

func toSubtitleExportPresetDTOs(values []library.SubtitleExportPreset) []dto.LibrarySubtitleExportPresetDTO {
	result := make([]dto.LibrarySubtitleExportPresetDTO, 0, len(values))
	for _, value := range values {
		targetFormat := strings.TrimSpace(value.TargetFormat)
		result = append(result, dto.LibrarySubtitleExportPresetDTO{
			ID:            value.ID,
			Name:          value.Name,
			Format:        targetFormat,
			TargetFormat:  targetFormat,
			MediaStrategy: value.MediaStrategy,
			Config:        toSubtitleExportConfigDTO(value.Config),
		})
	}
	return result
}

func toSubtitleStyleSources(values []dto.LibrarySubtitleStyleSourceDTO) []library.SubtitleStyleSource {
	result := make([]library.SubtitleStyleSource, 0, len(values))
	for _, value := range values {
		result = append(result, library.SubtitleStyleSource{
			ID:           strings.TrimSpace(value.ID),
			Name:         strings.TrimSpace(value.Name),
			Kind:         strings.TrimSpace(value.Kind),
			Provider:     strings.TrimSpace(value.Provider),
			URL:          strings.TrimSpace(value.URL),
			Prefix:       strings.TrimSpace(value.Prefix),
			Filename:     strings.TrimSpace(value.Filename),
			Priority:     value.Priority,
			BuiltIn:      value.BuiltIn,
			Owner:        strings.TrimSpace(value.Owner),
			Repo:         strings.TrimSpace(value.Repo),
			Ref:          strings.TrimSpace(value.Ref),
			ManifestPath: strings.TrimSpace(value.ManifestPath),
			Manifest:     toRemoteFontManifest(value.RemoteFontManifest),
			Enabled:      value.Enabled,
			FontCount:    value.FontCount,
			SyncStatus:   strings.TrimSpace(value.SyncStatus),
			LastSyncedAt: strings.TrimSpace(value.LastSyncedAt),
			LastError:    strings.TrimSpace(value.LastError),
		})
	}
	return result
}

func toSubtitleExportPresets(values []dto.LibrarySubtitleExportPresetDTO) []library.SubtitleExportPreset {
	result := make([]library.SubtitleExportPreset, 0, len(values))
	for _, value := range values {
		targetFormat := strings.TrimSpace(value.TargetFormat)
		if targetFormat == "" {
			targetFormat = strings.TrimSpace(value.Format)
		}
		result = append(result, library.SubtitleExportPreset{
			ID:            strings.TrimSpace(value.ID),
			Name:          strings.TrimSpace(value.Name),
			TargetFormat:  targetFormat,
			MediaStrategy: strings.TrimSpace(value.MediaStrategy),
			Config:        toSubtitleExportConfig(value.Config),
		})
	}
	return result
}

func toSubtitleExportConfigDTO(value library.SubtitleExportConfig) dto.SubtitleExportConfig {
	var result dto.SubtitleExportConfig
	if value.SRT != nil {
		result.SRT = &dto.SubtitleSRTExportConfig{
			Encoding: value.SRT.Encoding,
		}
	}
	if value.VTT != nil {
		result.VTT = &dto.SubtitleVTTExportConfig{
			Kind:     value.VTT.Kind,
			Language: value.VTT.Language,
		}
	}
	if value.ASS != nil {
		result.ASS = &dto.SubtitleASSExportConfig{
			PlayResX: value.ASS.PlayResX,
			PlayResY: value.ASS.PlayResY,
			Title:    value.ASS.Title,
		}
	}
	if value.ITT != nil {
		result.ITT = &dto.SubtitleITTExportConfig{
			FrameRate:           value.ITT.FrameRate,
			FrameRateMultiplier: value.ITT.FrameRateMultiplier,
			Language:            value.ITT.Language,
		}
	}
	if value.FCPXML != nil {
		result.FCPXML = &dto.SubtitleFCPXMLExportConfig{
			FrameDuration:        value.FCPXML.FrameDuration,
			Width:                value.FCPXML.Width,
			Height:               value.FCPXML.Height,
			ColorSpace:           value.FCPXML.ColorSpace,
			Version:              value.FCPXML.Version,
			LibraryName:          value.FCPXML.LibraryName,
			EventName:            value.FCPXML.EventName,
			ProjectName:          value.FCPXML.ProjectName,
			DefaultLane:          value.FCPXML.DefaultLane,
			StartTimecodeSeconds: value.FCPXML.StartTimecodeSeconds,
		}
	}
	return result
}

func toSubtitleExportConfig(value dto.SubtitleExportConfig) library.SubtitleExportConfig {
	var result library.SubtitleExportConfig
	if value.SRT != nil {
		result.SRT = &library.SubtitleSRTExportConfig{
			Encoding: strings.TrimSpace(value.SRT.Encoding),
		}
	}
	if value.VTT != nil {
		result.VTT = &library.SubtitleVTTExportConfig{
			Kind:     strings.TrimSpace(value.VTT.Kind),
			Language: strings.TrimSpace(value.VTT.Language),
		}
	}
	if value.ASS != nil {
		result.ASS = &library.SubtitleASSExportConfig{
			PlayResX: value.ASS.PlayResX,
			PlayResY: value.ASS.PlayResY,
			Title:    strings.TrimSpace(value.ASS.Title),
		}
	}
	if value.ITT != nil {
		result.ITT = &library.SubtitleITTExportConfig{
			FrameRate:           value.ITT.FrameRate,
			FrameRateMultiplier: strings.TrimSpace(value.ITT.FrameRateMultiplier),
			Language:            strings.TrimSpace(value.ITT.Language),
		}
	}
	if value.FCPXML != nil {
		result.FCPXML = &library.SubtitleFCPXMLExportConfig{
			FrameDuration:        strings.TrimSpace(value.FCPXML.FrameDuration),
			Width:                value.FCPXML.Width,
			Height:               value.FCPXML.Height,
			ColorSpace:           strings.TrimSpace(value.FCPXML.ColorSpace),
			Version:              strings.TrimSpace(value.FCPXML.Version),
			LibraryName:          strings.TrimSpace(value.FCPXML.LibraryName),
			EventName:            strings.TrimSpace(value.FCPXML.EventName),
			ProjectName:          strings.TrimSpace(value.FCPXML.ProjectName),
			DefaultLane:          value.FCPXML.DefaultLane,
			StartTimecodeSeconds: value.FCPXML.StartTimecodeSeconds,
		}
	}
	return result
}

func toRemoteFontManifestDTO(value library.SubtitleStyleSourceManifest) dto.LibraryRemoteFontManifestDTO {
	fonts := make(map[string]dto.LibraryRemoteFontManifestFontDTO, len(value.Fonts))
	for id, font := range value.Fonts {
		variants := make([]dto.LibraryRemoteFontManifestVariantDTO, 0, len(font.Variants))
		for _, variant := range font.Variants {
			files := make(map[string]string, len(variant.Files))
			for fileType, rawURL := range variant.Files {
				files[fileType] = rawURL
			}
			variants = append(variants, dto.LibraryRemoteFontManifestVariantDTO{
				Name:    variant.Name,
				Weight:  variant.Weight,
				Style:   variant.Style,
				Subsets: append([]string(nil), variant.Subsets...),
				Files:   files,
			})
		}
		fonts[id] = dto.LibraryRemoteFontManifestFontDTO{
			Name:          font.Name,
			Family:        font.Family,
			License:       font.License,
			LicenseURL:    font.LicenseURL,
			Designer:      font.Designer,
			Foundry:       font.Foundry,
			Version:       font.Version,
			Description:   font.Description,
			Categories:    append([]string(nil), font.Categories...),
			Tags:          append([]string(nil), font.Tags...),
			Popularity:    font.Popularity,
			LastModified:  font.LastModified,
			MetadataURL:   font.MetadataURL,
			SourceURL:     font.SourceURL,
			Variants:      variants,
			UnicodeRanges: append([]string(nil), font.UnicodeRanges...),
			Languages:     append([]string(nil), font.Languages...),
			SampleText:    font.SampleText,
		}
	}
	return dto.LibraryRemoteFontManifestDTO{
		SourceInfo: dto.LibraryRemoteFontManifestInfoDTO{
			Name:        value.SourceInfo.Name,
			Description: value.SourceInfo.Description,
			URL:         value.SourceInfo.URL,
			APIEndpoint: value.SourceInfo.APIEndpoint,
			Version:     value.SourceInfo.Version,
			LastUpdated: value.SourceInfo.LastUpdated,
			TotalFonts:  value.SourceInfo.TotalFonts,
		},
		Fonts: fonts,
	}
}

func toRemoteFontManifest(value dto.LibraryRemoteFontManifestDTO) library.SubtitleStyleSourceManifest {
	fonts := make(map[string]library.SubtitleStyleSourceManifestFont, len(value.Fonts))
	for id, font := range value.Fonts {
		variants := make([]library.SubtitleStyleSourceManifestVariant, 0, len(font.Variants))
		for _, variant := range font.Variants {
			files := make(map[string]string, len(variant.Files))
			for fileType, rawURL := range variant.Files {
				files[fileType] = rawURL
			}
			variants = append(variants, library.SubtitleStyleSourceManifestVariant{
				Name:    variant.Name,
				Weight:  variant.Weight,
				Style:   variant.Style,
				Subsets: append([]string(nil), variant.Subsets...),
				Files:   files,
			})
		}
		fonts[id] = library.SubtitleStyleSourceManifestFont{
			Name:          font.Name,
			Family:        font.Family,
			License:       font.License,
			LicenseURL:    font.LicenseURL,
			Designer:      font.Designer,
			Foundry:       font.Foundry,
			Version:       font.Version,
			Description:   font.Description,
			Categories:    append([]string(nil), font.Categories...),
			Tags:          append([]string(nil), font.Tags...),
			Popularity:    font.Popularity,
			LastModified:  font.LastModified,
			MetadataURL:   font.MetadataURL,
			SourceURL:     font.SourceURL,
			Variants:      variants,
			UnicodeRanges: append([]string(nil), font.UnicodeRanges...),
			Languages:     append([]string(nil), font.Languages...),
			SampleText:    font.SampleText,
		}
	}
	return library.SubtitleStyleSourceManifest{
		SourceInfo: library.SubtitleStyleSourceManifestInfo{
			Name:        value.SourceInfo.Name,
			Description: value.SourceInfo.Description,
			URL:         value.SourceInfo.URL,
			APIEndpoint: value.SourceInfo.APIEndpoint,
			Version:     value.SourceInfo.Version,
			LastUpdated: value.SourceInfo.LastUpdated,
			TotalFonts:  value.SourceInfo.TotalFonts,
		},
		Fonts: fonts,
	}
}

func toSubtitleStyleFonts(values []dto.LibrarySubtitleStyleFontDTO) []library.SubtitleStyleFont {
	result := make([]library.SubtitleStyleFont, 0, len(values))
	for _, value := range values {
		result = append(result, library.SubtitleStyleFont{
			ID:           strings.TrimSpace(value.ID),
			Family:       strings.TrimSpace(value.Family),
			Source:       strings.TrimSpace(value.Source),
			SystemFamily: strings.TrimSpace(value.SystemFamily),
			Enabled:      value.Enabled,
		})
	}
	return result
}

func toLibraryFileDTO(item library.LibraryFile) dto.LibraryFileDTO {
	result := dto.LibraryFileDTO{
		ID:                item.ID,
		LibraryID:         item.LibraryID,
		Kind:              string(item.Kind),
		Name:              item.Name,
		Storage:           dto.LibraryFileStorageDTO{Mode: item.Storage.Mode, LocalPath: item.Storage.LocalPath, DocumentID: item.Storage.DocumentID},
		Lineage:           dto.LibraryFileLineageDTO{RootFileID: item.Lineage.RootFileID},
		LatestOperationID: item.LatestOperationID,
		State: dto.LibraryFileStateDTO{
			Status:      item.State.Status,
			Deleted:     item.State.Deleted,
			Archived:    item.State.Archived,
			LastError:   item.State.LastError,
			LastChecked: item.State.LastChecked,
		},
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
		UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
	}
	if item.Origin.Import != nil {
		result.Origin = dto.LibraryFileOriginDTO{
			Kind: item.Origin.Kind,
			Import: &dto.LibraryImportOriginDTO{
				BatchID:        item.Origin.Import.BatchID,
				ImportPath:     item.Origin.Import.ImportPath,
				ImportedAt:     item.Origin.Import.ImportedAt.Format(time.RFC3339),
				KeepSourceFile: item.Origin.Import.KeepSourceFile,
			},
		}
	} else {
		result.Origin = dto.LibraryFileOriginDTO{Kind: item.Origin.Kind, OperationID: item.Origin.OperationID}
	}
	if item.Media != nil {
		result.Media = &dto.LibraryMediaInfoDTO{
			Format:      item.Media.Format,
			Codec:       item.Media.Codec,
			VideoCodec:  item.Media.VideoCodec,
			AudioCodec:  item.Media.AudioCodec,
			DurationMs:  item.Media.DurationMs,
			Width:       item.Media.Width,
			Height:      item.Media.Height,
			FrameRate:   item.Media.FrameRate,
			BitrateKbps: item.Media.BitrateKbps,
			Channels:    item.Media.Channels,
			SizeBytes:   item.Media.SizeBytes,
		}
	}
	if format := strings.TrimSpace(mediaFormatFromFile(item)); format != "" {
		if result.Media == nil {
			result.Media = &dto.LibraryMediaInfoDTO{}
		}
		if strings.TrimSpace(result.Media.Format) == "" {
			result.Media.Format = format
		}
	}
	result.DisplayLabel = buildLibraryFileDisplayLabel(result)
	return result
}

func toOperationDTO(item library.LibraryOperation) dto.LibraryOperationDTO {
	startedAt := ""
	if item.StartedAt != nil && !item.StartedAt.IsZero() {
		startedAt = item.StartedAt.Format(time.RFC3339)
	}
	finishedAt := ""
	if item.FinishedAt != nil && !item.FinishedAt.IsZero() {
		finishedAt = item.FinishedAt.Format(time.RFC3339)
	}
	return dto.LibraryOperationDTO{
		ID:           item.ID,
		LibraryID:    item.LibraryID,
		Kind:         item.Kind,
		Status:       string(item.Status),
		DisplayName:  item.DisplayName,
		Correlation:  dto.OperationCorrelationDTO{RequestID: item.Correlation.RequestID, RunID: item.Correlation.RunID, ParentOperationID: item.Correlation.ParentOperationID},
		InputJSON:    item.InputJSON,
		OutputJSON:   item.OutputJSON,
		SourceDomain: item.SourceDomain,
		SourceIcon:   item.SourceIcon,
		Meta:         dto.OperationMetaDTO{Platform: item.Meta.Platform, Uploader: item.Meta.Uploader, PublishTime: item.Meta.PublishTime},
		Request:      toOperationRequestPreviewDTO(item),
		Progress:     toProgressDTO(item.Progress, item.Kind, item.Status, item.ErrorMessage),
		OutputFiles:  toOutputFileDTOs(item.OutputFiles),
		Metrics:      dto.OperationMetricsDTO{FileCount: item.Metrics.FileCount, TotalSizeBytes: item.Metrics.TotalSizeBytes, DurationMs: item.Metrics.DurationMs},
		ErrorCode:    item.ErrorCode,
		ErrorMessage: item.ErrorMessage,
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
	}
}

func toOperationListItemDTO(item library.LibraryOperation, libraryName string) dto.OperationListItemDTO {
	startedAt := ""
	if item.StartedAt != nil && !item.StartedAt.IsZero() {
		startedAt = item.StartedAt.Format(time.RFC3339)
	}
	finishedAt := ""
	if item.FinishedAt != nil && !item.FinishedAt.IsZero() {
		finishedAt = item.FinishedAt.Format(time.RFC3339)
	}
	return dto.OperationListItemDTO{
		OperationID: item.ID,
		LibraryID:   item.LibraryID,
		LibraryName: strings.TrimSpace(libraryName),
		Name:        item.DisplayName,
		Kind:        item.Kind,
		Status:      string(item.Status),
		Domain:      item.SourceDomain,
		SourceIcon:  item.SourceIcon,
		Platform:    item.Meta.Platform,
		Uploader:    item.Meta.Uploader,
		PublishTime: item.Meta.PublishTime,
		Progress:    toProgressDTO(item.Progress, item.Kind, item.Status, item.ErrorMessage),
		OutputFiles: toOutputFileDTOs(item.OutputFiles),
		Metrics:     dto.OperationMetricsDTO{FileCount: item.Metrics.FileCount, TotalSizeBytes: item.Metrics.TotalSizeBytes, DurationMs: item.Metrics.DurationMs},
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
	}
}

func toOperationRequestPreviewDTO(item library.LibraryOperation) *dto.OperationRequestPreviewDTO {
	switch item.Kind {
	case "download":
		request := dto.CreateYTDLPJobRequest{}
		if err := json.Unmarshal([]byte(item.InputJSON), &request); err != nil {
			return nil
		}
		preview := dto.OperationRequestPreviewDTO{
			URL:          strings.TrimSpace(request.URL),
			Caller:       strings.TrimSpace(request.Caller),
			Extractor:    firstNonEmpty(strings.TrimSpace(item.Meta.Platform), strings.TrimSpace(request.Extractor)),
			Author:       firstNonEmpty(strings.TrimSpace(item.Meta.Uploader), strings.TrimSpace(request.Author)),
			ThumbnailURL: strings.TrimSpace(request.ThumbnailURL),
		}
		if preview == (dto.OperationRequestPreviewDTO{}) {
			return nil
		}
		return &preview
	default:
		return nil
	}
}

func toHistoryDTO(item library.HistoryRecord) dto.LibraryHistoryRecordDTO {
	result := dto.LibraryHistoryRecordDTO{
		RecordID:    item.ID,
		LibraryID:   item.LibraryID,
		Category:    item.Category,
		Action:      item.Action,
		DisplayName: item.DisplayName,
		Status:      item.Status,
		Source:      dto.LibraryHistoryRecordSourceDTO{Kind: item.Source.Kind, Caller: item.Source.Caller, RunID: item.Source.RunID, Actor: item.Source.Actor},
		Refs:        dto.LibraryHistoryRecordRefsDTO{OperationID: item.Refs.OperationID, ImportBatchID: item.Refs.ImportBatchID, FileIDs: item.Refs.FileIDs, FileEventIDs: item.Refs.FileEventIDs},
		Files:       toOutputFileDTOs(item.Files),
		Metrics:     dto.OperationMetricsDTO{FileCount: item.Metrics.FileCount, TotalSizeBytes: item.Metrics.TotalSizeBytes, DurationMs: item.Metrics.DurationMs},
		OccurredAt:  item.OccurredAt.Format(time.RFC3339),
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
	}
	if item.ImportMeta != nil {
		importedAt := strings.TrimSpace(item.ImportMeta.ImportedAt)
		if importedAt == "" {
			importedAt = item.OccurredAt.Format(time.RFC3339)
		}
		result.ImportMeta = &dto.LibraryImportRecordMetaDTO{ImportPath: item.ImportMeta.ImportPath, KeepSourceFile: item.ImportMeta.KeepSourceFile, ImportedAt: importedAt}
	}
	if item.OperationMeta != nil {
		result.OperationMeta = &dto.LibraryOperationRecordMetaDTO{Kind: item.OperationMeta.Kind, ErrorCode: item.OperationMeta.ErrorCode, ErrorMessage: item.OperationMeta.ErrorMessage}
	}
	return result
}

func toWorkspaceDTO(item library.WorkspaceStateRecord) dto.WorkspaceStateRecordDTO {
	return dto.WorkspaceStateRecordDTO{ID: item.ID, LibraryID: item.LibraryID, StateVersion: item.StateVersion, StateJSON: item.StateJSON, OperationID: item.OperationID, CreatedAt: item.CreatedAt.Format(time.RFC3339)}
}

func toFileEventDTO(item library.FileEventRecord) dto.FileEventRecordDTO {
	detail := dto.FileEventDetailDTO{}
	if strings.TrimSpace(item.DetailJSON) != "" {
		_ = json.Unmarshal([]byte(item.DetailJSON), &detail)
	}
	return dto.FileEventRecordDTO{ID: item.ID, LibraryID: item.LibraryID, FileID: item.FileID, OperationID: item.OperationID, EventType: item.EventType, Detail: detail, CreatedAt: item.CreatedAt.Format(time.RFC3339)}
}

func toProgressDTO(
	progress *library.OperationProgress,
	kind string,
	status library.OperationStatus,
	errorMessage string,
) *dto.OperationProgressDTO {
	if progress == nil {
		return nil
	}
	return &dto.OperationProgressDTO{
		Stage:     progress.Stage,
		Percent:   progress.Percent,
		Current:   progress.Current,
		Total:     progress.Total,
		Speed:     strings.TrimSpace(progress.Speed),
		Message:   normalizeTerminalProgressMessage(kind, status, progress.Message, errorMessage),
		UpdatedAt: progress.UpdatedAt,
	}
}

func toOutputFileDTOs(items []library.OperationOutputFile) []dto.OperationOutputFileDTO {
	result := make([]dto.OperationOutputFileDTO, 0, len(items))
	for _, item := range items {
		result = append(result, dto.OperationOutputFileDTO{
			FileID:    item.FileID,
			Kind:      item.Kind,
			Format:    item.Format,
			SizeBytes: item.SizeBytes,
			IsPrimary: item.IsPrimary,
			Deleted:   item.Deleted,
		})
	}
	return result
}

func buildOperationMetrics(files []library.LibraryFile) library.OperationMetrics {
	metrics := library.OperationMetrics{}
	var total int64
	for _, item := range files {
		if item.State.Deleted {
			continue
		}
		metrics.FileCount++
		if item.Media != nil && item.Media.SizeBytes != nil {
			total += *item.Media.SizeBytes
		}
	}
	if total > 0 {
		metrics.TotalSizeBytes = &total
	}
	return metrics
}

func buildOperationMetricsForOperation(files []library.LibraryFile, startedAt *time.Time, finishedAt *time.Time) library.OperationMetrics {
	metrics := buildOperationMetrics(files)
	metrics.DurationMs = durationMsBetween(startedAt, finishedAt)
	return metrics
}

func durationMsBetween(startedAt *time.Time, finishedAt *time.Time) *int64 {
	if startedAt == nil || finishedAt == nil || startedAt.IsZero() || finishedAt.IsZero() {
		return nil
	}
	duration := finishedAt.Sub(*startedAt).Milliseconds()
	if duration < 0 {
		duration = 0
	}
	return &duration
}

func mediaFormatFromFile(item library.LibraryFile) string {
	if item.Media != nil && strings.TrimSpace(item.Media.Format) != "" {
		return item.Media.Format
	}
	format := normalizeTranscodeFormat(filepath.Ext(strings.TrimSpace(item.Storage.LocalPath)))
	if format != "" {
		return format
	}
	return normalizeTranscodeFormat(filepath.Ext(strings.TrimSpace(item.Name)))
}

func mediaSizeFromFile(item library.LibraryFile) *int64 {
	if item.Media == nil {
		return nil
	}
	return item.Media.SizeBytes
}

func cloneMediaInfo(value *library.MediaInfo) *library.MediaInfo {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func mergeMediaInfo(base *library.MediaInfo, override *library.MediaInfo) *library.MediaInfo {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return cloneMediaInfo(override)
	}
	if override == nil {
		return base
	}
	if strings.TrimSpace(override.Format) != "" {
		base.Format = override.Format
	}
	if strings.TrimSpace(override.Codec) != "" {
		base.Codec = override.Codec
	}
	if strings.TrimSpace(override.VideoCodec) != "" {
		base.VideoCodec = override.VideoCodec
	}
	if strings.TrimSpace(override.AudioCodec) != "" {
		base.AudioCodec = override.AudioCodec
	}
	if override.DurationMs != nil && *override.DurationMs > 0 {
		value := *override.DurationMs
		base.DurationMs = &value
	}
	if override.Width != nil && *override.Width > 0 {
		value := *override.Width
		base.Width = &value
	}
	if override.Height != nil && *override.Height > 0 {
		value := *override.Height
		base.Height = &value
	}
	if override.FrameRate != nil && *override.FrameRate > 0 {
		value := *override.FrameRate
		base.FrameRate = &value
	}
	if override.BitrateKbps != nil && *override.BitrateKbps > 0 {
		value := *override.BitrateKbps
		base.BitrateKbps = &value
	}
	if override.Channels != nil && *override.Channels > 0 {
		value := *override.Channels
		base.Channels = &value
	}
	if override.SizeBytes != nil && *override.SizeBytes > 0 {
		value := *override.SizeBytes
		base.SizeBytes = &value
	}
	if strings.TrimSpace(base.Codec) == "" {
		base.Codec = firstNonEmpty(base.VideoCodec, base.AudioCodec)
	}
	return base
}

func rootFileID(item library.LibraryFile) string {
	if strings.TrimSpace(item.Lineage.RootFileID) != "" {
		return item.Lineage.RootFileID
	}
	return item.ID
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func resolveHistorySourceKind(source string) string {
	if strings.EqualFold(strings.TrimSpace(source), "agent") {
		return "agent"
	}
	if strings.TrimSpace(source) == "" {
		return "manual"
	}
	return strings.ToLower(strings.TrimSpace(source))
}

func deriveLibraryName(name string, fallback string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed != "" {
		return trimmed
	}
	if strings.TrimSpace(fallback) == "" {
		return "Library"
	}
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(fallback)), filepath.Ext(strings.TrimSpace(fallback)))
	if base == "" || base == "." {
		return "Library"
	}
	return base
}

func resolveInitialLibraryName(libraryID string, fallback string, initialNameFromID bool) string {
	if initialNameFromID {
		trimmedID := strings.TrimSpace(libraryID)
		if trimmedID != "" {
			return trimmedID
		}
	}
	return defaultLibraryName(fallback)
}

func resolveLibraryNameFromFile(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	base := filepath.Base(trimmed)
	if base == "" || base == "." {
		return ""
	}
	withoutExt := strings.TrimSuffix(base, filepath.Ext(base))
	if strings.TrimSpace(withoutExt) == "" {
		return base
	}
	return withoutExt
}

func defaultLibraryName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "Library"
	}
	return trimmed
}

func marshalJSON(value interface{}) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func toLookup(items []string) map[string]struct{} {
	result := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.ToLower(strings.TrimSpace(item))
		if trimmed != "" {
			result[trimmed] = struct{}{}
		}
	}
	return result
}

func paginateOperationList(items []dto.OperationListItemDTO, offset int, limit int) []dto.OperationListItemDTO {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []dto.OperationListItemDTO{}
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}

func paginateHistory(items []dto.LibraryHistoryRecordDTO, offset int, limit int) []dto.LibraryHistoryRecordDTO {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []dto.LibraryHistoryRecordDTO{}
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}

func paginateFileEvents(items []dto.FileEventRecordDTO, offset int, limit int) []dto.FileEventRecordDTO {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []dto.FileEventRecordDTO{}
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}

func (service *LibraryService) deriveManagedOutputPath(ctx context.Context, libraryID string, name string, format string, sourcePath string) (string, error) {
	baseDir, err := libraryBaseDir()
	if err != nil {
		return "", err
	}
	resolvedFormat := normalizeTranscodeFormat(format)
	if resolvedFormat == "" {
		resolvedFormat = normalizeTranscodeFormat(filepath.Ext(sourcePath))
	}
	if resolvedFormat == "" {
		resolvedFormat = "mp4"
	}
	safeName := sanitizeFileName(name)
	if safeName == "" {
		safeName = uuid.NewString()
	}
	return filepath.Join(baseDir, "libraries", libraryID, fmt.Sprintf("%s.%s", safeName, resolvedFormat)), nil
}

func sanitizeFileName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "", "\"", "", "<", "", ">", "", "|", "-")
	trimmed = replacer.Replace(trimmed)
	trimmed = strings.Join(strings.Fields(trimmed), "-")
	return strings.Trim(trimmed, "-. _")
}

func resolveSortByOccurred(history []library.HistoryRecord) {
	sort.Slice(history, func(i, j int) bool {
		return history[i].OccurredAt.After(history[j].OccurredAt)
	})
}

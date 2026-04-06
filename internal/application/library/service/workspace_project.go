package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

func (service *LibraryService) GetWorkspaceProject(
	ctx context.Context,
	request dto.GetWorkspaceProjectRequest,
) (dto.WorkspaceProjectDTO, error) {
	item, err := service.libraries.Get(ctx, strings.TrimSpace(request.LibraryID))
	if err != nil {
		return dto.WorkspaceProjectDTO{}, err
	}
	return service.buildWorkspaceProjectDTO(ctx, item)
}

func (service *LibraryService) buildWorkspaceProjectDTO(
	ctx context.Context,
	item library.Library,
) (dto.WorkspaceProjectDTO, error) {
	files, err := service.files.ListByLibraryID(ctx, item.ID)
	if err != nil {
		return dto.WorkspaceProjectDTO{}, err
	}
	operations, err := service.operations.ListByLibraryID(ctx, item.ID)
	if err != nil {
		return dto.WorkspaceProjectDTO{}, err
	}
	reviews := make([]library.SubtitleReviewSession, 0)
	if service.reviews != nil {
		reviews, err = service.reviews.ListByLibraryID(ctx, item.ID)
		if err != nil {
			return dto.WorkspaceProjectDTO{}, err
		}
	}
	moduleConfig, err := service.getModuleConfig(ctx)
	if err != nil {
		return dto.WorkspaceProjectDTO{}, err
	}
	var workspaceHead *dto.WorkspaceStateRecordDTO
	if service.workspace != nil {
		if head, headErr := service.workspace.GetHeadByLibraryID(ctx, item.ID); headErr == nil {
			mapped := toWorkspaceDTO(head)
			workspaceHead = &mapped
		} else if headErr != nil && headErr != library.ErrWorkspaceStateNotFound {
			return dto.WorkspaceProjectDTO{}, headErr
		}
	}
	pendingReviewsByFileID := make(map[string]library.SubtitleReviewSession)
	for _, review := range reviews {
		if review.Status != "pending" {
			continue
		}
		existing, ok := pendingReviewsByFileID[review.FileID]
		if !ok || existing.UpdatedAt.Before(review.UpdatedAt) {
			pendingReviewsByFileID[review.FileID] = review
		}
	}
	taskBuckets := make(map[string]dto.WorkspaceTrackTasksDTO)
	for _, operation := range operations {
		if operation.Status != library.OperationStatusQueued && operation.Status != library.OperationStatusRunning {
			continue
		}
		fileID := resolveWorkspaceOperationSourceFileID(operation)
		if fileID == "" {
			continue
		}
		bucket := taskBuckets[fileID]
		switch operation.Kind {
		case "subtitle_translate":
			bucket.Translate = append(bucket.Translate, toWorkspaceTaskSummaryDTO(operation))
		case "subtitle_proofread":
			task := toWorkspaceTaskSummaryDTO(operation)
			bucket.Proofread = &task
		case "subtitle_qa_review":
			task := toWorkspaceTaskSummaryDTO(operation)
			bucket.QA = &task
		}
		taskBuckets[fileID] = bucket
	}
	videoTracks := make([]dto.WorkspaceVideoTrackDTO, 0)
	subtitleTracks := make([]dto.WorkspaceSubtitleTrackDTO, 0)
	for _, file := range files {
		fileDTO, buildErr := service.buildFileDTOWithConfig(ctx, file, moduleConfig)
		if buildErr != nil {
			fileDTO = toLibraryFileDTO(file)
		}
		switch file.Kind {
		case library.FileKindVideo, library.FileKindAudio, library.FileKindTranscode:
			videoTracks = append(videoTracks, dto.WorkspaceVideoTrackDTO{
				TrackID: file.ID,
				File:    fileDTO,
				Display: buildWorkspaceVideoDisplay(fileDTO),
			})
		case library.FileKindSubtitle:
			pendingReviewDTO := toWorkspacePendingReviewDTO(pendingReviewsByFileID[file.ID])
			subtitleTracks = append(subtitleTracks, dto.WorkspaceSubtitleTrackDTO{
				TrackID:       file.ID,
				Role:          resolveWorkspaceSubtitleRole(file),
				File:          fileDTO,
				Display:       buildWorkspaceSubtitleDisplay(fileDTO),
				RunningTasks:  normalizeWorkspaceTrackTasks(taskBuckets[file.ID]),
				PendingReview: pendingReviewDTO,
			})
		}
	}
	sort.SliceStable(videoTracks, func(i, j int) bool {
		return videoTracks[i].File.CreatedAt > videoTracks[j].File.CreatedAt
	})
	sort.SliceStable(subtitleTracks, func(i, j int) bool {
		return subtitleTracks[i].File.CreatedAt > subtitleTracks[j].File.CreatedAt
	})
	updatedAt := item.UpdatedAt
	for _, review := range reviews {
		if review.UpdatedAt.After(updatedAt) {
			updatedAt = review.UpdatedAt
		}
	}
	workspaceMonoStyle, workspaceLingualStyle := resolveWorkspaceSubtitleStyles(
		moduleConfig,
		workspaceHead,
	)
	return dto.WorkspaceProjectDTO{
		Version:              dto.WorkspaceProjectSchemaVersion,
		LibraryID:            item.ID,
		Title:                item.Name,
		UpdatedAt:            updatedAt.Format(time.RFC3339),
		ViewStateHead:        workspaceHead,
		VideoTracks:          videoTracks,
		SubtitleTracks:       subtitleTracks,
		SubtitleMonoStyle:    workspaceMonoStyle,
		SubtitleLingualStyle: workspaceLingualStyle,
	}, nil
}

type workspaceStyleStateSnapshot struct {
	SubtitleMonoStyle    *dto.LibraryMonoStyleDTO      `json:"subtitleMonoStyle"`
	SubtitleLingualStyle *dto.LibraryBilingualStyleDTO `json:"subtitleLingualStyle"`
}

func resolveWorkspaceSubtitleStyles(
	moduleConfig library.ModuleConfig,
	workspaceHead *dto.WorkspaceStateRecordDTO,
) (*dto.LibraryMonoStyleDTO, *dto.LibraryBilingualStyleDTO) {
	defaultMono := resolveWorkspaceDefaultMonoStyle(moduleConfig)
	defaultLingual := resolveWorkspaceDefaultLingualStyle(moduleConfig)
	if workspaceHead == nil {
		return defaultMono, defaultLingual
	}
	trimmedStateJSON := strings.TrimSpace(workspaceHead.StateJSON)
	if trimmedStateJSON == "" {
		return defaultMono, defaultLingual
	}
	var snapshot workspaceStyleStateSnapshot
	if err := json.Unmarshal([]byte(trimmedStateJSON), &snapshot); err != nil {
		return defaultMono, defaultLingual
	}
	return firstWorkspaceMonoStyle(snapshot.SubtitleMonoStyle, defaultMono),
		firstWorkspaceLingualStyle(snapshot.SubtitleLingualStyle, defaultLingual)
}

func resolveWorkspaceDefaultMonoStyle(moduleConfig library.ModuleConfig) *dto.LibraryMonoStyleDTO {
	items := toMonoStyleDTOs(moduleConfig.SubtitleStyles.MonoStyles)
	if len(items) == 0 {
		return nil
	}
	selectedID := strings.TrimSpace(moduleConfig.SubtitleStyles.Defaults.MonoStyleID)
	style := items[0]
	for _, item := range items {
		if item.ID == selectedID {
			style = item
			break
		}
	}
	return &style
}

func resolveWorkspaceDefaultLingualStyle(moduleConfig library.ModuleConfig) *dto.LibraryBilingualStyleDTO {
	items := toBilingualStyleDTOs(moduleConfig.SubtitleStyles.BilingualStyles)
	if len(items) == 0 {
		return nil
	}
	selectedID := strings.TrimSpace(moduleConfig.SubtitleStyles.Defaults.BilingualStyleID)
	style := items[0]
	for _, item := range items {
		if item.ID == selectedID {
			style = item
			break
		}
	}
	return &style
}

func firstWorkspaceMonoStyle(values ...*dto.LibraryMonoStyleDTO) *dto.LibraryMonoStyleDTO {
	for _, value := range values {
		if value == nil {
			continue
		}
		if strings.TrimSpace(value.ID) == "" && strings.TrimSpace(value.Name) == "" {
			continue
		}
		return value
	}
	return nil
}

func firstWorkspaceLingualStyle(values ...*dto.LibraryBilingualStyleDTO) *dto.LibraryBilingualStyleDTO {
	for _, value := range values {
		if value == nil {
			continue
		}
		if strings.TrimSpace(value.ID) == "" && strings.TrimSpace(value.Name) == "" {
			continue
		}
		return value
	}
	return nil
}

func resolveWorkspaceOperationSourceFileID(operation library.LibraryOperation) string {
	switch strings.TrimSpace(operation.Kind) {
	case "subtitle_translate":
		return extractSubtitleTranslateRequest(operation.InputJSON).FileID
	case "subtitle_proofread":
		return extractSubtitleProofreadRequest(operation.InputJSON).FileID
	case "subtitle_qa_review":
		return extractSubtitleQAReviewRequest(operation.InputJSON).FileID
	default:
		return ""
	}
}

func toWorkspaceTaskSummaryDTO(operation library.LibraryOperation) dto.WorkspaceTaskSummaryDTO {
	result := dto.WorkspaceTaskSummaryDTO{
		OperationID: operation.ID,
		Kind:        operation.Kind,
		Status:      string(operation.Status),
		DisplayName: operation.DisplayName,
	}
	if operation.Progress != nil {
		result.Stage = strings.TrimSpace(operation.Progress.Stage)
		if operation.Progress.Current != nil {
			result.Current = *operation.Progress.Current
		}
		if operation.Progress.Total != nil {
			result.Total = *operation.Progress.Total
		}
		result.UpdatedAt = strings.TrimSpace(operation.Progress.UpdatedAt)
	}
	return result
}

func normalizeWorkspaceTrackTasks(value dto.WorkspaceTrackTasksDTO) dto.WorkspaceTrackTasksDTO {
	if len(value.Translate) > 1 {
		sort.SliceStable(value.Translate, func(i, j int) bool {
			return value.Translate[i].UpdatedAt > value.Translate[j].UpdatedAt
		})
	}
	return value
}

func toWorkspacePendingReviewDTO(session library.SubtitleReviewSession) *dto.WorkspaceTrackPendingReviewDTO {
	if strings.TrimSpace(session.ID) == "" || session.Status != "pending" {
		return nil
	}
	return &dto.WorkspaceTrackPendingReviewDTO{
		SessionID:           session.ID,
		Kind:                session.Kind,
		Status:              session.Status,
		SourceRevisionID:    session.SourceRevisionID,
		CandidateRevisionID: session.CandidateRevisionID,
		ChangedCueCount:     session.ChangedCueCount,
		BlockedActions:      []string{"translate", "proofread", "qa", "export"},
	}
}

func resolveWorkspaceSubtitleRole(file library.LibraryFile) string {
	switch strings.TrimSpace(file.Origin.Kind) {
	case "subtitle_translate":
		return "translation"
	case "subtitle_proofread":
		return "reference"
	default:
		return "source"
	}
}

func buildWorkspaceVideoDisplay(file dto.LibraryFileDTO) dto.WorkspaceTrackDisplayDTO {
	parts := make([]string, 0, 5)
	if resolution := resolveLibraryFileResolution(file.Media); resolution != "" {
		parts = append(parts, resolution)
	}
	if frameRate := resolveLibraryFileFrameRate(file.Media); frameRate != "" {
		parts = append(parts, strings.Replace(frameRate, "fps", " fps", 1))
	}
	if codec := resolveLibraryFileCodec(file); codec != "" {
		parts = append(parts, strings.ToUpper(codec))
	}
	if format := resolveLibraryFileFormat(file); format != "" {
		parts = append(parts, strings.ToUpper(format))
	}
	display := dto.WorkspaceTrackDisplayDTO{
		Label: strings.Join(parts, " · "),
		Hint:  strings.TrimSpace(file.Name),
	}
	if display.Label == "" {
		display.Label = strings.TrimSpace(file.Name)
	}
	if strings.EqualFold(file.Kind, "transcode") {
		display.Badges = append(display.Badges, "转码")
	}
	return display
}

func buildWorkspaceSubtitleDisplay(file dto.LibraryFileDTO) dto.WorkspaceTrackDisplayDTO {
	language := normalizeLibraryFileLanguage(file.Media)
	if language == "" {
		language = "TRACK"
	}
	format := strings.ToUpper(resolveLibraryFileFormat(file))
	role := ""
	switch strings.TrimSpace(file.Origin.Kind) {
	case "subtitle_translate":
		role = "翻译"
	case "subtitle_proofread":
		role = "历史校对"
	}
	parts := []string{language}
	if role != "" {
		parts = append(parts, role)
	} else if format != "" {
		parts = append(parts, format)
	}
	display := dto.WorkspaceTrackDisplayDTO{
		Label: strings.Join(parts, " · "),
		Hint:  strings.TrimSpace(file.Name),
	}
	if strings.TrimSpace(display.Label) == "" {
		display.Label = strings.TrimSpace(file.Name)
	}
	if role != "" {
		display.Badges = append(display.Badges, role)
	}
	return display
}

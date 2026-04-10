package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

type registeredLocalOutputParams struct {
	LibraryID     string
	RootFileID    string
	Name          string
	Kind          string
	OperationID   string
	OperationKind string
	OutputPath    string
	SourceMedia   *library.MediaInfo
	OccurredAt    time.Time
}

type ffmpegProgressReporter struct {
	service     *LibraryService
	operation   *library.LibraryOperation
	durationMs  int64
	mu          sync.Mutex
	currentMs   int64
	speed       string
	lastPercent int
	lastCurrent int64
	lastSpeed   string
	lastPublish time.Time
}

func newFFmpegProgressReporter(service *LibraryService, operation *library.LibraryOperation, durationMs int64) *ffmpegProgressReporter {
	return &ffmpegProgressReporter{
		service:     service,
		operation:   operation,
		durationMs:  durationMs,
		lastPercent: -1,
	}
}

func (reporter *ffmpegProgressReporter) HandleLine(line string) {
	if reporter == nil || reporter.service == nil || reporter.operation == nil || reporter.service.operations == nil {
		return
	}
	key, value, ok := parseFFmpegProgressLine(line)
	if !ok {
		return
	}

	reporter.mu.Lock()
	defer reporter.mu.Unlock()

	if currentMs, ok := parseFFmpegProgressMillis(key, value); ok {
		reporter.currentMs = currentMs
		return
	}
	switch key {
	case "speed":
		reporter.speed = normalizeFFmpegProgressSpeed(value)
	case "progress":
		reporter.persistLocked(strings.EqualFold(strings.TrimSpace(value), "end"))
	}
}

func (reporter *ffmpegProgressReporter) persistLocked(completed bool) {
	currentMs := reporter.currentMs
	speed := reporter.speed
	if currentMs <= 0 && speed == "" && !completed {
		return
	}

	totalMs := reporter.durationMs
	percent := -1
	if totalMs > 0 {
		if completed {
			currentMs = totalMs
			percent = 100
		} else {
			if currentMs > totalMs {
				currentMs = totalMs
			}
			percent = int((currentMs * 100) / totalMs)
			if percent >= 100 {
				percent = 99
			}
		}
	}

	if !completed &&
		percent == reporter.lastPercent &&
		currentMs == reporter.lastCurrent &&
		speed == reporter.lastSpeed &&
		!reporter.lastPublish.IsZero() &&
		time.Since(reporter.lastPublish) < 500*time.Millisecond {
		return
	}

	now := reporter.service.now().Format(time.RFC3339)
	progress := &library.OperationProgress{
		Stage:     progressText("library.progress.transcoding"),
		Message:   progressText("library.progressDetail.ffmpegRenderingOutput"),
		Speed:     speed,
		UpdatedAt: now,
	}
	if currentMs > 0 {
		value := currentMs
		progress.Current = &value
	}
	if totalMs > 0 {
		value := totalMs
		progress.Total = &value
	}
	if percent >= 0 {
		value := percent
		progress.Percent = &value
	}

	reporter.lastPercent = percent
	reporter.lastCurrent = currentMs
	reporter.lastSpeed = speed
	reporter.lastPublish = time.Now()
	reporter.operation.Progress = progress
	if reporter.operation.Status == library.OperationStatusQueued {
		reporter.operation.Status = library.OperationStatusRunning
	}
	if err := reporter.service.operations.Save(context.Background(), *reporter.operation); err != nil {
		return
	}
	reporter.service.publishOperationUpdate(toOperationDTO(*reporter.operation))
}

func (service *LibraryService) runTranscodeOperation(ctx context.Context, operation library.LibraryOperation, request dto.CreateTranscodeJobRequest) {
	ctx, cancel := context.WithCancel(ctx)
	service.registerOperationRun(operation.ID, cancel)
	defer func() {
		service.unregisterOperationRun(operation.ID)
		cancel()
	}()

	now := service.now()
	operation.Status = library.OperationStatusRunning
	operation.StartedAt = &now
	operation.FinishedAt = nil
	operation.ErrorCode = ""
	operation.ErrorMessage = ""
	operation.Progress = buildOperationProgress(
		now,
		progressText("library.progress.preparing"),
		0,
		1,
		progressText("library.progressDetail.preparingFfmpegTranscode"),
	)
	operation.OutputJSON = buildTranscodeOperationOutput(request, "running", "")
	if err := service.saveAndPublishOperation(ctx, operation); err != nil {
		return
	}

	sourceFile, err := service.resolveSourceFileForTranscode(ctx, request)
	if err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}
	probe, err := service.probeRequiredMedia(ctx, sourceFile.Storage.LocalPath)
	if err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}
	plan, err := service.resolveTranscodePlan(ctx, request, probe)
	if err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}
	ffmpegExecPath, err := resolveFFmpegExecPath(ctx, service.tools)
	if err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}

	displayName := resolveTranscodeTitle(request, sourceFile.Storage.LocalPath, plan.preset)
	outputPath, err := service.deriveManagedOutputPath(ctx, sourceFile.LibraryID, displayName, plan.request.Format, sourceFile.Storage.LocalPath)
	if err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}
	if err := ensureManagedOutputParentDir(outputPath); err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}

	tempDir, err := os.MkdirTemp("", "dreamcreator-transcode-*")
	if err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}
	defer os.RemoveAll(tempDir)

	subtitleContent := strings.TrimSpace(request.GeneratedSubtitleContent)
	subtitleHandling := normalizeTranscodeSubtitleHandling(request.SubtitleHandling)
	subtitleFormat := normalizeGeneratedSubtitleFormat(request.GeneratedSubtitleFormat)
	burninSubtitlePath := ""
	embeddedSubtitlePath := ""
	if subtitleHandling != "none" {
		if subtitleContent == "" && !hasGeneratedSubtitleDocumentContent(request.GeneratedSubtitleDocument) {
			service.failTranscodeOperation(ctx, operation, request, fmt.Errorf("subtitle export requires generated subtitles"))
			return
		}
		switch subtitleHandling {
		case "burnin":
			if subtitleFormat != "ass" {
				service.failTranscodeOperation(ctx, operation, request, fmt.Errorf("burn-in subtitles require ASS content"))
				return
			}
		case "embed":
			if subtitleFormat == "" {
				service.failTranscodeOperation(ctx, operation, request, fmt.Errorf("embedded subtitles require a generated subtitle format"))
				return
			}
		}

		tempSubtitlePath := filepath.Join(
			tempDir,
			firstNonEmpty(strings.TrimSpace(request.GeneratedSubtitleName), "workspace-subtitles")+"."+firstNonEmpty(subtitleFormat, "ass"),
		)
		resolvedSubtitleContent := resolveGeneratedSubtitleContentForTranscode(request, probe, subtitleFormat)
		if strings.TrimSpace(resolvedSubtitleContent) == "" {
			service.failTranscodeOperation(ctx, operation, request, fmt.Errorf("subtitle export produced empty subtitle content"))
			return
		}
		if subtitleHandling == "burnin" && subtitleFormat == "ass" {
			resolvedSubtitleContent = normalizeBurninASSFontForChinese(resolvedSubtitleContent)
		}
		if err := os.WriteFile(tempSubtitlePath, []byte(resolvedSubtitleContent), 0o644); err != nil {
			service.failTranscodeOperation(ctx, operation, request, err)
			return
		}
		if subtitleHandling == "burnin" {
			burninSubtitlePath = tempSubtitlePath
		} else if subtitleHandling == "embed" {
			embeddedSubtitlePath = tempSubtitlePath
		}
	}

	acceleratedPlan := plan
	hardwareVideoCodec := resolvePreferredHardwareVideoCodec(ctx, ffmpegExecPath, plan)
	if hardwareVideoCodec != "" {
		acceleratedPlan.request.VideoCodec = hardwareVideoCodec
		// Hardware encoders vary in preset/CRF semantics. Keep CLI conservative and fall back on failure.
		acceleratedPlan.request.Preset = ""
		acceleratedPlan.request.CRF = 0
	}

	ffmpegArgs, err := buildFFmpegTranscodeArgs(
		acceleratedPlan,
		sourceFile.Storage.LocalPath,
		outputPath,
		burninSubtitlePath,
		embeddedSubtitlePath,
		subtitleFormat,
		subtitleHandling,
	)
	if err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}

	progressTime := service.now()
	operation.Progress = buildOperationProgress(
		progressTime,
		progressText("library.progress.transcoding"),
		0,
		1,
		progressText("library.progressDetail.ffmpegRenderingOutput"),
	)
	if err := service.saveAndPublishOperation(ctx, operation); err != nil {
		return
	}

	outputText, err := service.runFFmpegCommandWithProgress(
		ctx,
		&operation,
		ffmpegExecPath,
		ffmpegArgs,
		filepath.Dir(sourceFile.Storage.LocalPath),
		probe.DurationMs,
	)
	if err != nil && hardwareVideoCodec != "" {
		fallbackArgs, fallbackBuildErr := buildFFmpegTranscodeArgs(
			plan,
			sourceFile.Storage.LocalPath,
			outputPath,
			burninSubtitlePath,
			embeddedSubtitlePath,
			subtitleFormat,
			subtitleHandling,
		)
		if fallbackBuildErr == nil {
			outputText, err = service.runFFmpegCommandWithProgress(
				ctx,
				&operation,
				ffmpegExecPath,
				fallbackArgs,
				filepath.Dir(sourceFile.Storage.LocalPath),
				probe.DurationMs,
			)
		}
	}
	if err != nil {
		message := strings.TrimSpace(outputText)
		if message == "" {
			message = err.Error()
		}
		service.failTranscodeOperation(ctx, operation, request, fmt.Errorf("ffmpeg transcode failed: %s", message))
		return
	}
	if !pathExists(outputPath) {
		service.failTranscodeOperation(ctx, operation, request, fmt.Errorf("ffmpeg produced no output file"))
		return
	}

	finishedAt := service.now()
	outputFile, err := service.registerManagedLocalOutputFile(ctx, registeredLocalOutputParams{
		LibraryID:     sourceFile.LibraryID,
		RootFileID:    rootFileID(sourceFile),
		Name:          displayName,
		Kind:          string(library.FileKindTranscode),
		OperationID:   operation.ID,
		OperationKind: "transcode",
		OutputPath:    outputPath,
		SourceMedia:   sourceFile.Media,
		OccurredAt:    finishedAt,
	})
	if err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}
	outputFile.LatestOperationID = operation.ID
	outputFile.UpdatedAt = finishedAt
	if err := service.files.Save(ctx, outputFile); err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}

	files := []library.LibraryFile{outputFile}
	operationOutputs := []library.OperationOutputFile{{
		FileID:    outputFile.ID,
		Kind:      string(outputFile.Kind),
		Format:    mediaFormatFromFile(outputFile),
		SizeBytes: mediaSizeFromFile(outputFile),
		IsPrimary: true,
		Deleted:   outputFile.State.Deleted,
	}}

	sourceFile.LatestOperationID = operation.ID
	sourceFile.UpdatedAt = finishedAt
	if err := service.files.Save(ctx, sourceFile); err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}

	operation.Status = library.OperationStatusSucceeded
	operation.OutputFiles = operationOutputs
	operation.Metrics = buildOperationMetricsForOperation(files, operation.StartedAt, &finishedAt)
	operation.FinishedAt = &finishedAt
	operation.ErrorCode = ""
	operation.ErrorMessage = ""
	operation.Progress = buildOperationProgress(
		finishedAt,
		progressText("library.status.succeeded"),
		1,
		1,
		progressText("library.progressDetail.ffmpegTranscodeCompleted"),
	)
	operation.OutputJSON = buildTranscodeOperationOutput(request, "completed", outputPath)
	if err := service.operations.Save(ctx, operation); err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}

	history, err := library.NewHistoryRecord(library.HistoryRecordParams{
		ID:          uuid.NewString(),
		LibraryID:   sourceFile.LibraryID,
		Category:    "operation",
		Action:      "transcode",
		DisplayName: displayName,
		Status:      string(operation.Status),
		Source:      library.HistoryRecordSource{Kind: resolveHistorySourceKind(request.Source), RunID: strings.TrimSpace(request.RunID)},
		Refs: library.HistoryRecordRefs{
			OperationID: operation.ID,
			FileIDs:     extractLibraryFileIDs(files),
		},
		OccurredAt: &finishedAt,
		CreatedAt:  &finishedAt,
		UpdatedAt:  &finishedAt,
	})
	if err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}
	history.Files = operation.OutputFiles
	history.Metrics = operation.Metrics
	history.OperationMeta = &library.OperationRecordMeta{Kind: "transcode"}
	if err := service.histories.Save(ctx, history); err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}
	if err := service.touchLibrary(ctx, sourceFile.LibraryID, finishedAt); err != nil {
		service.failTranscodeOperation(ctx, operation, request, err)
		return
	}
	if request.DeleteSourceFileAfterTranscode {
		sourceFile = service.cleanupSourceFileAfterSuccessfulTranscode(ctx, sourceFile, operation.ID)
	}

	service.publishOperationUpdate(toOperationDTO(operation))
	service.publishHistoryUpdate(toHistoryDTO(history))
	service.publishFileUpdate(service.mustBuildFileDTO(ctx, sourceFile))
	for _, file := range files {
		service.publishFileUpdate(service.mustBuildFileDTO(ctx, file))
	}
}

func (service *LibraryService) failTranscodeOperation(ctx context.Context, operation library.LibraryOperation, request dto.CreateTranscodeJobRequest, runErr error) {
	if service == nil || service.operations == nil {
		return
	}
	currentOperation := operation
	if item, err := service.operations.Get(ctx, operation.ID); err == nil {
		currentOperation = item
	}
	if errors.Is(runErr, context.Canceled) {
		currentOperation.Status = library.OperationStatusCanceled
		currentOperation.ErrorCode = "transcode_canceled"
		currentOperation.ErrorMessage = ""
	} else {
		currentOperation.Status = library.OperationStatusFailed
		currentOperation.ErrorCode = "transcode_failed"
		currentOperation.ErrorMessage = strings.TrimSpace(runErr.Error())
	}
	now := service.now()
	currentOperation.FinishedAt = &now
	currentOperation.Progress = buildOperationProgress(
		now,
		progressText(progressStageLocaleKey(string(currentOperation.Status))),
		0,
		1,
		terminalProgressMessage(currentOperation.Kind, currentOperation.Status),
	)
	currentOperation.OutputJSON = buildTranscodeOperationOutput(request, strings.TrimSpace(string(currentOperation.Status)), "")
	if err := service.operations.Save(ctx, currentOperation); err != nil {
		return
	}
	service.publishOperationUpdate(toOperationDTO(currentOperation))
}

func (service *LibraryService) cleanupSourceFileAfterSuccessfulTranscode(
	ctx context.Context,
	sourceFile library.LibraryFile,
	transcodeOperationID string,
) library.LibraryFile {
	if service == nil || service.files == nil {
		return sourceFile
	}
	if sourceFile.State.Deleted {
		return sourceFile
	}

	sourceFile.LatestOperationID = strings.TrimSpace(transcodeOperationID)
	if err := service.markLibraryFileDeleted(ctx, sourceFile, true); err != nil {
		return sourceFile
	}

	if updated, err := service.files.Get(ctx, sourceFile.ID); err == nil {
		sourceFile = updated
	}
	service.syncOperationAndHistoryForDeletedOutput(ctx, sourceFile.LibraryID, sourceFile.Origin.OperationID, sourceFile.ID)
	return sourceFile
}

func (service *LibraryService) syncOperationAndHistoryForDeletedOutput(
	ctx context.Context,
	libraryID string,
	operationID string,
	fileID string,
) {
	if service == nil {
		return
	}
	trimmedOperationID := strings.TrimSpace(operationID)
	trimmedFileID := strings.TrimSpace(fileID)
	if trimmedOperationID == "" || trimmedFileID == "" {
		return
	}

	if service.operations != nil {
		operation, err := service.operations.Get(ctx, trimmedOperationID)
		if err == nil {
			updatedOutputFiles, changed := markOperationOutputFileDeleted(operation.OutputFiles, trimmedFileID)
			if changed {
				operation.OutputFiles = updatedOutputFiles
				operation.Metrics = service.rebuildOperationMetricsFromOutputs(ctx, updatedOutputFiles, operation.StartedAt, operation.FinishedAt)
				if saveErr := service.operations.Save(ctx, operation); saveErr == nil {
					service.publishOperationUpdate(toOperationDTO(operation))
				}
			}
		}
	}

	if service.histories != nil {
		histories, err := service.histories.ListByLibraryID(ctx, strings.TrimSpace(libraryID))
		if err != nil {
			return
		}
		for _, history := range histories {
			if history.Refs.OperationID != trimmedOperationID {
				continue
			}
			updatedFiles, changed := markOperationOutputFileDeleted(history.Files, trimmedFileID)
			if !changed {
				return
			}
			history.Files = updatedFiles
			durationMs := history.Metrics.DurationMs
			history.Metrics = service.rebuildOperationMetricsFromOutputs(ctx, updatedFiles, nil, nil)
			history.Metrics.DurationMs = durationMs
			history.UpdatedAt = service.now()
			if saveErr := service.histories.Save(ctx, history); saveErr == nil {
				service.publishHistoryUpdate(toHistoryDTO(history))
			}
			return
		}
	}
}

func markOperationOutputFileDeleted(items []library.OperationOutputFile, fileID string) ([]library.OperationOutputFile, bool) {
	trimmedFileID := strings.TrimSpace(fileID)
	if trimmedFileID == "" || len(items) == 0 {
		return items, false
	}
	updated := append([]library.OperationOutputFile(nil), items...)
	changed := false
	for index := range updated {
		if updated[index].FileID != trimmedFileID || updated[index].Deleted {
			continue
		}
		updated[index].Deleted = true
		changed = true
	}
	return updated, changed
}

func (service *LibraryService) rebuildOperationMetricsFromOutputs(
	ctx context.Context,
	outputFiles []library.OperationOutputFile,
	startedAt *time.Time,
	finishedAt *time.Time,
) library.OperationMetrics {
	files := make([]library.LibraryFile, 0, len(outputFiles))
	if service != nil && service.files != nil {
		for _, output := range outputFiles {
			fileID := strings.TrimSpace(output.FileID)
			if fileID == "" {
				continue
			}
			fileItem, err := service.files.Get(ctx, fileID)
			if err != nil {
				continue
			}
			files = append(files, fileItem)
		}
	}
	return buildOperationMetricsForOperation(files, startedAt, finishedAt)
}

func (service *LibraryService) registerManagedLocalOutputFile(ctx context.Context, params registeredLocalOutputParams) (library.LibraryFile, error) {
	info, err := os.Stat(params.OutputPath)
	if err != nil {
		return library.LibraryFile{}, err
	}
	now := params.OccurredAt
	if now.IsZero() {
		now = service.now()
	}
	probedProbe, err := service.probeRequiredMedia(ctx, params.OutputPath)
	if err != nil {
		return library.LibraryFile{}, err
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
		return library.LibraryFile{}, err
	}
	if err := service.files.Save(ctx, fileItem); err != nil {
		return library.LibraryFile{}, err
	}
	return fileItem, nil
}

func (service *LibraryService) runFFmpegCommandWithProgress(
	ctx context.Context,
	operation *library.LibraryOperation,
	execPath string,
	args []string,
	workDir string,
	durationMs int64,
) (string, error) {
	commandArgs := withFFmpegProgressArgs(args)
	command := exec.CommandContext(ctx, execPath, commandArgs...)
	command.Dir = strings.TrimSpace(workDir)
	configureProcessGroup(command)

	stdout, err := command.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := command.Start(); err != nil {
		return "", err
	}

	reporter := newFFmpegProgressReporter(service, operation, durationMs)
	var stderrBuilder strings.Builder
	var stdoutErr error
	var stderrErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		stdoutErr = scanFFmpegOutput(stdout, nil, reporter.HandleLine)
	}()
	go func() {
		defer wg.Done()
		stderrErr = scanFFmpegOutput(stderr, &stderrBuilder, nil)
	}()

	waitErr := command.Wait()
	wg.Wait()

	if stdoutErr != nil {
		return strings.TrimSpace(stderrBuilder.String()), stdoutErr
	}
	if stderrErr != nil {
		return strings.TrimSpace(stderrBuilder.String()), stderrErr
	}
	return strings.TrimSpace(stderrBuilder.String()), waitErr
}

func ensureManagedOutputParentDir(outputPath string) error {
	parentDir := strings.TrimSpace(filepath.Dir(outputPath))
	if parentDir == "" || parentDir == "." {
		return nil
	}
	return os.MkdirAll(parentDir, 0o755)
}

func withFFmpegProgressArgs(args []string) []string {
	if len(args) == 0 {
		return []string{"-progress", "pipe:1", "-nostats"}
	}
	for index := 0; index < len(args)-1; index++ {
		if args[index] == "-progress" {
			return append([]string{}, args...)
		}
	}
	result := append([]string{}, args[:len(args)-1]...)
	result = append(result, "-progress", "pipe:1", "-nostats", args[len(args)-1])
	return result
}

func scanFFmpegOutput(reader io.Reader, builder *strings.Builder, handler func(string)) error {
	if reader == nil {
		return nil
	}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if builder != nil && strings.TrimSpace(line) != "" {
			if builder.Len() > 0 {
				builder.WriteByte('\n')
			}
			builder.WriteString(line)
		}
		if handler != nil {
			handler(line)
		}
	}
	return scanner.Err()
}

func parseFFmpegProgressLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", "", false
	}
	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", false
	}
	return key, strings.TrimSpace(parts[1]), true
}

func parseFFmpegProgressMillis(key string, value string) (int64, bool) {
	switch strings.TrimSpace(key) {
	case "out_time":
		return parseTimestampToMilliseconds(value)
	case "out_time_ms", "out_time_us":
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || parsed < 0 {
			return 0, false
		}
		return parsed / 1000, true
	default:
		return 0, false
	}
}

func normalizeFFmpegProgressSpeed(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.EqualFold(trimmed, "N/A") {
		return ""
	}
	return trimmed
}

func resolvePreferredHardwareVideoCodec(ctx context.Context, execPath string, plan transcodePlan) string {
	if plan.outputType == library.TranscodeOutputAudio {
		return ""
	}
	requested := normalizeVideoCodecName(plan.request.VideoCodec)
	if requested == "" || requested == "copy" || requested == "vp9" {
		return ""
	}
	encoderSet, err := listFFmpegEncoders(ctx, execPath)
	if err != nil || len(encoderSet) == 0 {
		return ""
	}
	for _, candidate := range preferredHardwareCodecCandidates(requested) {
		if _, ok := encoderSet[candidate]; ok {
			return candidate
		}
	}
	return ""
}

func preferredHardwareCodecCandidates(codec string) []string {
	normalized := normalizeVideoCodecName(codec)
	switch normalized {
	case "h264":
		switch runtime.GOOS {
		case "darwin":
			return []string{"h264_videotoolbox", "h264_qsv", "h264_nvenc"}
		case "windows":
			return []string{"h264_nvenc", "h264_qsv", "h264_amf"}
		default:
			return []string{"h264_nvenc", "h264_vaapi", "h264_qsv", "h264_v4l2m2m"}
		}
	case "h265":
		switch runtime.GOOS {
		case "darwin":
			return []string{"hevc_videotoolbox", "hevc_qsv", "hevc_nvenc"}
		case "windows":
			return []string{"hevc_nvenc", "hevc_qsv", "hevc_amf"}
		default:
			return []string{"hevc_nvenc", "hevc_vaapi", "hevc_qsv", "hevc_v4l2m2m"}
		}
	default:
		return nil
	}
}

func listFFmpegEncoders(ctx context.Context, execPath string) (map[string]struct{}, error) {
	command := exec.CommandContext(ctx, execPath, "-hide_banner", "-encoders")
	configureProcessGroup(command)
	output, err := command.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(output), "\n")
	result := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		flags := fields[0]
		if len(flags) != 6 {
			continue
		}
		encoder := strings.TrimSpace(fields[1])
		if encoder == "" {
			continue
		}
		result[encoder] = struct{}{}
	}
	return result, nil
}

func normalizeBurninASSFontForChinese(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || !containsHanRune(trimmed) {
		return content
	}
	targetFont := preferredChineseSubtitleFont()
	if targetFont == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	inStylesSection := false
	for index, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "[") && strings.HasSuffix(trimmedLine, "]") {
			sectionName := strings.ToLower(trimmedLine)
			inStylesSection = sectionName == "[v4+ styles]" || sectionName == "[v4 styles]"
			continue
		}
		if !inStylesSection || !strings.HasPrefix(strings.ToLower(trimmedLine), "style:") {
			continue
		}

		payload := strings.TrimSpace(trimmedLine[len("style:"):])
		parts := strings.Split(payload, ",")
		if len(parts) < 2 {
			continue
		}
		currentFont := strings.TrimSpace(parts[1])
		if !shouldNormalizeChineseSubtitleFont(currentFont) {
			continue
		}
		parts[1] = targetFont
		prefixIndex := strings.Index(strings.ToLower(line), "style:")
		if prefixIndex < 0 {
			lines[index] = "Style: " + strings.Join(parts, ",")
			continue
		}
		prefix := line[:prefixIndex] + "Style: "
		lines[index] = prefix + strings.Join(parts, ",")
	}
	return strings.Join(lines, "\n")
}

func resolveSourceVideoSubtitlePlayRes(probe mediaProbe) (int, int) {
	if probe.Width <= 0 || probe.Height <= 0 {
		return 0, 0
	}
	return probe.Width, probe.Height
}

func hasGeneratedSubtitleDocumentContent(document *dto.SubtitleDocument) bool {
	return document != nil && len(document.Cues) > 0
}

func resolveGeneratedSubtitleContentForTranscode(
	request dto.CreateTranscodeJobRequest,
	probe mediaProbe,
	subtitleFormat string,
) string {
	subtitleContent := strings.TrimSpace(request.GeneratedSubtitleContent)
	if !hasGeneratedSubtitleDocumentContent(request.GeneratedSubtitleDocument) {
		if subtitleFormat == "ass" {
			playResX, playResY := resolveSourceVideoSubtitlePlayRes(probe)
			return normalizeGeneratedASSPlayResForTranscode(subtitleContent, playResX, playResY)
		}
		return subtitleContent
	}

	renderFormat := firstNonEmpty(
		strings.TrimSpace(subtitleFormat),
		strings.TrimSpace(request.GeneratedSubtitleDocument.Format),
		"ass",
	)
	var exportConfig *dto.SubtitleExportConfig
	if renderFormat == "ass" {
		playResX, playResY := resolveSourceVideoSubtitlePlayRes(probe)
		exportConfig = &dto.SubtitleExportConfig{
			ASS: &dto.SubtitleASSExportConfig{
				PlayResX: playResX,
				PlayResY: playResY,
				Title: firstNonEmpty(
					strings.TrimSpace(request.GeneratedSubtitleName),
					strings.TrimSpace(request.Title),
					"DreamCreator Export",
				),
			},
		}
	}
	rendered := renderSubtitleContentWithConfig(
		*request.GeneratedSubtitleDocument,
		renderFormat,
		exportConfig,
		request.GeneratedSubtitleStyleDocumentContent,
	)
	if strings.TrimSpace(rendered) != "" {
		return rendered
	}
	if renderFormat == "ass" {
		playResX, playResY := resolveSourceVideoSubtitlePlayRes(probe)
		return normalizeGeneratedASSPlayResForTranscode(subtitleContent, playResX, playResY)
	}
	return subtitleContent
}

func normalizeGeneratedASSPlayResForTranscode(content string, playResX int, playResY int) string {
	normalized := normalizeSubtitleExportStyleDocumentContent(content)
	if normalized == "" || playResX <= 0 || playResY <= 0 {
		return content
	}
	lines := strings.Split(normalized, "\n")
	styleDocument, hasScriptInfo, _ := parseSubtitleExportStyleDocumentContent(
		normalized,
		subtitleExportStyleDocumentOptions{
			PlayResX: playResX,
			PlayResY: playResY,
		},
	)
	if !hasScriptInfo {
		return content
	}

	eventsStartIndex := -1
	for index, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if strings.EqualFold(trimmed, "[events]") {
			eventsStartIndex = index
			break
		}
	}
	if eventsStartIndex < 0 {
		return strings.Join(styleDocument.Lines, "\n")
	}

	result := append([]string{}, styleDocument.Lines...)
	if len(result) > 0 && strings.TrimSpace(result[len(result)-1]) != "" {
		result = append(result, "")
	}
	result = append(result, lines[eventsStartIndex:]...)
	return strings.Join(result, "\n")
}

func containsHanRune(content string) bool {
	for _, r := range content {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func preferredChineseSubtitleFont() string {
	switch runtime.GOOS {
	case "windows":
		return "Microsoft YaHei"
	case "darwin":
		return "PingFang SC"
	default:
		return "Noto Sans CJK SC"
	}
}

func shouldNormalizeChineseSubtitleFont(fontName string) bool {
	normalized := strings.ToLower(strings.TrimSpace(fontName))
	switch normalized {
	case "", "arial", "helvetica", "sans", "sans-serif":
		return true
	default:
		return false
	}
}

func buildFFmpegTranscodeArgs(
	plan transcodePlan,
	inputPath string,
	outputPath string,
	burninSubtitlePath string,
	embeddedSubtitlePath string,
	embeddedSubtitleFormat string,
	subtitleHandling string,
) ([]string, error) {
	args := []string{"-y", "-i", inputPath}
	if subtitleHandling == "embed" && strings.TrimSpace(embeddedSubtitlePath) != "" {
		args = append(args, "-i", embeddedSubtitlePath)
	}
	filters := make([]string, 0, 2)
	if subtitleHandling == "burnin" && strings.TrimSpace(burninSubtitlePath) != "" {
		filters = append(filters, "ass="+quoteFFmpegFilterPath(burninSubtitlePath))
	}
	scaleFilter, err := buildFFmpegScaleFilter(plan.request)
	if err != nil {
		return nil, err
	}
	if scaleFilter != "" {
		filters = append(filters, scaleFilter)
	}
	if len(filters) > 0 {
		args = append(args, "-vf", strings.Join(filters, ","))
	}

	outputType := plan.outputType
	container := normalizeContainer(plan.request.Format)
	if outputType == library.TranscodeOutputAudio || isAudioContainer(container) {
		args = append(args, "-vn")
	} else {
		if subtitleHandling == "embed" && strings.TrimSpace(embeddedSubtitlePath) != "" {
			args = append(args, "-map", "0:v:0?", "-map", "0:a:0?", "-map", "1:0")
		}
		videoCodec := normalizeVideoCodecName(plan.request.VideoCodec)
		if videoCodec == "" {
			videoCodec = "h264"
		}
		if len(filters) > 0 && videoCodec == "copy" {
			videoCodec = "h264"
		}
		selectedVideoCodec := ffmpegVideoCodec(videoCodec)
		args = append(args, "-c:v", selectedVideoCodec)
		isHardwareVideoCodec := ffmpegIsHardwareVideoCodec(selectedVideoCodec)
		if videoCodec != "copy" {
			if preset := strings.TrimSpace(plan.request.Preset); preset != "" && !isHardwareVideoCodec {
				args = append(args, "-preset", preset)
			}
			qualityMode := strings.ToLower(strings.TrimSpace(plan.request.QualityMode))
			switch qualityMode {
			case "bitrate":
				if plan.request.BitrateKbps > 0 {
					args = append(args, "-b:v", fmt.Sprintf("%dk", plan.request.BitrateKbps))
				}
			default:
				if plan.request.CRF > 0 && !isHardwareVideoCodec {
					args = append(args, "-crf", strconv.Itoa(plan.request.CRF))
				}
			}
		}
	}

	audioCodec := normalizeAudioCodecName(plan.request.AudioCodec)
	if audioCodec == "" {
		if outputType == library.TranscodeOutputAudio {
			audioCodec = firstNonEmpty(defaultAudioCodecForContainer(container), "mp3")
		} else {
			audioCodec = firstNonEmpty(defaultAudioCodecForContainer(container), "aac")
		}
	}
	if audioCodec != "" {
		args = append(args, "-c:a", ffmpegAudioCodec(audioCodec))
		if audioCodec != "copy" && plan.request.AudioBitrateKbps > 0 {
			args = append(args, "-b:a", fmt.Sprintf("%dk", plan.request.AudioBitrateKbps))
		}
	}
	if subtitleHandling == "embed" && strings.TrimSpace(embeddedSubtitlePath) != "" {
		subtitleCodec, err := resolveFFmpegSubtitleCodec(container, embeddedSubtitleFormat)
		if err != nil {
			return nil, err
		}
		args = append(args, "-c:s", subtitleCodec)
	}

	args = append(args, outputPath)
	return args, nil
}

func buildFFmpegScaleFilter(request dto.CreateTranscodeJobRequest) (string, error) {
	scale := strings.ToLower(strings.TrimSpace(request.Scale))
	if scale == "" || scale == "original" {
		return "", nil
	}
	width := request.Width
	height := request.Height
	if scale != "custom" {
		target, ok := scaleTargets[scale]
		if !ok {
			return "", fmt.Errorf("unsupported scale preset")
		}
		width = target[0]
		height = target[1]
	}
	if width <= 0 || height <= 0 {
		return "", fmt.Errorf("invalid scale dimensions")
	}
	return fmt.Sprintf(
		"scale=w=%d:h=%d:force_original_aspect_ratio=decrease:force_divisible_by=2,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
		width,
		height,
		width,
		height,
	), nil
}

func quoteFFmpegFilterPath(path string) string {
	escaped := filepath.ToSlash(strings.TrimSpace(path))
	replacer := strings.NewReplacer("\\", "\\\\", ":", "\\:", "'", "\\'", ",", "\\,", "[", "\\[", "]", "\\]")
	return fmt.Sprintf("filename='%s'", replacer.Replace(escaped))
}

func ffmpegVideoCodec(codec string) string {
	switch normalizeVideoCodecName(codec) {
	case "h264":
		return "libx264"
	case "h265":
		return "libx265"
	case "vp9":
		return "libvpx-vp9"
	default:
		return firstNonEmpty(codec, "libx264")
	}
}

func ffmpegIsHardwareVideoCodec(codec string) bool {
	normalized := strings.ToLower(strings.TrimSpace(codec))
	return strings.HasSuffix(normalized, "_nvenc") ||
		strings.HasSuffix(normalized, "_videotoolbox") ||
		strings.HasSuffix(normalized, "_qsv") ||
		strings.HasSuffix(normalized, "_vaapi") ||
		strings.HasSuffix(normalized, "_amf") ||
		strings.HasSuffix(normalized, "_v4l2m2m")
}

func ffmpegAudioCodec(codec string) string {
	switch normalizeAudioCodecName(codec) {
	case "aac":
		return "aac"
	case "mp3":
		return "libmp3lame"
	case "opus":
		return "libopus"
	case "flac":
		return "flac"
	case "pcm":
		return "pcm_s16le"
	default:
		return firstNonEmpty(codec, "aac")
	}
}

func resolveFFmpegSubtitleCodec(container string, subtitleFormat string) (string, error) {
	switch normalizeContainer(container) {
	case "mp4", "mov":
		return "mov_text", nil
	case "webm":
		return "webvtt", nil
	case "", "mkv":
		switch normalizeGeneratedSubtitleFormat(subtitleFormat) {
		case "vtt":
			return "webvtt", nil
		case "ass":
			return "ass", nil
		case "srt":
			return "subrip", nil
		default:
			return "", fmt.Errorf("unsupported embedded subtitle format")
		}
	default:
		switch normalizeGeneratedSubtitleFormat(subtitleFormat) {
		case "vtt":
			return "webvtt", nil
		case "ass":
			return "ass", nil
		case "srt":
			return "subrip", nil
		default:
			return "", fmt.Errorf("unsupported embedded subtitle format")
		}
	}
}

func normalizeGeneratedSubtitleFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ssa":
		return "ass"
	case "ass", "srt", "vtt":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func normalizeTranscodeSubtitleHandling(value string) string {
	switch strings.TrimSpace(value) {
	case "burnin":
		return "burnin"
	case "embed":
		return "embed"
	default:
		return "none"
	}
}

func buildTranscodeOperationOutput(request dto.CreateTranscodeJobRequest, status string, outputPath string) string {
	payload := map[string]any{
		"status":                         strings.TrimSpace(status),
		"subtitleHandling":               normalizeTranscodeSubtitleHandling(request.SubtitleHandling),
		"displayMode":                    strings.TrimSpace(request.DisplayMode),
		"subtitleDocumentId":             strings.TrimSpace(request.SubtitleDocumentID),
		"subtitleFileId":                 strings.TrimSpace(request.SubtitleFileID),
		"secondaryFileId":                strings.TrimSpace(request.SecondarySubtitleFileID),
		"generatedFormat":                normalizeGeneratedSubtitleFormat(request.GeneratedSubtitleFormat),
		"generatedName":                  strings.TrimSpace(request.GeneratedSubtitleName),
		"embeddedSubtitleFormat":         firstNonEmpty(normalizeGeneratedSubtitleFormat(request.GeneratedSubtitleFormat), ""),
		"deleteSourceFileAfterTranscode": request.DeleteSourceFileAfterTranscode,
		"outputPath":                     strings.TrimSpace(outputPath),
	}
	return marshalJSON(payload)
}

func extractLibraryFileIDs(files []library.LibraryFile) []string {
	result := make([]string, 0, len(files))
	for _, file := range files {
		if trimmed := strings.TrimSpace(file.ID); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func extractTranscodeRequest(inputJSON string) dto.CreateTranscodeJobRequest {
	request := dto.CreateTranscodeJobRequest{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(inputJSON)), &request)
	return request
}

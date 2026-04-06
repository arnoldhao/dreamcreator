package service

import (
	"context"
	"math"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	appytdlp "dreamcreator/internal/application/ytdlp"
	"dreamcreator/internal/domain/library"
)

const (
	ytdlpProgressPrefix = appytdlp.ProgressPrefix
	progressI18nPrefix  = "i18n:"
)

type ytdlpProgressReporter struct {
	service     *LibraryService
	operation   *library.LibraryOperation
	mu          sync.Mutex
	stageCode   string
	lastPercent float64
	lastPublish time.Time
}

func newYTDLPProgressReporter(service *LibraryService, operation *library.LibraryOperation) *ytdlpProgressReporter {
	return &ytdlpProgressReporter{
		service:     service,
		operation:   operation,
		lastPercent: -1,
	}
}

func (reporter *ytdlpProgressReporter) HandleLine(line string) {
	if reporter == nil || reporter.service == nil || reporter.operation == nil {
		return
	}

	stage, current, total, ok := appytdlp.DetectStage(line)
	if ok {
		reporter.updateStage(stage, current, total, line)
	}

	if jsonProgress, ok := appytdlp.ParseProgressJSON(line, ytdlpProgressPrefix); ok {
		reporter.emitJSONProgress(jsonProgress)
		return
	}

	percent, totalBytes, speed, ok := appytdlp.ParseDownloadProgress(line)
	if !ok {
		return
	}
	if reporter.stageCode == "" {
		reporter.stageCode = "downloading"
	}
	if !reporter.shouldPublish(percent) {
		return
	}

	currentBytes := int64(0)
	var currentPtr *int64
	if totalBytes != nil {
		currentBytes = int64(math.Round(float64(*totalBytes) * percent / 100))
		currentPtr = &currentBytes
	}
	reporter.persistProgress(currentPtr, totalBytes, floatPtr(percent), buildProgressMessage("", speed), speed)
}

func (reporter *ytdlpProgressReporter) updateStage(stage string, current int, total int, line string) {
	if reporter == nil || stage == "" {
		return
	}
	stageCode := ytdlpStageCode(stage)
	if stageCode == "" {
		return
	}
	if stageCode == reporter.stageCode {
		return
	}

	reporter.stageCode = stageCode
	var currentPtr *int64
	var totalPtr *int64
	if current > 0 {
		value := int64(current)
		currentPtr = &value
	}
	if total > 0 {
		value := int64(total)
		totalPtr = &value
	}
	reporter.persistProgress(currentPtr, totalPtr, nil, normalizeProgressDetail(stage, line), "")
}

func (reporter *ytdlpProgressReporter) emitJSONProgress(progress appytdlp.JSONProgress) {
	if reporter == nil {
		return
	}

	stage := appytdlp.StageFromProgressStatus(progress.Status)
	if stage != "" {
		nextStageCode := ytdlpStageCode(stage)
		if nextStageCode == "downloading" {
			if reporter.stageCode == "" || !strings.HasPrefix(reporter.stageCode, "downloading") {
				reporter.stageCode = nextStageCode
			}
		} else if nextStageCode != "" {
			reporter.stageCode = nextStageCode
		}
	}
	if reporter.stageCode == "" {
		reporter.stageCode = "downloading"
	}

	percent, totalBytes := appytdlp.ComputeJSONProgressPercent(progress)
	if percent != nil {
		if !reporter.shouldPublish(*percent) {
			return
		}
		reporter.persistProgress(progress.DownloadedBytes, totalBytes, percent, buildProgressMessage("", progress.Speed), progress.Speed)
		return
	}

	if !reporter.lastPublish.IsZero() && time.Since(reporter.lastPublish) < 500*time.Millisecond {
		return
	}
	reporter.persistProgress(progress.DownloadedBytes, totalBytes, nil, buildProgressMessage("", progress.Speed), progress.Speed)
}

func (reporter *ytdlpProgressReporter) Finalize() {
	if reporter == nil || reporter.service == nil || reporter.operation == nil {
		return
	}
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	now := reporter.service.now().Format(time.RFC3339)
	percent := 100
	reporter.operation.Progress = &library.OperationProgress{
		Stage:     progressText("library.status.succeeded"),
		Percent:   &percent,
		Message:   progressText("library.status.succeeded"),
		UpdatedAt: now,
	}
}

func (reporter *ytdlpProgressReporter) shouldPublish(percent float64) bool {
	if percent < 0 {
		return false
	}
	if reporter.lastPublish.IsZero() {
		return true
	}
	if time.Since(reporter.lastPublish) < 300*time.Millisecond && math.Abs(percent-reporter.lastPercent) < 0.2 {
		return false
	}
	return true
}

func (reporter *ytdlpProgressReporter) persistProgress(current *int64, total *int64, percent *float64, message string, speed string) {
	if reporter == nil || reporter.service == nil || reporter.operation == nil {
		return
	}
	reporter.mu.Lock()
	defer reporter.mu.Unlock()

	now := reporter.service.now().Format(time.RFC3339)
	progress := &library.OperationProgress{
		Stage:     progressText(progressStageLocaleKey(reporter.stageCode)),
		Current:   current,
		Total:     total,
		Speed:     strings.TrimSpace(speed),
		Message:   strings.TrimSpace(message),
		UpdatedAt: now,
	}
	if percent != nil {
		value := int(math.Round(*percent))
		if value < 0 {
			value = 0
		}
		if value > 100 {
			value = 100
		}
		progress.Percent = &value
		reporter.lastPercent = *percent
	}

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

func (reporter *ytdlpProgressReporter) WithOperationLock(fn func()) {
	if reporter == nil || fn == nil {
		return
	}
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	fn()
}

func terminalProgressMessage(kind string, status library.OperationStatus) string {
	switch status {
	case library.OperationStatusCanceled:
		switch strings.TrimSpace(kind) {
		case "download":
			return progressText("library.progressDetail.downloadCanceled")
		case "transcode":
			return progressText("library.progressDetail.transcodeCanceled")
		case "subtitle_translate":
			return progressText("library.progressDetail.subtitleTranslationCanceled")
		case "subtitle_proofread":
			return progressText("library.progressDetail.subtitleProofreadCanceled")
		case "subtitle_qa_review":
			return progressText("library.progressDetail.subtitleQaReviewCanceled")
		default:
			return progressText("library.progressDetail.operationCanceled")
		}
	case library.OperationStatusFailed:
		switch strings.TrimSpace(kind) {
		case "download":
			return progressText("library.progressDetail.downloadFailed")
		case "transcode":
			return progressText("library.progressDetail.transcodeFailed")
		case "subtitle_translate":
			return progressText("library.progressDetail.subtitleTranslationFailed")
		case "subtitle_proofread":
			return progressText("library.progressDetail.subtitleProofreadFailed")
		case "subtitle_qa_review":
			return progressText("library.progressDetail.subtitleQaReviewFailed")
		default:
			return progressText("library.progressDetail.operationFailed")
		}
	default:
		return ""
	}
}

func normalizeTerminalProgressMessage(
	kind string,
	status library.OperationStatus,
	currentMessage string,
	errorMessage string,
) string {
	message := strings.TrimSpace(currentMessage)
	if status != library.OperationStatusFailed && status != library.OperationStatusCanceled {
		return message
	}

	fallback := terminalProgressMessage(kind, status)
	if fallback == "" {
		return message
	}
	if message == "" {
		return fallback
	}
	if trimmedError := strings.TrimSpace(errorMessage); trimmedError != "" && message == trimmedError {
		return fallback
	}
	if strings.EqualFold(message, string(status)) {
		return fallback
	}
	return message
}

func normalizeProgressDetail(stage string, line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return stage
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "[download]") {
		return stage
	}
	return trimmed
}

func buildProgressMessage(detail string, speed string) string {
	parts := make([]string, 0, 2)
	if trimmed := strings.TrimSpace(detail); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if trimmed := strings.TrimSpace(speed); trimmed != "" {
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, " · ")
}

func floatPtr(value float64) *float64 {
	copyValue := value
	return &copyValue
}

func progressText(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	return progressI18nPrefix + key
}

func progressTextTemplate(key string, params map[string]string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(params) == 0 {
		return progressText(key)
	}
	values := url.Values{}
	keys := make([]string, 0, len(params))
	for field := range params {
		keys = append(keys, field)
	}
	sort.Strings(keys)
	for _, field := range keys {
		value := strings.TrimSpace(params[field])
		if value == "" {
			continue
		}
		values.Set(field, value)
	}
	if encoded := values.Encode(); encoded != "" {
		return progressText(key) + "?" + encoded
	}
	return progressText(key)
}

func progressStageLocaleKey(code string) string {
	switch strings.TrimSpace(code) {
	case "starting":
		return "library.progress.starting"
	case "preparing":
		return "library.progress.preparing"
	case "fetching_metadata":
		return "library.progress.fetchingMetadata"
	case "translating":
		return "library.progress.translating"
	case "proofreading":
		return "library.progress.proofreading"
	case "qa_reviewing":
		return "library.progress.qaReviewing"
	case "transcoding":
		return "library.progress.transcoding"
	case "downloading":
		return "library.progress.downloading"
	case "downloading_video":
		return "library.progress.downloadingVideo"
	case "downloading_audio":
		return "library.progress.downloadingAudio"
	case "downloading_subtitles":
		return "library.progress.downloadingSubtitles"
	case "downloading_thumbnail":
		return "library.progress.downloadingThumbnail"
	case "muxing":
		return "library.progress.muxing"
	case "cleaning_up":
		return "library.progress.cleaningUp"
	case "post_processing":
		return "library.progress.postProcessing"
	case "queued":
		return "library.status.queued"
	case "running":
		return "library.status.running"
	case "completed":
		return "library.status.succeeded"
	case "failed":
		return "library.status.failed"
	case "canceled":
		return "library.status.canceled"
	default:
		return ""
	}
}

func ytdlpStageCode(stage string) string {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "starting":
		return "starting"
	case "downloading":
		return "downloading"
	case "fetching metadata":
		return "fetching_metadata"
	case "downloading video":
		return "downloading_video"
	case "downloading audio":
		return "downloading_audio"
	case "downloading subtitles":
		return "downloading_subtitles"
	case "downloading thumbnail":
		return "downloading_thumbnail"
	case "muxing":
		return "muxing"
	case "cleaning up":
		return "cleaning_up"
	case "post-processing", "post processing", "postprocessing":
		return "post_processing"
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	default:
		return ""
	}
}

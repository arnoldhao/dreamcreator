package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

type transcodeTestOperationRepo struct {
	saved []library.LibraryOperation
}

func (repo *transcodeTestOperationRepo) List(_ context.Context) ([]library.LibraryOperation, error) {
	return nil, nil
}

func (repo *transcodeTestOperationRepo) ListByLibraryID(_ context.Context, _ string) ([]library.LibraryOperation, error) {
	return nil, nil
}

func (repo *transcodeTestOperationRepo) Get(_ context.Context, id string) (library.LibraryOperation, error) {
	for index := len(repo.saved) - 1; index >= 0; index-- {
		if repo.saved[index].ID == id {
			return repo.saved[index], nil
		}
	}
	return library.LibraryOperation{}, library.ErrOperationNotFound
}

func (repo *transcodeTestOperationRepo) Save(_ context.Context, item library.LibraryOperation) error {
	repo.saved = append(repo.saved, item)
	return nil
}

func (repo *transcodeTestOperationRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func TestBuildFFmpegTranscodeArgsBurnInEscapesASSPathAndForcesCodec(t *testing.T) {
	plan := transcodePlan{
		request: dto.CreateTranscodeJobRequest{
			Format:     "mp4",
			VideoCodec: "copy",
			AudioCodec: "copy",
			Preset:     "medium",
			CRF:        23,
		},
		outputType: library.TranscodeOutputVideo,
	}

	args, err := buildFFmpegTranscodeArgs(
		plan,
		"/tmp/input.mp4",
		"/tmp/output.mp4",
		"/tmp/subtitles:demo,track[1]'s.ass",
		"",
		"",
		"burnin",
	)
	if err != nil {
		t.Fatalf("buildFFmpegTranscodeArgs returned error: %v", err)
	}

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-vf ass=filename='/tmp/subtitles\\:demo\\,track\\[1\\]\\'s.ass'") {
		t.Fatalf("expected escaped ass filter in args, got %q", joined)
	}
	if !strings.Contains(joined, "-c:v libx264") {
		t.Fatalf("expected filtered copy job to fall back to libx264, got %q", joined)
	}
	if !strings.Contains(joined, "-c:a copy") {
		t.Fatalf("expected audio copy to remain unchanged, got %q", joined)
	}
}

func TestBuildFFmpegTranscodeArgsEmbedMapsSubtitleTrackAndUsesContainerCodec(t *testing.T) {
	plan := transcodePlan{
		request: dto.CreateTranscodeJobRequest{
			Format:     "mp4",
			VideoCodec: "h264",
			AudioCodec: "aac",
			Preset:     "medium",
			CRF:        23,
		},
		outputType: library.TranscodeOutputVideo,
	}

	args, err := buildFFmpegTranscodeArgs(
		plan,
		"/tmp/input.mp4",
		"/tmp/output.mp4",
		"",
		"/tmp/subtitles.srt",
		"srt",
		"embed",
	)
	if err != nil {
		t.Fatalf("buildFFmpegTranscodeArgs returned error: %v", err)
	}

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-i /tmp/subtitles.srt") {
		t.Fatalf("expected embedded subtitle input, got %q", joined)
	}
	if !strings.Contains(joined, "-map 0:v:0? -map 0:a:0? -map 1:0") {
		t.Fatalf("expected explicit stream mapping for embedded subtitles, got %q", joined)
	}
	if !strings.Contains(joined, "-c:s mov_text") {
		t.Fatalf("expected mov_text subtitle codec for mp4, got %q", joined)
	}
}

func TestBuildFFmpegTranscodeArgsAudioOutputDisablesVideo(t *testing.T) {
	plan := transcodePlan{
		request: dto.CreateTranscodeJobRequest{
			Format: "mp3",
		},
		outputType: library.TranscodeOutputAudio,
	}

	args, err := buildFFmpegTranscodeArgs(plan, "/tmp/input.mp4", "/tmp/output.mp3", "", "", "", "none")
	if err != nil {
		t.Fatalf("buildFFmpegTranscodeArgs returned error: %v", err)
	}

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-vn") {
		t.Fatalf("expected audio-only transcode to disable video, got %q", joined)
	}
	if !strings.Contains(joined, "-c:a libmp3lame") {
		t.Fatalf("expected default mp3 audio codec, got %q", joined)
	}
}

func TestBuildFFmpegTranscodeArgsExpandedAudioContainersUseExpectedCodec(t *testing.T) {
	testCases := []struct {
		name          string
		format        string
		audioCodec    string
		expectedCodec string
	}{
		{name: "m4a aac", format: "m4a", audioCodec: "aac", expectedCodec: "-c:a aac"},
		{name: "ogg opus", format: "ogg", audioCodec: "opus", expectedCodec: "-c:a libopus"},
		{name: "flac lossless", format: "flac", audioCodec: "flac", expectedCodec: "-c:a flac"},
		{name: "wav pcm", format: "wav", audioCodec: "pcm", expectedCodec: "-c:a pcm_s16le"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			plan := transcodePlan{
				request: dto.CreateTranscodeJobRequest{
					Format:     testCase.format,
					AudioCodec: testCase.audioCodec,
				},
				outputType: library.TranscodeOutputAudio,
			}

			args, err := buildFFmpegTranscodeArgs(plan, "/tmp/input.mp4", "/tmp/output."+testCase.format, "", "", "", "none")
			if err != nil {
				t.Fatalf("buildFFmpegTranscodeArgs returned error: %v", err)
			}

			joined := strings.Join(args, " ")
			if !strings.Contains(joined, testCase.expectedCodec) {
				t.Fatalf("expected %q in args, got %q", testCase.expectedCodec, joined)
			}
		})
	}
}

func TestBuildTranscodeOperationOutputIncludesSubtitleFields(t *testing.T) {
	request := dto.CreateTranscodeJobRequest{
		SubtitleHandling:         "embed",
		DisplayMode:              "dual",
		SubtitleDocumentID:       "legacy-document-id",
		SubtitleFileID:           "sub-primary",
		SecondarySubtitleFileID:  "sub-secondary",
		GeneratedSubtitleFormat:  "srt",
		GeneratedSubtitleName:    "Workspace subtitles",
		GeneratedSubtitleContent: "1\n00:00:00,000 --> 00:00:01,000\nHello",
	}

	outputJSON := buildTranscodeOperationOutput(request, "completed", "/tmp/output.mp4")

	payload := map[string]any{}
	if err := json.Unmarshal([]byte(outputJSON), &payload); err != nil {
		t.Fatalf("buildTranscodeOperationOutput returned invalid json: %v", err)
	}

	if got := payload["subtitleHandling"]; got != "embed" {
		t.Fatalf("unexpected subtitleHandling: %#v", got)
	}
	if got := payload["displayMode"]; got != "dual" {
		t.Fatalf("unexpected displayMode: %#v", got)
	}
	if got := payload["subtitleDocumentId"]; got != "legacy-document-id" {
		t.Fatalf("unexpected subtitleDocumentId: %#v", got)
	}
	if got := payload["embeddedSubtitleFormat"]; got != "srt" {
		t.Fatalf("unexpected embeddedSubtitleFormat: %#v", got)
	}
}

func TestEnsureManagedOutputParentDirCreatesDirectory(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "libraries", "lib-1", "video.mkv")

	if err := ensureManagedOutputParentDir(outputPath); err != nil {
		t.Fatalf("ensureManagedOutputParentDir returned error: %v", err)
	}

	info, err := os.Stat(filepath.Dir(outputPath))
	if err != nil {
		t.Fatalf("expected parent directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected parent path to be a directory")
	}
}

func TestWithFFmpegProgressArgsInsertsFlagsBeforeOutputPath(t *testing.T) {
	args := []string{"-y", "-i", "/tmp/input.mp4", "-c:v", "libx264", "/tmp/output.mp4"}

	got := withFFmpegProgressArgs(args)
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "-progress pipe:1 -nostats /tmp/output.mp4") {
		t.Fatalf("expected progress flags before output path, got %q", joined)
	}
	if strings.Count(joined, "-progress") != 1 {
		t.Fatalf("expected exactly one progress flag, got %q", joined)
	}
}

func TestFFmpegProgressReporterPublishesPercentFromStructuredOutput(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	operation, err := library.NewLibraryOperation(library.LibraryOperationParams{
		ID:          "op-transcode",
		LibraryID:   "lib-1",
		Kind:        "transcode",
		Status:      string(library.OperationStatusRunning),
		DisplayName: "demo",
		InputJSON:   "{}",
		OutputJSON:  "{}",
		CreatedAt:   &now,
	})
	if err != nil {
		t.Fatalf("new operation: %v", err)
	}

	repo := &transcodeTestOperationRepo{}
	service := &LibraryService{
		operations: repo,
		nowFunc:    func() time.Time { return now },
	}
	reporter := newFFmpegProgressReporter(service, &operation, 10000)

	reporter.HandleLine("out_time=00:00:05.000000")
	reporter.HandleLine("speed=1.5x")
	reporter.HandleLine("progress=continue")

	if len(repo.saved) != 1 {
		t.Fatalf("expected one progress save, got %d", len(repo.saved))
	}
	progress := repo.saved[0].Progress
	if progress == nil || progress.Percent == nil || *progress.Percent != 50 {
		t.Fatalf("expected 50%% progress, got %#v", progress)
	}
	if progress.Current == nil || *progress.Current != 5000 {
		t.Fatalf("expected current progress 5000ms, got %#v", progress)
	}
	if progress.Total == nil || *progress.Total != 10000 {
		t.Fatalf("expected total progress 10000ms, got %#v", progress)
	}
	if progress.Speed != "1.5x" {
		t.Fatalf("expected speed 1.5x, got %#v", progress)
	}
	if progress.Stage != progressText("library.progress.transcoding") {
		t.Fatalf("expected stage key %q, got %#v", progressText("library.progress.transcoding"), progress)
	}
}

func TestParseFFmpegProgressMillisSupportsTimestampAndMicroseconds(t *testing.T) {
	if got, ok := parseFFmpegProgressMillis("out_time", "00:00:04.500000"); !ok || got != 4500 {
		t.Fatalf("expected timestamp progress 4500ms, got %d, %v", got, ok)
	}
	if got, ok := parseFFmpegProgressMillis("out_time_ms", "4500000"); !ok || got != 4500 {
		t.Fatalf("expected microsecond progress 4500ms, got %d, %v", got, ok)
	}
	if got, ok := parseFFmpegProgressMillis("out_time_us", "7200000"); !ok || got != 7200 {
		t.Fatalf("expected microsecond progress 7200ms, got %d, %v", got, ok)
	}
}

func TestNormalizeBurninASSFontForChineseReplacesArialStyle(t *testing.T) {
	content := "[Script Info]\n[V4+ Styles]\nStyle: Default,Arial,48,&H00FFFFFF,&H00FFFFFF,&H00111111,&HFF111111,-1,0,0,0,100,100,0,0,1,2,0,2,72,72,56,1\n[Events]\nDialogue: 0,0:00:00.00,0:00:01.00,Default,,0,0,0,,测试字幕\n"
	got := normalizeBurninASSFontForChinese(content)

	switch runtime.GOOS {
	case "windows":
		if !strings.Contains(got, "Style: Default,Microsoft YaHei,48,") {
			t.Fatalf("expected Windows CJK fallback font in style line, got %q", got)
		}
	case "darwin":
		if !strings.Contains(got, "Style: Default,PingFang SC,48,") {
			t.Fatalf("expected macOS CJK fallback font in style line, got %q", got)
		}
	default:
		if !strings.Contains(got, "Style: Default,Noto Sans CJK SC,48,") {
			t.Fatalf("expected Linux CJK fallback font in style line, got %q", got)
		}
	}
}

func TestNormalizeBurninASSFontForChineseLeavesNonChineseUntouched(t *testing.T) {
	content := "[Script Info]\n[V4+ Styles]\nStyle: Default,Arial,48,&H00FFFFFF,&H00FFFFFF,&H00111111,&HFF111111,-1,0,0,0,100,100,0,0,1,2,0,2,72,72,56,1\n[Events]\nDialogue: 0,0:00:00.00,0:00:01.00,Default,,0,0,0,,hello world\n"
	got := normalizeBurninASSFontForChinese(content)
	if got != content {
		t.Fatalf("expected non-Chinese subtitles to remain unchanged")
	}
}

func TestFFmpegIsHardwareVideoCodec(t *testing.T) {
	if !ffmpegIsHardwareVideoCodec("h264_nvenc") {
		t.Fatalf("expected nvenc to be recognized as hardware codec")
	}
	if ffmpegIsHardwareVideoCodec("libx264") {
		t.Fatalf("did not expect libx264 to be recognized as hardware codec")
	}
}

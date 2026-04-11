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
			Scale:      "1080p",
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
	if !strings.Contains(joined, "-vf ass=filename='/tmp/subtitles\\:demo\\,track\\[1\\]\\'s.ass',scale=w=1920:h=1080:force_original_aspect_ratio=decrease:force_divisible_by=2,pad=1920:1080:(ow-iw)/2:(oh-ih)/2") {
		t.Fatalf("expected escaped ass filter in args, got %q", joined)
	}
	if !strings.Contains(joined, "-map 0:v:0? -map 0:a:0?") {
		t.Fatalf("expected explicit primary stream mapping, got %q", joined)
	}
	if !strings.Contains(joined, "-c:v libx264") {
		t.Fatalf("expected filtered copy job to fall back to libx264, got %q", joined)
	}
	if !strings.Contains(joined, "-c:a copy") {
		t.Fatalf("expected audio copy to remain unchanged, got %q", joined)
	}
	if !strings.Contains(joined, "-movflags +faststart") {
		t.Fatalf("expected mp4 output to use faststart, got %q", joined)
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
	if !strings.Contains(joined, "-map 0:a:0? -vn") {
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
	if !strings.Contains(joined, "-nostdin -progress pipe:1 -nostats /tmp/output.mp4") {
		t.Fatalf("expected progress flags before output path, got %q", joined)
	}
	if strings.Count(joined, "-progress") != 1 {
		t.Fatalf("expected exactly one progress flag, got %q", joined)
	}
}

func TestBuildFFmpegTranscodeArgsVP9UsesLibvpxBestPracticeFlags(t *testing.T) {
	plan := transcodePlan{
		request: dto.CreateTranscodeJobRequest{
			Format:      "webm",
			VideoCodec:  "vp9",
			AudioCodec:  "opus",
			QualityMode: "crf",
			CRF:         defaultVP9VideoCRF,
		},
		outputType: library.TranscodeOutputVideo,
	}

	args, err := buildFFmpegTranscodeArgs(
		plan,
		"/tmp/input.mp4",
		"/tmp/output.webm",
		"",
		"",
		"",
		"none",
	)
	if err != nil {
		t.Fatalf("buildFFmpegTranscodeArgs returned error: %v", err)
	}

	joined := strings.Join(args, " ")
	if strings.Contains(joined, "-preset ") {
		t.Fatalf("did not expect libvpx-vp9 to receive x264/x265 preset flags, got %q", joined)
	}
	if !strings.Contains(joined, "-deadline good -cpu-used 0 -row-mt 1") {
		t.Fatalf("expected libvpx-vp9 best-practice quality flags, got %q", joined)
	}
	if !strings.Contains(joined, "-b:v 0 -crf 20") {
		t.Fatalf("expected VP9 CRF mode to disable target bitrate, got %q", joined)
	}
}

func TestBuildFFmpegTranscodeArgsLosslessAudioSkipsBitrateFlags(t *testing.T) {
	plan := transcodePlan{
		request: dto.CreateTranscodeJobRequest{
			Format:           "flac",
			AudioCodec:       "flac",
			AudioBitrateKbps: 320,
		},
		outputType: library.TranscodeOutputAudio,
	}

	args, err := buildFFmpegTranscodeArgs(plan, "/tmp/input.wav", "/tmp/output.flac", "", "", "", "none")
	if err != nil {
		t.Fatalf("buildFFmpegTranscodeArgs returned error: %v", err)
	}

	joined := strings.Join(args, " ")
	if strings.Contains(joined, "-b:a") {
		t.Fatalf("did not expect bitrate flags for lossless audio, got %q", joined)
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

func TestResolveSourceVideoSubtitlePlayResUsesSourceCanvas(t *testing.T) {
	probe := mediaProbe{Width: 3840, Height: 2160}

	width, height := resolveSourceVideoSubtitlePlayRes(probe)
	if width != 3840 || height != 2160 {
		t.Fatalf("expected subtitle playres to use source probe dimensions, got %dx%d", width, height)
	}
}

func TestNormalizeGeneratedASSPlayResForTranscodeOverridesScriptInfo(t *testing.T) {
	content := strings.Join([]string{
		"[Script Info]",
		"Title: Workspace subtitles",
		"ScriptType: v4.00+",
		"PlayResX: 1920",
		"PlayResY: 1080",
		"",
		"[V4+ Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
		"Style: Primary,PingFang SC,52,&H00FFFFFF,&H00FFFFFF,&H00101010,&H80000000,-1,0,0,0,100,100,0,0,1,0,5,2,72,72,72,1",
		"",
		"[Events]",
		"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
		"Dialogue: 0,0:00:00.00,0:00:01.00,Primary,,0,0,0,,Hello",
		"",
	}, "\n")

	got := normalizeGeneratedASSPlayResForTranscode(content, 3840, 2160)

	if !strings.Contains(got, "PlayResX: 3840") {
		t.Fatalf("expected PlayResX override, got %q", got)
	}
	if !strings.Contains(got, "PlayResY: 2160") {
		t.Fatalf("expected PlayResY override, got %q", got)
	}
	if strings.Contains(got, "PlayResX: 1920") || strings.Contains(got, "PlayResY: 1080") {
		t.Fatalf("expected original playres to be replaced, got %q", got)
	}
	if !strings.Contains(got, "Style: Primary,PingFang SC,104,") {
		t.Fatalf("expected style fontsize to scale with playres, got %q", got)
	}
	if !strings.Contains(got, ",1,0,10,2,144,144,144,1") {
		t.Fatalf("expected outline, shadow, and margins to scale with playres, got %q", got)
	}
	if !strings.Contains(got, "Dialogue: 0,0:00:00.00,0:00:01.00,Primary,,0,0,0,,Hello") {
		t.Fatalf("expected events to remain intact, got %q", got)
	}
}

func TestResolveGeneratedSubtitleContentForTranscodeRendersASSFromDocumentUsingSourceCanvas(t *testing.T) {
	styleDocument := strings.Join([]string{
		"[Script Info]",
		"Title: Workspace subtitles",
		"ScriptType: v4.00+",
		"PlayResX: 1920",
		"PlayResY: 1080",
		"",
		"[V4+ Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
		"Style: Primary,PingFang SC,52,&H00FFFFFF,&H00FFFFFF,&H00101010,&H80000000,-1,0,0,0,100,100,0,0,1,0,5,2,72,72,72,1",
		"",
		"[Events]",
		"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
		"",
	}, "\n")
	request := dto.CreateTranscodeJobRequest{
		Title:                                 "Workspace export",
		GeneratedSubtitleFormat:               "ass",
		GeneratedSubtitleName:                 "Workspace subtitles",
		GeneratedSubtitleStyleDocumentContent: styleDocument,
		GeneratedSubtitleDocument: &dto.SubtitleDocument{
			Format: "srt",
			Cues: []dto.SubtitleCue{
				{
					Index: 1,
					Start: "00:00:00,000",
					End:   "00:00:01,000",
					Text:  "Hello",
				},
			},
		},
	}

	got := resolveGeneratedSubtitleContentForTranscode(request, mediaProbe{Width: 3840, Height: 2160}, "ass")
	if !strings.Contains(got, "PlayResX: 3840") || !strings.Contains(got, "PlayResY: 2160") {
		t.Fatalf("expected generated ass to use source video canvas, got %q", got)
	}
	if !strings.Contains(got, "Style: Primary,PingFang SC,104,") {
		t.Fatalf("expected generated ass to scale style fontsize to 104, got %q", got)
	}
	if !strings.Contains(got, "Dialogue: 0,0:00:00.00,0:00:01.00,Primary,,0,0,0,,Hello") {
		t.Fatalf("expected generated ass to render dialogue from subtitle document, got %q", got)
	}
}

func TestResolveGeneratedSubtitleContentForTranscodeMatchesExportSubtitleASSAtSourceCanvas(t *testing.T) {
	t.Parallel()

	styleDocument := strings.Join([]string{
		"[Script Info]",
		"Title: Workspace subtitles",
		"ScriptType: v4.00+",
		"PlayResX: 1920",
		"PlayResY: 1080",
		"",
		"[V4+ Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
		"Style: Primary,PingFang SC,52,&H00FFFFFF,&H00FFFFFF,&H00101010,&H80000000,-1,0,0,0,100,100,0,0,1,0,5,2,72,72,72,1",
		"",
		"[Events]",
		"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
		"",
	}, "\n")
	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{
				Index: 1,
				Start: "00:00:00,000",
				End:   "00:00:01,000",
				Text:  "Hello 4K",
			},
		},
	}
	request := dto.CreateTranscodeJobRequest{
		Title:                                 "Workspace export",
		GeneratedSubtitleFormat:               "ass",
		GeneratedSubtitleName:                 "Workspace subtitles",
		GeneratedSubtitleStyleDocumentContent: styleDocument,
		GeneratedSubtitleDocument:             &document,
	}

	exportedASS := renderSubtitleContentWithConfig(document, "ass", &dto.SubtitleExportConfig{
		ASS: &dto.SubtitleASSExportConfig{
			PlayResX: 3840,
			PlayResY: 2160,
			Title:    "Workspace subtitles",
		},
	}, styleDocument)
	renderedASS := resolveGeneratedSubtitleContentForTranscode(
		request,
		mediaProbe{Width: 3840, Height: 2160},
		"ass",
	)

	if exportedASS != renderedASS {
		t.Fatalf("expected burn-in ass to match exported ass\nexported=%s\nrendered=%s", exportedASS, renderedASS)
	}
	if got := requireASSSectionValue(t, renderedASS, "[Script Info]", "PlayResX"); got != "3840" {
		t.Fatalf("expected 4k playres x, got %q", got)
	}
	if got := requireASSSectionValue(t, renderedASS, "[Script Info]", "PlayResY"); got != "2160" {
		t.Fatalf("expected 4k playres y, got %q", got)
	}
	if got := requireASSStyleField(t, renderedASS, "[V4+ Styles]", "fontsize"); got != "104" {
		t.Fatalf("expected 1080p 52px style to scale to 104px at 4k, got %q", got)
	}
}

func TestBurninASSTranscodeUsesSourceCanvasAndScalesAfterSubtitles(t *testing.T) {
	content := strings.Join([]string{
		"[Script Info]",
		"Title: Workspace subtitles",
		"ScriptType: v4.00+",
		"PlayResX: 1920",
		"PlayResY: 1080",
		"",
		"[V4+ Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
		"Style: Primary,PingFang SC,52,&H00FFFFFF,&H00FFFFFF,&H00101010,&H80000000,-1,0,0,0,100,100,0,0,1,0,5,2,72,72,72,1",
		"",
		"[Events]",
		"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
		"Dialogue: 0,0:00:00.00,0:00:01.00,Primary,,0,0,0,,Hello",
		"",
	}, "\n")

	normalized := normalizeGeneratedASSPlayResForTranscode(content, 3840, 2160)
	if !strings.Contains(normalized, "PlayResX: 3840") || !strings.Contains(normalized, "PlayResY: 2160") {
		t.Fatalf("expected generated ass to be normalized against source video canvas, got %q", normalized)
	}
	if !strings.Contains(normalized, "Style: Primary,PingFang SC,104,") {
		t.Fatalf("expected source-video normalization to scale fontsize to 104, got %q", normalized)
	}

	args, err := buildFFmpegTranscodeArgs(
		transcodePlan{
			request: dto.CreateTranscodeJobRequest{
				Format:     "mp4",
				VideoCodec: "h264",
				AudioCodec: "aac",
				Preset:     "medium",
				CRF:        23,
				Scale:      "1080p",
			},
			outputType: library.TranscodeOutputVideo,
		},
		"/tmp/input.mp4",
		"/tmp/output.mp4",
		"/tmp/subtitles.ass",
		"",
		"",
		"burnin",
	)
	if err != nil {
		t.Fatalf("buildFFmpegTranscodeArgs returned error: %v", err)
	}

	joined := strings.Join(args, " ")
	assIndex := strings.Index(joined, "ass=filename='/tmp/subtitles.ass'")
	scaleIndex := strings.Index(joined, "scale=w=1920:h=1080:force_original_aspect_ratio=decrease:force_divisible_by=2,pad=1920:1080:(ow-iw)/2:(oh-ih)/2")
	if assIndex < 0 || scaleIndex < 0 {
		t.Fatalf("expected burnin args to include ass and scale filters, got %q", joined)
	}
	if assIndex > scaleIndex {
		t.Fatalf("expected burnin args to apply subtitles before scaling, got %q", joined)
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

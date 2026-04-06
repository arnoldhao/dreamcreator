package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	settingsdto "dreamcreator/internal/application/settings/dto"
	"dreamcreator/internal/domain/externaltools"
	"dreamcreator/internal/domain/library"
)

type mediaProbe struct {
	Format      string
	Codec       string
	VideoCodec  string
	AudioCodec  string
	DurationMs  int64
	Width       int
	Height      int
	FrameRate   float64
	BitrateKbps int
	Channels    int
	SizeBytes   int64
}

type ffprobePayload struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecType    string            `json:"codec_type"`
	CodecName    string            `json:"codec_name"`
	Width        int               `json:"width"`
	Height       int               `json:"height"`
	Channels     int               `json:"channels"`
	AvgFrameRate string            `json:"avg_frame_rate"`
	RFrameRate   string            `json:"r_frame_rate"`
	BitRate      string            `json:"bit_rate"`
	Tags         map[string]string `json:"tags"`
}

type ffprobeFormat struct {
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	BitRate    string `json:"bit_rate"`
}

func (probe mediaProbe) toMediaInfo() library.MediaInfo {
	result := library.MediaInfo{
		Format:     probe.Format,
		Codec:      probe.Codec,
		VideoCodec: probe.VideoCodec,
		AudioCodec: probe.AudioCodec,
	}
	if probe.DurationMs > 0 {
		value := probe.DurationMs
		result.DurationMs = &value
	}
	if probe.Width > 0 {
		value := probe.Width
		result.Width = &value
	}
	if probe.Height > 0 {
		value := probe.Height
		result.Height = &value
	}
	if probe.FrameRate > 0 {
		value := probe.FrameRate
		result.FrameRate = &value
	}
	if probe.BitrateKbps > 0 {
		value := probe.BitrateKbps
		result.BitrateKbps = &value
	}
	if probe.Channels > 0 {
		value := probe.Channels
		result.Channels = &value
	}
	if probe.SizeBytes > 0 {
		value := probe.SizeBytes
		result.SizeBytes = &value
	}
	return result
}

func libraryBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Downloads", "dreamcreator"), nil
}

func (service *LibraryService) resolveDownloadDirectory(ctx context.Context) (string, error) {
	if service != nil && service.settings != nil {
		settings, err := service.settings.GetSettings(ctx)
		if err == nil {
			if trimmed := strings.TrimSpace(settings.DownloadDirectory); trimmed != "" {
				return trimmed, nil
			}
		}
	}
	baseDir, err := libraryBaseDir()
	if err != nil {
		return "", err
	}
	return baseDir, nil
}

func (service *LibraryService) resolveInputPath(ctx context.Context, rawPath string, source string, allowTemp bool) (string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}
	if service != nil && service.isAgentSource(source) {
		return service.resolveAgentPath(ctx, trimmed, allowTemp, true)
	}
	resolved, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func (service *LibraryService) probeLocalMedia(ctx context.Context, path string) mediaProbe {
	fallback := probeLocalMedia(path)
	if service == nil {
		return fallback
	}
	if isSubtitleFormat(fallback.Format) {
		return fallback
	}
	probe, err := service.ffprobeLocalMedia(ctx, path)
	if err != nil {
		return fallback
	}
	return mergeMediaProbe(fallback, probe)
}

func (service *LibraryService) probeRequiredMedia(ctx context.Context, path string) (mediaProbe, error) {
	fallback := probeLocalMedia(path)
	if isSubtitleFormat(fallback.Format) {
		return fallback, nil
	}
	if service == nil || service.tools == nil {
		return mediaProbe{}, fmt.Errorf("ffmpeg is not installed")
	}
	probe, err := service.ffprobeLocalMedia(ctx, path)
	if err != nil {
		return mediaProbe{}, err
	}
	return mergeMediaProbe(fallback, probe), nil
}

func probeLocalMedia(path string) mediaProbe {
	resolved := strings.TrimSpace(path)
	probe := mediaProbe{Format: normalizeTranscodeFormat(filepath.Ext(resolved))}
	if info, err := os.Stat(resolved); err == nil {
		probe.SizeBytes = info.Size()
	}
	switch probe.Format {
	case "mp3", "m4a", "wav", "flac", "aac", "opus", "ogg":
		probe.AudioCodec = probe.Format
		probe.Codec = probe.Format
	case "srt", "vtt", "ass", "ssa", "ttml", "xml":
		probe.Codec = probe.Format
	default:
		probe.VideoCodec = probe.Format
		probe.AudioCodec = "aac"
		probe.Codec = probe.Format
	}
	return probe
}

func (service *LibraryService) ffprobeLocalMedia(ctx context.Context, path string) (mediaProbe, error) {
	execPath, err := resolveFFprobeExecPath(ctx, service.tools)
	if err != nil {
		return mediaProbe{}, err
	}
	command := exec.CommandContext(ctx, execPath,
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		strings.TrimSpace(path),
	)
	configureProcessGroup(command)
	output, err := command.Output()
	if err != nil {
		return mediaProbe{}, err
	}
	return parseFFprobeMediaProbe(output, path)
}

func parseFFprobeMediaProbe(output []byte, path string) (mediaProbe, error) {
	payload := ffprobePayload{}
	if err := json.Unmarshal(output, &payload); err != nil {
		return mediaProbe{}, err
	}
	result := mediaProbe{
		Format: normalizeFFprobeFormat(payload.Format.FormatName, path),
	}
	if result.SizeBytes == 0 {
		result.SizeBytes = parseFFprobeSize(payload.Format.Size)
	}
	if result.DurationMs == 0 {
		result.DurationMs = parseFFprobeDurationMillis(payload.Format.Duration)
	}
	if result.BitrateKbps == 0 {
		result.BitrateKbps = parseFFprobeBitrateKbps(payload.Format.BitRate)
	}
	for _, stream := range payload.Streams {
		switch strings.ToLower(strings.TrimSpace(stream.CodecType)) {
		case "video":
			if result.VideoCodec == "" {
				result.VideoCodec = normalizeTranscodeFormat(stream.CodecName)
			}
			if result.Width == 0 && stream.Width > 0 {
				result.Width = stream.Width
			}
			if result.Height == 0 && stream.Height > 0 {
				result.Height = stream.Height
			}
			if result.FrameRate == 0 {
				result.FrameRate = parseFFprobeFrameRate(firstNonEmpty(stream.AvgFrameRate, stream.RFrameRate))
			}
			if result.BitrateKbps == 0 {
				result.BitrateKbps = parseFFprobeBitrateKbps(stream.BitRate)
			}
		case "audio":
			if result.AudioCodec == "" {
				result.AudioCodec = normalizeTranscodeFormat(stream.CodecName)
			}
			if result.Channels == 0 && stream.Channels > 0 {
				result.Channels = stream.Channels
			}
			if result.BitrateKbps == 0 {
				result.BitrateKbps = parseFFprobeBitrateKbps(stream.BitRate)
			}
		}
	}
	if result.Codec == "" {
		result.Codec = firstNonEmpty(result.VideoCodec, result.AudioCodec)
	}
	if result.SizeBytes == 0 {
		if info, err := os.Stat(strings.TrimSpace(path)); err == nil {
			result.SizeBytes = info.Size()
		}
	}
	return result, nil
}

func mergeMediaProbe(base mediaProbe, override mediaProbe) mediaProbe {
	result := base
	if strings.TrimSpace(override.Format) != "" {
		result.Format = override.Format
	}
	if strings.TrimSpace(override.Codec) != "" {
		result.Codec = override.Codec
	}
	if strings.TrimSpace(override.VideoCodec) != "" {
		result.VideoCodec = override.VideoCodec
	}
	if strings.TrimSpace(override.AudioCodec) != "" {
		result.AudioCodec = override.AudioCodec
	}
	if override.DurationMs > 0 {
		result.DurationMs = override.DurationMs
	}
	if override.Width > 0 {
		result.Width = override.Width
	}
	if override.Height > 0 {
		result.Height = override.Height
	}
	if override.FrameRate > 0 {
		result.FrameRate = override.FrameRate
	}
	if override.BitrateKbps > 0 {
		result.BitrateKbps = override.BitrateKbps
	}
	if override.Channels > 0 {
		result.Channels = override.Channels
	}
	if override.SizeBytes > 0 {
		result.SizeBytes = override.SizeBytes
	}
	if strings.TrimSpace(result.Codec) == "" {
		result.Codec = firstNonEmpty(result.VideoCodec, result.AudioCodec)
	}
	return result
}

func normalizeFFprobeFormat(formatName string, path string) string {
	if extension := normalizeFileExtension(path); extension != "" {
		return extension
	}
	for _, candidate := range strings.Split(strings.TrimSpace(formatName), ",") {
		if trimmed := normalizeTranscodeFormat(candidate); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseFFprobeDurationMillis(value string) int64 {
	durationSeconds, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || durationSeconds <= 0 {
		return 0
	}
	return int64(durationSeconds * 1000)
}

func parseFFprobeBitrateKbps(value string) int {
	bitrate, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || bitrate <= 0 {
		return 0
	}
	return int((bitrate + 500) / 1000)
}

func parseFFprobeSize(value string) int64 {
	size, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || size <= 0 {
		return 0
	}
	return size
}

func parseFFprobeFrameRate(value string) float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	if strings.Contains(trimmed, "/") {
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) != 2 {
			return 0
		}
		numerator, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil || numerator <= 0 {
			return 0
		}
		denominator, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil || denominator <= 0 {
			return 0
		}
		return numerator / denominator
	}
	frameRate, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || frameRate <= 0 {
		return 0
	}
	return frameRate
}

func normalizeTranscodeFormat(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	trimmed = strings.TrimPrefix(trimmed, ".")
	return trimmed
}

func normalizeFileExtension(path string) string {
	return normalizeTranscodeFormat(filepath.Ext(strings.TrimSpace(path)))
}

func isSubtitleFormat(format string) bool {
	switch normalizeSubtitleFormat(format) {
	case "srt", "vtt", "ass", "ssa", "itt", "fcpxml":
		return true
	default:
		return false
	}
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(strings.TrimSpace(path))
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func copyLocalFile(sourcePath string, targetPath string) error {
	source, err := os.Open(strings.TrimSpace(sourcePath))
	if err != nil {
		return err
	}
	defer source.Close()
	if err := ensureParentDir(targetPath); err != nil {
		return err
	}
	target, err := os.Create(strings.TrimSpace(targetPath))
	if err != nil {
		return err
	}
	defer target.Close()
	if _, err := io.Copy(target, source); err != nil {
		return err
	}
	return target.Sync()
}

func resolveFFmpegDir(ctx context.Context, resolver ToolResolver) string {
	if resolver == nil {
		return ""
	}
	dir, err := resolver.ResolveToolDirectory(ctx, externaltools.ToolFFmpeg)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(dir)
}

func resolveFFprobeExecPath(ctx context.Context, resolver ToolResolver) (string, error) {
	if resolver == nil {
		return "", fmt.Errorf("ffmpeg is not installed")
	}
	ready, reason, err := resolver.ToolReadiness(ctx, externaltools.ToolFFmpeg)
	if err != nil {
		return "", err
	}
	if !ready {
		switch strings.TrimSpace(reason) {
		case "invalid":
			return "", fmt.Errorf("ffmpeg is invalid")
		case "ffprobe_not_found":
			return "", fmt.Errorf("ffprobe is not installed")
		case "missing_exec_path", "exec_not_found", "not_found", "not_installed", "":
			return "", fmt.Errorf("ffmpeg is not installed")
		default:
			return "", fmt.Errorf("ffmpeg is not ready: %s", reason)
		}
	}
	dir, err := resolver.ResolveToolDirectory(ctx, externaltools.ToolFFmpeg)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(strings.TrimSpace(dir), ffprobeExecutableName())
	info, statErr := os.Stat(candidate)
	if statErr != nil || info.IsDir() {
		return "", fmt.Errorf("ffprobe is not installed")
	}
	return candidate, nil
}

func resolveFFmpegExecPath(ctx context.Context, resolver ToolResolver) (string, error) {
	if resolver == nil {
		return "", fmt.Errorf("ffmpeg is not installed")
	}
	ready, reason, err := resolver.ToolReadiness(ctx, externaltools.ToolFFmpeg)
	if err != nil {
		return "", err
	}
	if !ready {
		switch strings.TrimSpace(reason) {
		case "invalid":
			return "", fmt.Errorf("ffmpeg is invalid")
		case "ffprobe_not_found":
			return "", fmt.Errorf("ffprobe is not installed")
		case "missing_exec_path", "exec_not_found", "not_found", "not_installed", "":
			return "", fmt.Errorf("ffmpeg is not installed")
		default:
			return "", fmt.Errorf("ffmpeg is not ready: %s", reason)
		}
	}
	dir, err := resolver.ResolveToolDirectory(ctx, externaltools.ToolFFmpeg)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(strings.TrimSpace(dir), ffmpegExecutableName())
	info, statErr := os.Stat(candidate)
	if statErr != nil || info.IsDir() {
		return "", fmt.Errorf("ffmpeg is not installed")
	}
	return candidate, nil
}

func ffmpegExecutableName() string {
	if runtime.GOOS == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

func ffprobeExecutableName() string {
	if runtime.GOOS == "windows" {
		return "ffprobe.exe"
	}
	return "ffprobe"
}

func settingsDownloadDirectory(settings settingsdto.Settings) string {
	return strings.TrimSpace(settings.DownloadDirectory)
}

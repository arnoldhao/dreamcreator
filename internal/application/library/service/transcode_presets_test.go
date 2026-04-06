package service

import (
	"testing"
	"time"
)

func TestDefaultTranscodePresetsExposeExpandedBuiltinSet(t *testing.T) {
	presets := defaultTranscodePresets(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if len(presets) != 52 {
		t.Fatalf("expected 52 builtin transcode presets, got %d", len(presets))
	}

	seen := make(map[string]struct{}, len(presets))
	for _, preset := range presets {
		if _, ok := seen[preset.ID]; ok {
			t.Fatalf("duplicate preset id detected: %s", preset.ID)
		}
		seen[preset.ID] = struct{}{}
	}

	required := []string{
		"builtin-video-h264-mp4-original",
		"builtin-video-h265-mov-2160p",
		"builtin-video-vp9-mkv-1080p",
		"builtin-video-vp9-webm-720p",
		"builtin-audio-mp3-192k",
		"builtin-audio-aac-m4a-256k",
		"builtin-audio-opus-ogg-128k",
		"builtin-audio-flac-lossless",
		"builtin-audio-wav-pcm",
	}
	for _, id := range required {
		if _, ok := seen[id]; !ok {
			t.Fatalf("expected builtin preset %s to exist", id)
		}
	}
}

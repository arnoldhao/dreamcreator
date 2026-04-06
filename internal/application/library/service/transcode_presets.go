package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

const defaultFFmpegPreset = "medium"

type builtinVideoPresetSeries struct {
	idPrefix   string
	namePrefix string
	container  string
	videoCodec string
	audioCodec string
	crf        int
}

type builtinVideoScaleSpec struct {
	idSuffix   string
	nameSuffix string
	scale      string
}

type builtinAudioPresetSpec struct {
	id         string
	name       string
	container  string
	audioCodec string
	bitrate    int
}

var builtinVideoPresetSeriesSpecs = []builtinVideoPresetSeries{
	{idPrefix: "builtin-video-h264-mp4", namePrefix: "H.264 MP4", container: "mp4", videoCodec: "h264", audioCodec: "aac", crf: 23},
	{idPrefix: "builtin-video-h265-mp4", namePrefix: "H.265 MP4", container: "mp4", videoCodec: "h265", audioCodec: "aac", crf: 28},
	{idPrefix: "builtin-video-h264-mov", namePrefix: "H.264 MOV", container: "mov", videoCodec: "h264", audioCodec: "aac", crf: 23},
	{idPrefix: "builtin-video-h265-mov", namePrefix: "H.265 MOV", container: "mov", videoCodec: "h265", audioCodec: "aac", crf: 28},
	{idPrefix: "builtin-video-h264-mkv", namePrefix: "H.264 MKV", container: "mkv", videoCodec: "h264", audioCodec: "aac", crf: 23},
	{idPrefix: "builtin-video-h265-mkv", namePrefix: "H.265 MKV", container: "mkv", videoCodec: "h265", audioCodec: "aac", crf: 28},
	{idPrefix: "builtin-video-vp9-mkv", namePrefix: "VP9 MKV", container: "mkv", videoCodec: "vp9", audioCodec: "opus", crf: 32},
	{idPrefix: "builtin-video-vp9-webm", namePrefix: "VP9 WebM", container: "webm", videoCodec: "vp9", audioCodec: "opus", crf: 32},
}

var builtinVideoScaleSpecs = []builtinVideoScaleSpec{
	{idSuffix: "original", nameSuffix: "Original", scale: "original"},
	{idSuffix: "2160p", nameSuffix: "2160p", scale: "2160p"},
	{idSuffix: "1080p", nameSuffix: "1080p", scale: "1080p"},
	{idSuffix: "720p", nameSuffix: "720p", scale: "720p"},
	{idSuffix: "480p", nameSuffix: "480p", scale: "480p"},
}

var builtinAudioPresetSpecs = []builtinAudioPresetSpec{
	{id: "builtin-audio-mp3-128k", name: "MP3 128k", container: "mp3", audioCodec: "mp3", bitrate: 128},
	{id: "builtin-audio-mp3-192k", name: "MP3 192k", container: "mp3", audioCodec: "mp3", bitrate: 192},
	{id: "builtin-audio-mp3-256k", name: "MP3 256k", container: "mp3", audioCodec: "mp3", bitrate: 256},
	{id: "builtin-audio-mp3-320k", name: "MP3 320k", container: "mp3", audioCodec: "mp3", bitrate: 320},
	{id: "builtin-audio-aac-m4a-128k", name: "AAC M4A 128k", container: "m4a", audioCodec: "aac", bitrate: 128},
	{id: "builtin-audio-aac-m4a-192k", name: "AAC M4A 192k", container: "m4a", audioCodec: "aac", bitrate: 192},
	{id: "builtin-audio-aac-m4a-256k", name: "AAC M4A 256k", container: "m4a", audioCodec: "aac", bitrate: 256},
	{id: "builtin-audio-opus-ogg-96k", name: "Opus OGG 96k", container: "ogg", audioCodec: "opus", bitrate: 96},
	{id: "builtin-audio-opus-ogg-128k", name: "Opus OGG 128k", container: "ogg", audioCodec: "opus", bitrate: 128},
	{id: "builtin-audio-opus-ogg-192k", name: "Opus OGG 192k", container: "ogg", audioCodec: "opus", bitrate: 192},
	{id: "builtin-audio-flac-lossless", name: "FLAC Lossless", container: "flac", audioCodec: "flac"},
	{id: "builtin-audio-wav-pcm", name: "WAV PCM 16-bit", container: "wav", audioCodec: "pcm"},
}

func (service *LibraryService) EnsureDefaultTranscodePresets(ctx context.Context) error {
	if service.presets == nil {
		return nil
	}
	now := service.now()
	defaults := defaultTranscodePresets(now)
	defaultBuiltinIDs := make(map[string]struct{}, len(defaults))
	for _, preset := range defaults {
		defaultBuiltinIDs[preset.ID] = struct{}{}
	}
	existing, err := service.presets.List(ctx)
	if err != nil {
		return err
	}
	for _, preset := range existing {
		if !preset.IsBuiltin {
			continue
		}
		if _, ok := defaultBuiltinIDs[preset.ID]; ok {
			continue
		}
		if err := service.presets.Delete(ctx, preset.ID); err != nil {
			return err
		}
	}
	for _, preset := range defaults {
		if err := service.presets.Save(ctx, preset); err != nil {
			return err
		}
	}
	return nil
}

func (service *LibraryService) ListTranscodePresets(ctx context.Context) ([]dto.TranscodePreset, error) {
	var presets []library.TranscodePreset
	if service.presets == nil {
		presets = defaultTranscodePresets(service.now())
	} else {
		items, err := service.presets.List(ctx)
		if err != nil {
			return nil, err
		}
		presets = items
	}
	result := make([]dto.TranscodePreset, 0, len(presets))
	for _, preset := range presets {
		result = append(result, toTranscodePresetDTO(preset))
	}
	return result, nil
}

func (service *LibraryService) SaveTranscodePreset(ctx context.Context, preset dto.TranscodePreset) (dto.TranscodePreset, error) {
	if service.presets == nil {
		return dto.TranscodePreset{}, fmt.Errorf("transcode preset repository not configured")
	}
	now := service.now()
	id := strings.TrimSpace(preset.ID)
	createdAt := time.Time{}
	if id != "" {
		existing, err := service.presets.Get(ctx, id)
		if err != nil && err != library.ErrPresetNotFound {
			return dto.TranscodePreset{}, err
		}
		if err == nil {
			if existing.IsBuiltin {
				return dto.TranscodePreset{}, fmt.Errorf("builtin preset cannot be modified")
			}
			createdAt = existing.CreatedAt
		}
		if strings.HasPrefix(id, "builtin-") {
			return dto.TranscodePreset{}, fmt.Errorf("preset id is reserved")
		}
	} else {
		id = uuid.NewString()
	}
	model, err := library.NewTranscodePreset(library.TranscodePresetParams{
		ID:               id,
		Name:             preset.Name,
		OutputType:       preset.OutputType,
		Container:        preset.Container,
		VideoCodec:       preset.VideoCodec,
		AudioCodec:       preset.AudioCodec,
		QualityMode:      preset.QualityMode,
		CRF:              preset.CRF,
		BitrateKbps:      preset.BitrateKbps,
		AudioBitrateKbps: preset.AudioBitrateKbps,
		Scale:            preset.Scale,
		Width:            preset.Width,
		Height:           preset.Height,
		FFmpegPreset:     preset.FFmpegPreset,
		AllowUpscale:     preset.AllowUpscale,
		RequiresVideo:    preset.RequiresVideo,
		RequiresAudio:    preset.RequiresAudio,
		IsBuiltin:        false,
		CreatedAt:        timeOrNil(createdAt),
		UpdatedAt:        &now,
	})
	if err != nil {
		return dto.TranscodePreset{}, err
	}
	if err := service.presets.Save(ctx, model); err != nil {
		return dto.TranscodePreset{}, err
	}
	return toTranscodePresetDTO(model), nil
}

func (service *LibraryService) DeleteTranscodePreset(ctx context.Context, request dto.DeleteTranscodePresetRequest) error {
	if service.presets == nil {
		return fmt.Errorf("transcode preset repository not configured")
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return fmt.Errorf("preset id is required")
	}
	preset, err := service.presets.Get(ctx, id)
	if err != nil {
		return err
	}
	if preset.IsBuiltin {
		return fmt.Errorf("builtin preset cannot be deleted")
	}
	return service.presets.Delete(ctx, id)
}

func defaultTranscodePresets(now time.Time) []library.TranscodePreset {
	result := make([]library.TranscodePreset, 0, len(builtinVideoPresetSeriesSpecs)*len(builtinVideoScaleSpecs)+len(builtinAudioPresetSpecs))
	add := func(params library.TranscodePresetParams) {
		params.IsBuiltin = true
		params.CreatedAt = &now
		params.UpdatedAt = &now
		preset, err := library.NewTranscodePreset(params)
		if err == nil {
			result = append(result, preset)
		}
	}
	for _, series := range builtinVideoPresetSeriesSpecs {
		for _, scale := range builtinVideoScaleSpecs {
			add(library.TranscodePresetParams{
				ID:               fmt.Sprintf("%s-%s", series.idPrefix, scale.idSuffix),
				Name:             fmt.Sprintf("%s %s", series.namePrefix, scale.nameSuffix),
				OutputType:       "video",
				Container:        series.container,
				VideoCodec:       series.videoCodec,
				AudioCodec:       series.audioCodec,
				QualityMode:      "crf",
				CRF:              series.crf,
				AudioBitrateKbps: 128,
				Scale:            scale.scale,
				FFmpegPreset:     defaultFFmpegPreset,
				RequiresVideo:    true,
			})
		}
	}
	for _, spec := range builtinAudioPresetSpecs {
		add(library.TranscodePresetParams{
			ID:               spec.id,
			Name:             spec.name,
			OutputType:       "audio",
			Container:        spec.container,
			AudioCodec:       spec.audioCodec,
			AudioBitrateKbps: spec.bitrate,
			RequiresAudio:    true,
		})
	}
	return result
}

func toTranscodePresetDTO(preset library.TranscodePreset) dto.TranscodePreset {
	createdAt := ""
	if !preset.CreatedAt.IsZero() {
		createdAt = preset.CreatedAt.Format(time.RFC3339)
	}
	updatedAt := ""
	if !preset.UpdatedAt.IsZero() {
		updatedAt = preset.UpdatedAt.Format(time.RFC3339)
	}
	return dto.TranscodePreset{
		ID:               preset.ID,
		Name:             preset.Name,
		OutputType:       string(preset.OutputType),
		Container:        preset.Container,
		VideoCodec:       preset.VideoCodec,
		AudioCodec:       preset.AudioCodec,
		QualityMode:      preset.QualityMode,
		CRF:              preset.CRF,
		BitrateKbps:      preset.BitrateKbps,
		AudioBitrateKbps: preset.AudioBitrateKbps,
		Scale:            preset.Scale,
		Width:            preset.Width,
		Height:           preset.Height,
		FFmpegPreset:     preset.FFmpegPreset,
		AllowUpscale:     preset.AllowUpscale,
		RequiresVideo:    preset.RequiresVideo,
		RequiresAudio:    preset.RequiresAudio,
		IsBuiltin:        preset.IsBuiltin,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}

func timeOrNil(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

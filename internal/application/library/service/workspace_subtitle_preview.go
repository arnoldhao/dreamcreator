package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

const workspacePreviewDefaultCueDurationMS = int64(1500)

func (service *LibraryService) GenerateWorkspacePreviewVTT(
	_ context.Context,
	request dto.GenerateWorkspacePreviewVTTRequest,
) (dto.GenerateWorkspacePreviewVTTResult, error) {
	displayMode := normalizeWorkspacePreviewDisplayMode(request.DisplayMode)
	rows := normalizeWorkspacePreviewRows(request.Rows)
	options := previewVTTRenderOptions{
		FontMappings:  request.FontMappings,
		PreviewWidth:  request.PreviewWidth,
		PreviewHeight: request.PreviewHeight,
	}

	if displayMode == "dual" {
		style, err := resolveWorkspacePreviewLingualStyle(request.Lingual, request.Mono)
		if err != nil {
			return dto.GenerateWorkspacePreviewVTTResult{}, err
		}
		vttContent := buildWorkspaceLingualPreviewVTT(style, rows, options)
		return dto.GenerateWorkspacePreviewVTTResult{
			VTTContent: vttContent,
		}, nil
	}

	style, err := resolveWorkspacePreviewMonoStyle(request.Mono)
	if err != nil {
		return dto.GenerateWorkspacePreviewVTTResult{}, err
	}
	vttContent := buildWorkspaceMonoPreviewVTT(style, rows, options)
	return dto.GenerateWorkspacePreviewVTTResult{
		VTTContent: vttContent,
	}, nil
}

func normalizeWorkspacePreviewDisplayMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dual", "bilingual":
		return "dual"
	default:
		return "single"
	}
}

func normalizeWorkspacePreviewRows(values []dto.WorkspacePreviewCueDTO) []dto.WorkspacePreviewCueDTO {
	if len(values) == 0 {
		return nil
	}
	result := make([]dto.WorkspacePreviewCueDTO, 0, len(values))
	for _, value := range values {
		startMS := maxInt64(0, value.StartMS)
		endMS := maxInt64(0, value.EndMS)
		if endMS <= startMS {
			endMS = startMS + workspacePreviewDefaultCueDurationMS
		}
		result = append(result, dto.WorkspacePreviewCueDTO{
			StartMS:       startMS,
			EndMS:         endMS,
			PrimaryText:   value.PrimaryText,
			SecondaryText: value.SecondaryText,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].StartMS == result[j].StartMS {
			return result[i].EndMS < result[j].EndMS
		}
		return result[i].StartMS < result[j].StartMS
	})
	return result
}

func resolveWorkspacePreviewMonoStyle(value *dto.LibraryMonoStyleDTO) (library.MonoStyle, error) {
	if value != nil {
		return normalizePreviewMonoStyle(*value)
	}
	return normalizePreviewMonoStyle(defaultWorkspaceMonoStyleDTO())
}

func resolveWorkspacePreviewLingualStyle(
	lingual *dto.LibraryBilingualStyleDTO,
	mono *dto.LibraryMonoStyleDTO,
) (library.BilingualStyle, error) {
	if lingual != nil {
		return normalizePreviewBilingualStyle(*lingual)
	}
	monoStyle, err := resolveWorkspacePreviewMonoStyle(mono)
	if err != nil {
		return library.BilingualStyle{}, err
	}
	return normalizePreviewBilingualStyle(buildFallbackLingualStyleFromMono(monoStyle))
}

func defaultWorkspaceMonoStyleDTO() dto.LibraryMonoStyleDTO {
	return dto.LibraryMonoStyleDTO{
		ID:              "workspace-mono-default",
		Name:            "Workspace Mono",
		BasePlayResX:    1920,
		BasePlayResY:    1080,
		BaseAspectRatio: "16:9",
		Style: dto.AssStyleSpecDTO{
			Fontname:        "Arial",
			Fontsize:        48,
			PrimaryColour:   "&H00FFFFFF",
			SecondaryColour: "&H00FFFFFF",
			OutlineColour:   "&H00111111",
			BackColour:      "&HFF111111",
			Bold:            false,
			Italic:          false,
			Underline:       false,
			StrikeOut:       false,
			ScaleX:          100,
			ScaleY:          100,
			Spacing:         0,
			Angle:           0,
			BorderStyle:     1,
			Outline:         2,
			Shadow:          0,
			Alignment:       2,
			MarginL:         72,
			MarginR:         72,
			MarginV:         56,
			Encoding:        1,
		},
	}
}

func buildFallbackLingualStyleFromMono(mono library.MonoStyle) dto.LibraryBilingualStyleDTO {
	snapshot := dto.LibraryMonoStyleSnapshotDTO{
		SourceMonoStyleID:   strings.TrimSpace(mono.ID),
		SourceMonoStyleName: strings.TrimSpace(mono.Name),
		Name:                firstNonEmpty(strings.TrimSpace(mono.Name), "Primary"),
		BasePlayResX:        mono.BasePlayResX,
		BasePlayResY:        mono.BasePlayResY,
		BaseAspectRatio:     mono.BaseAspectRatio,
		Style:               toAssStyleSpecDTO(mono.Style),
	}
	return dto.LibraryBilingualStyleDTO{
		ID:              "workspace-lingual-fallback",
		Name:            "Workspace Lingual",
		BasePlayResX:    mono.BasePlayResX,
		BasePlayResY:    mono.BasePlayResY,
		BaseAspectRatio: mono.BaseAspectRatio,
		Primary:         snapshot,
		Secondary:       snapshot,
		Layout: dto.LibraryBilingualLayoutDTO{
			Gap:         24,
			BlockAnchor: 2,
		},
	}
}

func buildWorkspaceMonoPreviewVTT(
	style library.MonoStyle,
	rows []dto.WorkspacePreviewCueDTO,
	options previewVTTRenderOptions,
) string {
	lines := []string{
		"WEBVTT",
		"",
		"STYLE",
		"::cue {",
		"  white-space: pre-line;",
		"}",
	}
	lines = append(lines, buildVTTStyleBlock("mono", style.Style, style.BasePlayResX, style.BasePlayResY, options)...)
	lines = append(lines, "")
	cueSettings := buildVTTCueSettings(style.Style, style.BasePlayResX, style.BasePlayResY)
	for _, row := range rows {
		text := escapeVTTPreviewText(row.PrimaryText)
		if strings.TrimSpace(text) == "" {
			continue
		}
		lines = append(lines,
			fmt.Sprintf("%s --> %s %s", formatVTTTimestamp(row.StartMS), formatVTTTimestamp(row.EndMS), cueSettings),
			fmt.Sprintf("<c.mono>%s</c>", text),
			"",
		)
	}
	return strings.Join(lines, "\n")
}

func buildWorkspaceLingualPreviewVTT(
	style library.BilingualStyle,
	rows []dto.WorkspacePreviewCueDTO,
	options previewVTTRenderOptions,
) string {
	primaryStyle, secondaryStyle := resolveBilingualPreviewStylePair(style)
	lines := []string{
		"WEBVTT",
		"",
		"STYLE",
		"::cue {",
		"  white-space: pre-line;",
		"}",
	}
	lines = append(lines, buildVTTStyleBlock("primary", primaryStyle, style.BasePlayResX, style.BasePlayResY, options)...)
	lines = append(lines, buildVTTStyleBlock("secondary", secondaryStyle, style.BasePlayResX, style.BasePlayResY, options)...)
	lines = append(lines, "")
	primaryCueSettings := buildVTTCueSettings(primaryStyle, style.BasePlayResX, style.BasePlayResY)
	secondaryCueSettings := buildVTTCueSettings(secondaryStyle, style.BasePlayResX, style.BasePlayResY)
	for _, row := range rows {
		primaryText := escapeVTTPreviewText(row.PrimaryText)
		secondaryText := escapeVTTPreviewText(row.SecondaryText)
		primaryEmpty := strings.TrimSpace(primaryText) == ""
		secondaryEmpty := strings.TrimSpace(secondaryText) == ""
		if primaryEmpty && secondaryEmpty {
			continue
		}
		if !primaryEmpty {
			lines = append(lines,
				fmt.Sprintf("%s --> %s %s", formatVTTTimestamp(row.StartMS), formatVTTTimestamp(row.EndMS), primaryCueSettings),
				fmt.Sprintf("<c.primary>%s</c>", primaryText),
				"",
			)
		}
		if !secondaryEmpty {
			lines = append(lines,
				fmt.Sprintf("%s --> %s %s", formatVTTTimestamp(row.StartMS), formatVTTTimestamp(row.EndMS), secondaryCueSettings),
				fmt.Sprintf("<c.secondary>%s</c>", secondaryText),
				"",
			)
		}
	}
	return strings.Join(lines, "\n")
}

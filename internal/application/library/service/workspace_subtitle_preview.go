package service

import (
	"context"
	"fmt"
	"strings"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

func (service *LibraryService) GenerateWorkspacePreviewASS(
	ctx context.Context,
	request dto.GenerateWorkspacePreviewASSRequest,
) (dto.GenerateWorkspacePreviewASSResult, error) {
	libraryID := strings.TrimSpace(request.LibraryID)
	if libraryID == "" {
		return dto.GenerateWorkspacePreviewASSResult{}, fmt.Errorf("library id is required")
	}

	displayMode := normalizeWorkspacePreviewDisplayMode(request.DisplayMode)
	moduleConfig, err := service.getModuleConfig(ctx)
	if err != nil {
		return dto.GenerateWorkspacePreviewASSResult{}, err
	}

	var workspaceHead *dto.WorkspaceStateRecordDTO
	if service.workspace != nil {
		if head, headErr := service.workspace.GetHeadByLibraryID(ctx, libraryID); headErr == nil {
			mapped := toWorkspaceDTO(head)
			workspaceHead = &mapped
		} else if headErr != nil && headErr != library.ErrWorkspaceStateNotFound {
			return dto.GenerateWorkspacePreviewASSResult{}, headErr
		}
	}

	workspaceMonoStyle, workspaceLingualStyle := resolveWorkspaceSubtitleStyles(moduleConfig, workspaceHead)
	fontMappings := toSubtitleStyleFontDTOs(moduleConfig.SubtitleStyles.Fonts)

	primaryFile, primaryDocument, err := service.resolveWorkspacePreviewTrackDocument(
		ctx,
		libraryID,
		request.PrimarySubtitleTrackID,
	)
	if err != nil {
		return dto.GenerateWorkspacePreviewASSResult{}, err
	}

	primaryParsed := parseSubtitleDocument(
		primaryDocument.WorkingContent,
		detectSubtitleFormat(primaryDocument.Format, primaryFile.Storage.LocalPath, primaryDocument.Format),
	)

	if displayMode == "bilingual" {
		secondaryTrackID := strings.TrimSpace(request.SecondarySubtitleTrackID)
		if secondaryTrackID == "" {
			return dto.GenerateWorkspacePreviewASSResult{}, fmt.Errorf("secondary subtitle track id is required for bilingual preview")
		}
		secondaryFile, secondaryDocument, err := service.resolveWorkspacePreviewTrackDocument(ctx, libraryID, secondaryTrackID)
		if err != nil {
			return dto.GenerateWorkspacePreviewASSResult{}, err
		}
		secondaryParsed := parseSubtitleDocument(
			secondaryDocument.WorkingContent,
			detectSubtitleFormat(secondaryDocument.Format, secondaryFile.Storage.LocalPath, secondaryDocument.Format),
		)
		style, err := resolveWorkspacePreviewLingualStyle(workspaceLingualStyle, workspaceMonoStyle)
		if err != nil {
			return dto.GenerateWorkspacePreviewASSResult{}, err
		}
		style = mapPreviewBilingualStyleFontMappings(style, fontMappings)
		styleDocumentContent := buildBilingualStylePreviewASS(style)
		document := buildWorkspacePreviewSubtitleDocument(primaryParsed, secondaryParsed)
		assContent := renderSubtitleContentWithConfig(
			document,
			"ass",
			&dto.SubtitleExportConfig{
				ASS: &dto.SubtitleASSExportConfig{
					PlayResX: style.BasePlayResX,
					PlayResY: style.BasePlayResY,
					Title:    firstNonEmpty(strings.TrimSpace(style.Name), "DreamCreator Workspace Preview"),
				},
			},
			styleDocumentContent,
		)
		return dto.GenerateWorkspacePreviewASSResult{
			ASSContent:             assContent,
			ReferencedFontFamilies: collectWorkspacePreviewFontFamilies(primaryStyleFontSpec(style), secondaryStyleFontSpec(style)),
		}, nil
	}

	style, err := resolveWorkspacePreviewMonoStyle(workspaceMonoStyle)
	if err != nil {
		return dto.GenerateWorkspacePreviewASSResult{}, err
	}
	style = mapPreviewMonoStyleFontMappings(style, fontMappings)
	styleDocumentContent := buildMonoStylePreviewASS(style)
	assContent := renderSubtitleContentWithConfig(
		primaryParsed,
		"ass",
		&dto.SubtitleExportConfig{
			ASS: &dto.SubtitleASSExportConfig{
				PlayResX: style.BasePlayResX,
				PlayResY: style.BasePlayResY,
				Title:    firstNonEmpty(strings.TrimSpace(style.Name), "DreamCreator Workspace Preview"),
			},
		},
		styleDocumentContent,
	)
	return dto.GenerateWorkspacePreviewASSResult{
		ASSContent:             assContent,
		ReferencedFontFamilies: collectWorkspacePreviewFontFamilies(style.Style),
	}, nil
}

func normalizeWorkspacePreviewDisplayMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dual", "bilingual":
		return "bilingual"
	default:
		return "mono"
	}
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
			FontFace:        "Regular",
			FontWeight:      400,
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

func (service *LibraryService) resolveWorkspacePreviewTrackDocument(
	ctx context.Context,
	libraryID string,
	trackID string,
) (library.LibraryFile, library.SubtitleDocument, error) {
	trimmedTrackID := strings.TrimSpace(trackID)
	if trimmedTrackID == "" {
		return library.LibraryFile{}, library.SubtitleDocument{}, fmt.Errorf("subtitle track id is required")
	}
	fileItem, document, err := service.resolveSubtitleFileAndDocument(ctx, trimmedTrackID, "", "")
	if err != nil {
		return library.LibraryFile{}, library.SubtitleDocument{}, err
	}
	if fileItem.LibraryID != libraryID || document.LibraryID != libraryID {
		return library.LibraryFile{}, library.SubtitleDocument{}, fmt.Errorf("subtitle track %q does not belong to library %q", trimmedTrackID, libraryID)
	}
	return fileItem, document, nil
}

func buildWorkspacePreviewSubtitleDocument(
	primary dto.SubtitleDocument,
	secondary dto.SubtitleDocument,
) dto.SubtitleDocument {
	cues := make([]dto.SubtitleCue, 0, len(primary.Cues))
	primaryTexts := make([]string, 0, len(primary.Cues))
	secondaryTexts := make([]string, 0, len(primary.Cues))
	for index, cue := range primary.Cues {
		primaryText := normalizeSubtitleText(cue.Text)
		secondaryText := ""
		if index < len(secondary.Cues) {
			secondaryText = normalizeSubtitleText(secondary.Cues[index].Text)
		}
		primaryTexts = append(primaryTexts, primaryText)
		secondaryTexts = append(secondaryTexts, secondaryText)
		cues = append(cues, dto.SubtitleCue{
			Index: cue.Index,
			Start: cue.Start,
			End:   cue.End,
			Text:  joinSubtitleExportText(primaryText, secondaryText),
		})
	}
	metadata := cloneSubtitleDocumentMetadata(primary.Metadata)
	metadata[subtitleExportDisplayModeKey] = "bilingual"
	metadata[subtitleExportPrimaryTextsKey] = primaryTexts
	metadata[subtitleExportSecondaryTextsKey] = secondaryTexts
	return dto.SubtitleDocument{
		Format:   primary.Format,
		Cues:     cues,
		Metadata: metadata,
	}
}

func cloneSubtitleDocumentMetadata(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(source)+3)
	for key, value := range source {
		result[key] = value
	}
	return result
}

func collectWorkspacePreviewFontFamilies(styles ...library.AssStyleSpec) []string {
	seen := make(map[string]struct{}, len(styles))
	result := make([]string, 0, len(styles))
	for _, style := range styles {
		fontFamily := strings.TrimSpace(resolveASSStyleFontName(style))
		if fontFamily == "" {
			continue
		}
		normalized := strings.ToLower(fontFamily)
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, fontFamily)
	}
	return result
}

func primaryStyleFontSpec(style library.BilingualStyle) library.AssStyleSpec {
	primary, _ := resolveBilingualPreviewStylePair(style)
	return primary
}

func secondaryStyleFontSpec(style library.BilingualStyle) library.AssStyleSpec {
	_, secondary := resolveBilingualPreviewStylePair(style)
	return secondary
}

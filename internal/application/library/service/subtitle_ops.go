package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

func (service *LibraryService) ParseSubtitle(ctx context.Context, request dto.SubtitleParseRequest) (dto.SubtitleParseResult, error) {
	content, format, _, _, err := service.resolveSubtitleContent(ctx, request.FileID, request.DocumentID, request.Path, request.Content, request.Format)
	if err != nil {
		return dto.SubtitleParseResult{}, err
	}
	document := parseSubtitleDocument(content, format)
	return dto.SubtitleParseResult{Format: format, CueCount: len(document.Cues), Document: document}, nil
}

func (service *LibraryService) ConvertSubtitle(ctx context.Context, request dto.SubtitleConvertRequest) (dto.SubtitleConvertResult, error) {
	content, format, _, _, err := service.resolveSubtitleContent(ctx, request.FileID, request.DocumentID, request.Path, request.Content, request.FromFormat)
	if err != nil {
		return dto.SubtitleConvertResult{}, err
	}
	targetFormat := detectSubtitleFormat(request.TargetFormat, "", format)
	if normalizeSubtitleFormat(targetFormat) == normalizeSubtitleFormat(format) {
		return dto.SubtitleConvertResult{TargetFormat: targetFormat, Content: content}, nil
	}
	return dto.SubtitleConvertResult{TargetFormat: targetFormat, Content: renderSubtitleContent(parseSubtitleDocument(content, format), targetFormat)}, nil
}

func (service *LibraryService) ExportSubtitle(ctx context.Context, request dto.SubtitleExportRequest) (dto.SubtitleExportResult, error) {
	exportPath := strings.TrimSpace(request.ExportPath)
	if exportPath == "" {
		return dto.SubtitleExportResult{}, fmt.Errorf("export path is required")
	}
	content, format, err := service.resolveSubtitleValidationContent(
		ctx,
		request.FileID,
		request.DocumentID,
		request.Path,
		request.Content,
		request.Format,
		request.Document,
	)
	if err != nil {
		return dto.SubtitleExportResult{}, err
	}
	targetFormat := detectSubtitleFormat(request.TargetFormat, exportPath, format)
	if request.Document != nil {
		originalContent := firstNonEmpty(
			subtitleDocumentSourceContent(*request.Document),
			content,
		)
		if preserved, ok := renderSubtitleContentPreservingSource(
			*request.Document,
			targetFormat,
			request.ExportConfig,
			request.StyleDocumentContent,
			originalContent,
		); ok && normalizeSubtitleFormat(targetFormat) == normalizeSubtitleFormat(format) {
			content = preserved
		} else {
			content = renderSubtitleContentWithConfig(*request.Document, targetFormat, request.ExportConfig, request.StyleDocumentContent)
		}
	} else if targetFormat != format {
		content = renderSubtitleContentWithConfig(
			parseSubtitleDocument(content, format),
			targetFormat,
			request.ExportConfig,
			request.StyleDocumentContent,
		)
	}
	if err := os.WriteFile(exportPath, []byte(content), 0o644); err != nil {
		return dto.SubtitleExportResult{}, err
	}
	return dto.SubtitleExportResult{ExportPath: exportPath, Format: targetFormat, Bytes: len(content)}, nil
}

func (service *LibraryService) ValidateSubtitle(ctx context.Context, request dto.SubtitleValidateRequest) (dto.SubtitleValidateResult, error) {
	content, format, err := service.resolveSubtitleValidationContent(ctx, request.FileID, request.DocumentID, request.Path, request.Content, request.Format, request.Document)
	if err != nil {
		return dto.SubtitleValidateResult{}, err
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return dto.SubtitleValidateResult{Valid: false, IssueCount: 1, Issues: []dto.SubtitleValidateIssue{{Severity: "error", Code: "empty_content", Message: "subtitle content is empty"}}}, nil
	}
	return validateSubtitleDocument(parseSubtitleDocument(content, format)), nil
}

func (service *LibraryService) FixSubtitleTypos(ctx context.Context, request dto.SubtitleFixTyposRequest) (dto.SubtitleFixTyposResult, error) {
	content, format, _, _, err := service.resolveSubtitleContent(ctx, request.FileID, request.DocumentID, request.Path, request.Content, request.Format)
	if err != nil && request.Document == nil {
		return dto.SubtitleFixTyposResult{}, err
	}
	document := dto.SubtitleDocument{}
	if request.Document != nil {
		document = *request.Document
		if strings.TrimSpace(document.Format) == "" {
			document.Format = format
		}
	} else {
		document = parseSubtitleDocument(content, format)
	}
	document, changes := normalizeSubtitleDocumentText(document)
	normalizedContent := renderSubtitleContent(document, detectSubtitleFormat(document.Format, "", format))
	if preserved, ok := renderSubtitleContentPreservingSource(
		document,
		firstNonEmpty(document.Format, format),
		nil,
		"",
		firstNonEmpty(subtitleDocumentSourceContent(document), content),
	); ok && normalizeSubtitleFormat(document.Format) == normalizeSubtitleFormat(format) {
		normalizedContent = preserved
	}
	return dto.SubtitleFixTyposResult{
		Format:      format,
		Content:     normalizedContent,
		ChangeCount: changes,
		Document:    subtitleDocumentWithSource(normalizedContent, format, document.Cues, document.Metadata),
	}, nil
}

func (service *LibraryService) SaveSubtitle(ctx context.Context, request dto.SubtitleSaveRequest) (dto.SubtitleSaveResult, error) {
	content, format, fileItem, documentItem, err := service.resolveSubtitleContent(ctx, request.FileID, request.DocumentID, request.Path, request.Content, request.Format)
	if err != nil {
		return dto.SubtitleSaveResult{}, err
	}
	if request.Document != nil {
		format = detectSubtitleFormat(request.TargetFormat, "", request.Document.Format)
		originalContent := firstNonEmpty(subtitleDocumentSourceContent(*request.Document), content)
		if preserved, ok := renderSubtitleContentPreservingSource(*request.Document, format, nil, "", originalContent); ok {
			content = preserved
		} else {
			content = renderSubtitleContent(*request.Document, format)
		}
	} else if strings.TrimSpace(request.TargetFormat) != "" {
		format = detectSubtitleFormat(request.TargetFormat, "", format)
	}
	if documentItem != nil {
		documentItem.Format = format
		documentItem.WorkingContent = content
		documentItem.UpdatedAt = service.now()
		if err := service.subtitles.Save(ctx, *documentItem); err != nil {
			return dto.SubtitleSaveResult{}, err
		}
	}
	if fileItem != nil {
		fileItem.UpdatedAt = service.now()
		if fileItem.Media == nil {
			fileItem.Media = libraryMediaInfo(format, int64(len(content)))
		} else {
			fileItem.Media.Format = format
			sizeValue := int64(len(content))
			fileItem.Media.SizeBytes = &sizeValue
		}
		if err := service.files.Save(ctx, *fileItem); err != nil {
			return dto.SubtitleSaveResult{}, err
		}
	}
	return dto.SubtitleSaveResult{Path: subtitleResultPath(fileItem), Format: format, Bytes: len(content)}, nil
}

func (service *LibraryService) RestoreSubtitleOriginal(ctx context.Context, request dto.RestoreSubtitleOriginalRequest) (dto.RestoreSubtitleOriginalResult, error) {
	_, _, fileItem, documentItem, err := service.resolveSubtitleContent(ctx, request.FileID, request.DocumentID, request.Path, "", "")
	if err != nil {
		return dto.RestoreSubtitleOriginalResult{}, err
	}
	if documentItem == nil {
		return dto.RestoreSubtitleOriginalResult{}, fmt.Errorf("subtitle document not found")
	}
	documentItem.WorkingContent = documentItem.OriginalContent
	documentItem.UpdatedAt = service.now()
	if err := service.subtitles.Save(ctx, *documentItem); err != nil {
		return dto.RestoreSubtitleOriginalResult{}, err
	}
	return dto.RestoreSubtitleOriginalResult{FileID: subtitleResultFileID(fileItem), Format: documentItem.Format, Bytes: len(documentItem.WorkingContent)}, nil
}

func (service *LibraryService) resolveSubtitleContent(ctx context.Context, fileID string, documentID string, path string, content string, format string) (string, string, *library.LibraryFile, *library.SubtitleDocument, error) {
	if trimmed := strings.TrimSpace(content); trimmed != "" {
		resolvedFormat := detectSubtitleFormat(format, path, "")
		return trimmed, resolvedFormat, nil, nil, nil
	}
	if strings.TrimSpace(documentID) != "" {
		documentItem, err := service.subtitles.Get(ctx, strings.TrimSpace(documentID))
		if err != nil {
			return "", "", nil, nil, err
		}
		fileItem, err := service.files.Get(ctx, documentItem.FileID)
		if err != nil {
			return "", "", nil, nil, err
		}
		resolvedFormat := detectSubtitleFormat(format, fileItem.Storage.LocalPath, documentItem.Format)
		text := strings.TrimSpace(documentItem.WorkingContent)
		if text == "" {
			text = documentItem.OriginalContent
		}
		return text, resolvedFormat, &fileItem, &documentItem, nil
	}
	if strings.TrimSpace(fileID) != "" {
		fileItem, err := service.files.Get(ctx, strings.TrimSpace(fileID))
		if err != nil {
			return "", "", nil, nil, err
		}
		documentItem, err := service.subtitles.GetByFileID(ctx, fileItem.ID)
		if err != nil {
			return "", "", nil, nil, err
		}
		resolvedFormat := detectSubtitleFormat(format, fileItem.Storage.LocalPath, documentItem.Format)
		text := strings.TrimSpace(documentItem.WorkingContent)
		if text == "" {
			text = documentItem.OriginalContent
		}
		return text, resolvedFormat, &fileItem, &documentItem, nil
	}
	if strings.TrimSpace(path) != "" {
		return "", "", nil, nil, fmt.Errorf("subtitle processing requires documentId or fileId when content is empty; path is only allowed for import")
	}
	return "", "", nil, nil, fmt.Errorf("documentId, fileId, or content is required")
}

func (service *LibraryService) resolveSubtitleValidationContent(
	ctx context.Context,
	fileID string,
	documentID string,
	path string,
	content string,
	format string,
	document *dto.SubtitleDocument,
) (string, string, error) {
	if document != nil {
		resolvedFormat := detectSubtitleFormat(format, path, document.Format)
		originalContent := subtitleDocumentSourceContent(*document)
		if preserved, ok := renderSubtitleContentPreservingSource(*document, resolvedFormat, nil, "", originalContent); ok {
			return preserved, resolvedFormat, nil
		}
		return renderSubtitleContent(*document, resolvedFormat), resolvedFormat, nil
	}
	resolvedContent, resolvedFormat, _, _, err := service.resolveSubtitleContent(ctx, fileID, documentID, path, content, format)
	if err != nil {
		return "", "", err
	}
	return resolvedContent, resolvedFormat, nil
}

func renderSubtitleContent(document dto.SubtitleDocument, targetFormat string) string {
	return renderSubtitleContentWithConfig(document, targetFormat, nil, "")
}

func detectSubtitleFormat(explicit string, path string, fallback string) string {
	if trimmed := normalizeSubtitleFormat(explicit); trimmed != "" {
		return trimmed
	}
	if trimmed := normalizeSubtitleFormat(normalizeFileExtension(path)); trimmed != "" {
		return trimmed
	}
	if trimmed := normalizeSubtitleFormat(fallback); trimmed != "" {
		return trimmed
	}
	return "srt"
}

func libraryMediaInfo(format string, size int64) *library.MediaInfo {
	sizeCopy := size
	return &library.MediaInfo{Format: format, SizeBytes: &sizeCopy}
}

func subtitleResultPath(fileItem *library.LibraryFile) string {
	if fileItem == nil {
		return ""
	}
	return fileItem.Storage.LocalPath
}

func subtitleResultFileID(fileItem *library.LibraryFile) string {
	if fileItem == nil {
		return ""
	}
	return fileItem.ID
}

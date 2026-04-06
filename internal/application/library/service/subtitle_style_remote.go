package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
	domainweb "dreamcreator/internal/domain/web"
)

const remoteSubtitleStyleHTTPTimeout = 20 * time.Second

type subtitleStyleRemoteManifest struct {
	Styles []subtitleStyleRemoteManifestItem `json:"styles"`
}

type subtitleStyleRemoteManifestItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	File        string   `json:"file"`
	Path        string   `json:"path"`
	ASSPath     string   `json:"assPath"`
	ContentPath string   `json:"contentPath"`
	Content     string   `json:"content"`
	Fonts       []string `json:"fonts"`
}

func (service *LibraryService) BrowseSubtitleStyleRemoteSource(
	ctx context.Context,
	request dto.BrowseSubtitleStyleRemoteSourceRequest,
) ([]dto.SubtitleStyleRemoteManifestItemDTO, error) {
	source, err := normalizeRemoteSubtitleStyleSource(request.Source, "style")
	if err != nil {
		return nil, err
	}

	manifest, err := loadRemoteSubtitleStyleManifest(ctx, source)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SubtitleStyleRemoteManifestItemDTO, 0, len(manifest.Styles))
	for _, item := range manifest.Styles {
		itemID := strings.TrimSpace(firstNonEmpty(item.ID, item.Name))
		if itemID == "" {
			continue
		}
		items = append(items, dto.SubtitleStyleRemoteManifestItemDTO{
			ID:          itemID,
			Name:        strings.TrimSpace(firstNonEmpty(item.Name, item.ID)),
			Description: strings.TrimSpace(item.Description),
			Version:     strings.TrimSpace(item.Version),
			FilePath:    strings.TrimSpace(resolveRemoteSubtitleStyleItemPath(item)),
			Fonts:       append([]string(nil), item.Fonts...),
		})
	}

	sort.Slice(items, func(left, right int) bool {
		if items[left].Name == items[right].Name {
			return items[left].ID < items[right].ID
		}
		return items[left].Name < items[right].Name
	})
	return items, nil
}

func (service *LibraryService) ImportSubtitleStyleRemoteSourceItem(
	ctx context.Context,
	request dto.ImportSubtitleStyleRemoteSourceItemRequest,
) (dto.LibrarySubtitleStyleDocumentDTO, error) {
	source, err := normalizeRemoteSubtitleStyleSource(request.Source, "style")
	if err != nil {
		return dto.LibrarySubtitleStyleDocumentDTO{}, err
	}
	itemID := strings.TrimSpace(request.ItemID)
	if itemID == "" {
		return dto.LibrarySubtitleStyleDocumentDTO{}, fmt.Errorf("remote style item id is required")
	}

	manifest, err := loadRemoteSubtitleStyleManifest(ctx, source)
	if err != nil {
		return dto.LibrarySubtitleStyleDocumentDTO{}, err
	}

	var selected *subtitleStyleRemoteManifestItem
	for index := range manifest.Styles {
		candidate := manifest.Styles[index]
		candidateID := strings.TrimSpace(firstNonEmpty(candidate.ID, candidate.Name))
		if candidateID == itemID {
			selected = &candidate
			break
		}
	}
	if selected == nil {
		return dto.LibrarySubtitleStyleDocumentDTO{}, fmt.Errorf("remote style item %q not found", itemID)
	}

	content := strings.TrimSpace(selected.Content)
	if content == "" {
		itemPath := resolveRemoteSubtitleStyleItemPath(*selected)
		if itemPath == "" {
			return dto.LibrarySubtitleStyleDocumentDTO{}, fmt.Errorf("remote style item %q is missing content path", itemID)
		}
		resolvedPath := resolveRemoteManifestRelativePath(source.ManifestPath, itemPath)
		rawURL := buildGitHubRawURL(source.Owner, source.Repo, source.Ref, resolvedPath)
		rawContent, err := fetchRemoteText(ctx, rawURL)
		if err != nil {
			return dto.LibrarySubtitleStyleDocumentDTO{}, err
		}
		content = strings.TrimSpace(rawContent)
	}
	if content == "" {
		return dto.LibrarySubtitleStyleDocumentDTO{}, fmt.Errorf("remote style item %q returned empty content", itemID)
	}

	analysis := library.AnalyzeSubtitleStyleDocument(content)
	return dto.LibrarySubtitleStyleDocumentDTO{
		ID:          "ass-" + uuid.NewString(),
		Name:        strings.TrimSpace(firstNonEmpty(selected.Name, selected.ID, itemID)),
		Description: strings.TrimSpace(selected.Description),
		Source:      "remote",
		SourceRef:   fmt.Sprintf("%s/%s@%s#%s", source.Owner, source.Repo, source.Ref, itemID),
		Version:     strings.TrimSpace(firstNonEmpty(selected.Version, "1")),
		Enabled:     true,
		Format:      "ass",
		Content:     content + "\n",
		Analysis: dto.LibrarySubtitleStyleDocumentAnalysisDTO{
			DetectedFormat:   analysis.DetectedFormat,
			ScriptType:       analysis.ScriptType,
			PlayResX:         analysis.PlayResX,
			PlayResY:         analysis.PlayResY,
			StyleCount:       analysis.StyleCount,
			DialogueCount:    analysis.DialogueCount,
			CommentCount:     analysis.CommentCount,
			StyleNames:       append([]string(nil), analysis.StyleNames...),
			Fonts:            append([]string(nil), analysis.Fonts...),
			FeatureFlags:     append([]string(nil), analysis.FeatureFlags...),
			ValidationIssues: append([]string(nil), analysis.ValidationIssues...),
		},
	}, nil
}

func normalizeRemoteSubtitleStyleSource(source dto.LibrarySubtitleStyleSourceDTO, expectedKind string) (library.SubtitleStyleSource, error) {
	normalized := toSubtitleStyleSources([]dto.LibrarySubtitleStyleSourceDTO{source})
	if len(normalized) == 0 {
		return library.SubtitleStyleSource{}, fmt.Errorf("remote source is invalid")
	}
	result := normalized[0]
	if expectedKind != "" && result.Kind != expectedKind {
		return library.SubtitleStyleSource{}, fmt.Errorf("remote source kind %q is not supported here", result.Kind)
	}
	if strings.TrimSpace(result.Provider) != "github" {
		return library.SubtitleStyleSource{}, fmt.Errorf("remote source provider %q is not supported", result.Provider)
	}
	if strings.TrimSpace(result.Owner) == "" || strings.TrimSpace(result.Repo) == "" {
		return library.SubtitleStyleSource{}, fmt.Errorf("remote source owner and repo are required")
	}
	return result, nil
}

func loadRemoteSubtitleStyleManifest(ctx context.Context, source library.SubtitleStyleSource) (subtitleStyleRemoteManifest, error) {
	rawURL := buildGitHubRawURL(source.Owner, source.Repo, source.Ref, source.ManifestPath)
	text, err := fetchRemoteText(ctx, rawURL)
	if err != nil {
		return subtitleStyleRemoteManifest{}, err
	}

	var manifest subtitleStyleRemoteManifest
	if err := json.Unmarshal([]byte(text), &manifest); err != nil {
		return subtitleStyleRemoteManifest{}, fmt.Errorf("parse remote style manifest: %w", err)
	}
	if manifest.Styles == nil {
		manifest.Styles = []subtitleStyleRemoteManifestItem{}
	}
	return manifest, nil
}

func resolveRemoteSubtitleStyleItemPath(item subtitleStyleRemoteManifestItem) string {
	return strings.TrimSpace(firstNonEmpty(item.ContentPath, item.ASSPath, item.File, item.Path))
}

func resolveRemoteManifestRelativePath(manifestPath string, assetPath string) string {
	trimmedAssetPath := strings.TrimSpace(assetPath)
	if trimmedAssetPath == "" {
		return ""
	}
	if strings.HasPrefix(trimmedAssetPath, "/") {
		return strings.TrimPrefix(trimmedAssetPath, "/")
	}
	baseDir := path.Dir(strings.TrimSpace(manifestPath))
	if baseDir == "." || baseDir == "/" {
		return trimmedAssetPath
	}
	return path.Clean(path.Join(baseDir, trimmedAssetPath))
}

func buildGitHubRawURL(owner string, repo string, ref string, filePath string) string {
	if parsed, err := url.Parse(strings.TrimSpace(filePath)); err == nil && parsed.Scheme != "" {
		return parsed.String()
	}
	joinedPath := path.Join(
		strings.TrimSpace(owner),
		strings.TrimSpace(repo),
		strings.TrimSpace(ref),
		strings.TrimSpace(strings.TrimPrefix(filePath, "/")),
	)
	return "https://raw.githubusercontent.com/" + joinedPath
}

func fetchRemoteText(ctx context.Context, rawURL string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", domainweb.DefaultBrowserRequestUserAgent)

	client := &http.Client{Timeout: remoteSubtitleStyleHTTPTimeout}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return "", fmt.Errorf("remote request failed: %s %s", response.Status, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

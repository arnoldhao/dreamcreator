package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"dreamcreator/internal/domain/library"
	domainweb "dreamcreator/internal/domain/web"
	"dreamcreator/internal/infrastructure/processutil"
)

const (
	remoteFontHTTPTimeout     = 30 * time.Second
	remoteFontSourceCacheTTL  = 24 * time.Hour
	maxRemoteManifestSize     = 64 << 20
	maxRemoteFontDownloadSize = 512 << 20
	defaultRemoteFontPriority = 100
)

type RemoteFontSource struct {
	ID       string
	Name     string
	Provider string
	URL      string
	Prefix   string
	Filename string
	Priority int
	BuiltIn  bool
}

type FontInstallTarget string

const (
	FontInstallTargetUser    FontInstallTarget = "user"
	FontInstallTargetMachine FontInstallTarget = "machine"
)

type InstalledRemoteFontFamily struct {
	Family         string
	InstalledFiles []string
	Target         FontInstallTarget
}

const (
	remoteFontAvailabilityInstallable        = "installable"
	remoteFontAvailabilityNoDownloadSource   = "no_download_source"
	remoteFontAvailabilityUnsupportedAsset   = "unsupported_asset"
	remoteFontAvailabilityUnknownUnavailable = "unavailable"
)

type RemoteFontSearchCandidate struct {
	SourceID           string
	SourceName         string
	FontID             string
	Family             string
	MatchName          string
	MatchType          string
	AssetCount         int
	DeclaredAssetCount int
	Installable        bool
	Availability       string
	UnavailableReason  string
}

type RemoteFontSourceSyncStatus struct {
	SourceID     string
	SourceName   string
	Manifest     library.SubtitleStyleSourceManifest
	FontCount    int
	SyncStatus   string
	LastSyncedAt string
	LastError    string
}

type resolvedRemoteFontCandidate struct {
	source             RemoteFontSource
	sourceName         string
	entry              remoteFontCatalogEntry
	matchName          string
	matchType          string
	score              int
	assetCount         int
	declaredAssetCount int
}

type remoteFontCatalog struct {
	sourceInfo remoteFontSourceInfo
	manifest   library.SubtitleStyleSourceManifest
	fonts      map[string]remoteFontCatalogEntry
}

type remoteFontSourceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	APIEndpoint string `json:"api_endpoint"`
	Version     string `json:"version"`
	LastUpdated string `json:"last_updated"`
	TotalFonts  int    `json:"total_fonts"`
}

type remoteFontCatalogEntry struct {
	ID                 string
	Name               string
	Family             string
	Aliases            []string
	Description        string
	Popularity         int
	Assets             []remoteFontDownloadAsset
	DeclaredAssetCount int
	UnavailableReason  string
}

type remoteFontDownloadAsset struct {
	FileName string
	URL      string
}

type remoteFontCatalogEnvelope struct {
	SourceInfo json.RawMessage `json:"source_info"`
	Fonts      json.RawMessage `json:"fonts"`
}

type remoteFontSourceFile struct {
	Name          string                       `json:"name"`
	Family        string                       `json:"family"`
	ID            string                       `json:"id,omitempty"`
	License       string                       `json:"license,omitempty"`
	LicenseURL    string                       `json:"license_url,omitempty"`
	Designer      string                       `json:"designer,omitempty"`
	Foundry       string                       `json:"foundry,omitempty"`
	Version       string                       `json:"version,omitempty"`
	Description   string                       `json:"description,omitempty"`
	Categories    []string                     `json:"categories,omitempty"`
	Tags          []string                     `json:"tags,omitempty"`
	Popularity    int                          `json:"popularity,omitempty"`
	LastModified  string                       `json:"last_modified,omitempty"`
	MetadataURL   string                       `json:"metadata_url,omitempty"`
	SourceURL     string                       `json:"source_url,omitempty"`
	DownloadURL   string                       `json:"download_url,omitempty"`
	Files         map[string]string            `json:"files,omitempty"`
	Variants      []remoteFontSourceVariant    `json:"variants,omitempty"`
	Aliases       []string                     `json:"aliases,omitempty"`
	VariantFiles  map[string]map[string]string `json:"variant_files,omitempty"`
	UnicodeRanges []string                     `json:"unicode_ranges,omitempty"`
	Languages     []string                     `json:"languages,omitempty"`
	SampleText    string                       `json:"sample_text,omitempty"`
}

type remoteFontSourceVariant struct {
	Name    string            `json:"name"`
	Files   map[string]string `json:"files"`
	Weight  int               `json:"weight,omitempty"`
	Style   string            `json:"style,omitempty"`
	Subsets []string          `json:"subsets,omitempty"`
}

type cachedRemoteFontCatalog struct {
	catalog  remoteFontCatalog
	loadedAt time.Time
}

var (
	remoteFontCatalogCacheMu sync.Mutex
	remoteFontCatalogCache   = map[string]cachedRemoteFontCatalog{}
)

func (service *FontService) SearchRemoteFontFamily(
	ctx context.Context,
	family string,
	sources []RemoteFontSource,
) ([]RemoteFontSearchCandidate, error) {
	candidates, err := searchRemoteFontCandidates(ctx, family, sources)
	if err != nil {
		return nil, err
	}
	result := make([]RemoteFontSearchCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, RemoteFontSearchCandidate{
			SourceID:           candidate.source.ID,
			SourceName:         candidate.sourceName,
			FontID:             candidate.entry.ID,
			Family:             strings.TrimSpace(candidate.entry.Family),
			MatchName:          candidate.matchName,
			MatchType:          candidate.matchType,
			AssetCount:         candidate.assetCount,
			DeclaredAssetCount: candidate.declaredAssetCount,
			Installable:        candidate.assetCount > 0,
			Availability:       resolveRemoteFontCatalogEntryAvailability(candidate.entry),
			UnavailableReason:  candidate.entry.UnavailableReason,
		})
	}
	if result == nil {
		return []RemoteFontSearchCandidate{}, nil
	}
	return result, nil
}

func (service *FontService) InstallRemoteFontFamily(
	ctx context.Context,
	family string,
	sources []RemoteFontSource,
	target FontInstallTarget,
	preferredSourceID string,
) (InstalledRemoteFontFamily, error) {
	trimmedFamily := strings.TrimSpace(family)
	if trimmedFamily == "" {
		return InstalledRemoteFontFamily{}, fmt.Errorf("font family is required")
	}
	installTarget := normalizeFontInstallTarget(target)
	if installTarget == "" {
		installTarget = FontInstallTargetUser
	}

	if err := service.ensureCatalog(ctx); err == nil {
		service.mu.RLock()
		entries := service.catalog.filesByFamily[normalizeFontFamilyKey(trimmedFamily)]
		service.mu.RUnlock()
		if len(entries) > 0 {
			return InstalledRemoteFontFamily{
				Family:         entries[0].family,
				InstalledFiles: []string{},
				Target:         installTarget,
			}, nil
		}
	}

	candidates, err := searchRemoteFontCandidates(ctx, trimmedFamily, sources)
	if err != nil {
		return InstalledRemoteFontFamily{}, err
	}
	if len(candidates) == 0 {
		return InstalledRemoteFontFamily{}, fmt.Errorf("font %q was not found in any configured font source", trimmedFamily)
	}

	trimmedPreferredSourceID := strings.TrimSpace(preferredSourceID)
	var lastErr error
	var unavailableCandidate *resolvedRemoteFontCandidate
	for _, candidate := range candidates {
		if trimmedPreferredSourceID != "" && candidate.source.ID != trimmedPreferredSourceID {
			continue
		}
		if candidate.assetCount <= 0 {
			if unavailableCandidate == nil {
				copied := candidate
				unavailableCandidate = &copied
			}
			continue
		}
		result, installErr := service.installRemoteFontCatalogEntry(ctx, installTarget, candidate.entry, trimmedFamily)
		if installErr == nil {
			return result, nil
		}
		lastErr = installErr
	}
	if lastErr != nil {
		return InstalledRemoteFontFamily{}, lastErr
	}
	if unavailableCandidate != nil {
		return InstalledRemoteFontFamily{}, buildRemoteFontUnavailableError(trimmedFamily, *unavailableCandidate)
	}
	if trimmedPreferredSourceID != "" {
		return InstalledRemoteFontFamily{}, fmt.Errorf("font %q was not found in the selected font source", trimmedFamily)
	}
	return InstalledRemoteFontFamily{}, fmt.Errorf("font %q was not found in any configured font source", trimmedFamily)
}

func (service *FontService) SyncRemoteFontSource(
	ctx context.Context,
	source RemoteFontSource,
) (RemoteFontSourceSyncStatus, error) {
	normalized, ok := normalizeRemoteFontSource(source, 0)
	if !ok {
		return RemoteFontSourceSyncStatus{
			SourceID:   strings.TrimSpace(source.ID),
			SourceName: strings.TrimSpace(firstNonEmpty(source.Name, source.Prefix, source.ID)),
			SyncStatus: "error",
			LastError:  "font source is invalid",
		}, fmt.Errorf("font source is invalid")
	}

	status := RemoteFontSourceSyncStatus{
		SourceID:   normalized.ID,
		SourceName: normalized.Name,
		SyncStatus: "error",
	}

	catalog, err := loadRemoteFontCatalog(ctx, normalized, true)
	if err != nil {
		status.LastError = err.Error()
		return status, err
	}

	status.Manifest = catalog.manifest
	fontCount := catalog.manifest.SourceInfo.TotalFonts
	if fontCount <= 0 {
		fontCount = catalog.sourceInfo.TotalFonts
	}
	if fontCount <= 0 {
		fontCount = len(catalog.manifest.Fonts)
	}
	if fontCount <= 0 {
		fontCount = len(catalog.fonts)
	}
	status.SourceName = strings.TrimSpace(firstNonEmpty(catalog.manifest.SourceInfo.Name, catalog.sourceInfo.Name, normalized.Name))
	status.FontCount = fontCount
	status.SyncStatus = "synced"
	status.LastSyncedAt = time.Now().UTC().Format(time.RFC3339)
	status.LastError = ""
	return status, nil
}

func (service *FontService) installRemoteFontCatalogEntry(
	ctx context.Context,
	target FontInstallTarget,
	entry remoteFontCatalogEntry,
	requestedFamily string,
) (InstalledRemoteFontFamily, error) {
	if len(entry.Assets) == 0 {
		return InstalledRemoteFontFamily{}, fmt.Errorf("font %q does not expose downloadable font files", requestedFamily)
	}

	installTarget := normalizeFontInstallTarget(target)
	installDir, err := installDirectoryForTarget(installTarget)
	if err != nil {
		return InstalledRemoteFontFamily{}, err
	}
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return InstalledRemoteFontFamily{}, err
	}

	tempDir, err := os.MkdirTemp("", "dreamcreator-font-install-*")
	if err != nil {
		return InstalledRemoteFontFamily{}, err
	}
	defer os.RemoveAll(tempDir)

	installedFiles := make([]string, 0, len(entry.Assets))
	seenInstalled := make(map[string]struct{}, len(entry.Assets)*2)
	for _, asset := range entry.Assets {
		if ctx.Err() != nil {
			return InstalledRemoteFontFamily{}, ctx.Err()
		}

		downloadedPath, err := downloadRemoteFontAsset(ctx, tempDir, asset)
		if err != nil {
			return InstalledRemoteFontFamily{}, err
		}

		fontFiles, err := resolveDownloadedFontFiles(downloadedPath, asset.FileName, tempDir)
		if err != nil {
			return InstalledRemoteFontFamily{}, err
		}
		for _, fontFile := range fontFiles {
			data, err := os.ReadFile(fontFile)
			if err != nil {
				return InstalledRemoteFontFamily{}, err
			}
			if size := int64(len(data)); size <= 0 || size > maxFontFileSizeBytes {
				return InstalledRemoteFontFamily{}, fmt.Errorf("remote font asset %q has invalid size", filepath.Base(fontFile))
			}
			targetPath := filepath.Join(installDir, filepath.Base(fontFile))
			if _, exists := seenInstalled[targetPath]; exists {
				continue
			}
			if err := writeAtomicFile(targetPath, data, 0o644); err != nil {
				return InstalledRemoteFontFamily{}, err
			}
			seenInstalled[targetPath] = struct{}{}
			installedFiles = append(installedFiles, targetPath)
		}
	}

	if len(installedFiles) == 0 {
		return InstalledRemoteFontFamily{}, fmt.Errorf("font %q did not provide installable font files", requestedFamily)
	}

	if err := refreshInstalledFontCache(installTarget, installDir); err != nil {
		return InstalledRemoteFontFamily{}, err
	}
	if err := service.RefreshCatalog(ctx); err != nil {
		return InstalledRemoteFontFamily{}, err
	}

	return InstalledRemoteFontFamily{
		Family:         strings.TrimSpace(firstNonEmpty(entry.Family, entry.Name, requestedFamily)),
		InstalledFiles: installedFiles,
		Target:         installTarget,
	}, nil
}

func searchRemoteFontCandidates(
	ctx context.Context,
	family string,
	sources []RemoteFontSource,
) ([]resolvedRemoteFontCandidate, error) {
	trimmedFamily := strings.TrimSpace(family)
	if trimmedFamily == "" {
		return nil, fmt.Errorf("font family is required")
	}

	normalizedSources := normalizeRemoteFontSources(sources)
	if len(normalizedSources) == 0 {
		return nil, fmt.Errorf("no enabled font source is configured")
	}

	var (
		candidates []resolvedRemoteFontCandidate
		lastErr    error
		loadedAny  bool
	)

	for _, source := range normalizedSources {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		catalog, err := loadRemoteFontCatalog(ctx, source, false)
		if err != nil {
			lastErr = err
			continue
		}
		loadedAny = true
		sourceName := strings.TrimSpace(firstNonEmpty(catalog.sourceInfo.Name, source.Name))
		candidates = append(candidates, findRemoteFontCatalogMatches(source, sourceName, catalog, trimmedFamily)...)
	}

	if !loadedAny && lastErr != nil {
		return nil, lastErr
	}

	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].score != candidates[right].score {
			return candidates[left].score > candidates[right].score
		}
		if (candidates[left].assetCount > 0) != (candidates[right].assetCount > 0) {
			return candidates[left].assetCount > 0
		}
		if candidates[left].source.Priority != candidates[right].source.Priority {
			return candidates[left].source.Priority < candidates[right].source.Priority
		}
		if candidates[left].entry.Popularity != candidates[right].entry.Popularity {
			return candidates[left].entry.Popularity > candidates[right].entry.Popularity
		}
		if candidates[left].sourceName != candidates[right].sourceName {
			return candidates[left].sourceName < candidates[right].sourceName
		}
		return strings.TrimSpace(candidates[left].entry.Family) < strings.TrimSpace(candidates[right].entry.Family)
	})

	if candidates == nil {
		return []resolvedRemoteFontCandidate{}, nil
	}
	return candidates, nil
}

func normalizeRemoteFontSources(sources []RemoteFontSource) []RemoteFontSource {
	result := make([]RemoteFontSource, 0, len(sources))
	for index, source := range sources {
		normalized, ok := normalizeRemoteFontSource(source, index)
		if !ok {
			continue
		}
		result = append(result, normalized)
	}

	sort.SliceStable(result, func(left, right int) bool {
		if result[left].BuiltIn != result[right].BuiltIn {
			return result[left].BuiltIn
		}
		if result[left].Priority != result[right].Priority {
			return result[left].Priority < result[right].Priority
		}
		return result[left].Name < result[right].Name
	})
	return result
}

func normalizeRemoteFontSource(source RemoteFontSource, index int) (RemoteFontSource, bool) {
	provider := strings.ToLower(strings.TrimSpace(source.Provider))
	if provider == "" {
		provider = "fontget"
	}

	if provider != "fontget" {
		return RemoteFontSource{}, false
	}

	name := strings.TrimSpace(firstNonEmpty(source.Name, source.Prefix, source.ID))
	if name == "" {
		name = fmt.Sprintf("Font source %d", index+1)
	}

	prefix := normalizeRemoteSourcePrefix(source.Prefix, name)
	urlValue := strings.TrimSpace(source.URL)
	filename := strings.TrimSpace(source.Filename)
	if urlValue == "" {
		return RemoteFontSource{}, false
	}
	if filename == "" {
		filename = path.Base(strings.TrimSpace(urlValue))
		if filename == "" || filename == "." || filename == "/" {
			filename = "fonts.json"
		}
	}
	if prefix == "" {
		prefix = normalizeRemoteSourcePrefix(path.Base(strings.TrimSuffix(filename, filepath.Ext(filename))), name)
	}
	priority := source.Priority
	if priority <= 0 {
		priority = defaultRemoteFontPriority + index + 1
	}

	id := strings.TrimSpace(source.ID)
	if id == "" {
		switch {
		case prefix != "":
			id = prefix
		default:
			id = normalizeRemoteSourcePrefix("", name)
		}
	}

	return RemoteFontSource{
		ID:       id,
		Name:     name,
		Provider: provider,
		URL:      urlValue,
		Prefix:   prefix,
		Filename: filename,
		Priority: priority,
		BuiltIn:  source.BuiltIn,
	}, true
}

func loadRemoteFontCatalog(ctx context.Context, source RemoteFontSource, forceRefresh bool) (remoteFontCatalog, error) {
	cacheKey := remoteFontSourceCacheKey(source)
	now := time.Now()

	if !forceRefresh {
		remoteFontCatalogCacheMu.Lock()
		if entry, exists := remoteFontCatalogCache[cacheKey]; exists && now.Sub(entry.loadedAt) < remoteFontSourceCacheTTL {
			remoteFontCatalogCacheMu.Unlock()
			return entry.catalog, nil
		}
		remoteFontCatalogCacheMu.Unlock()
	}

	manifestText, err := fetchRemoteBinary(ctx, source.URL, maxRemoteManifestSize)
	if err != nil {
		return remoteFontCatalog{}, err
	}
	catalog, err := parseRemoteFontCatalog(manifestText, source)
	if err != nil {
		return remoteFontCatalog{}, err
	}

	remoteFontCatalogCacheMu.Lock()
	remoteFontCatalogCache[cacheKey] = cachedRemoteFontCatalog{
		catalog:  catalog,
		loadedAt: now,
	}
	remoteFontCatalogCacheMu.Unlock()

	return catalog, nil
}

func remoteFontSourceCacheKey(source RemoteFontSource) string {
	return strings.Join([]string{
		strings.TrimSpace(source.ID),
		strings.TrimSpace(source.Provider),
		strings.TrimSpace(source.URL),
		strings.TrimSpace(source.Prefix),
	}, "\x00")
}

func parseRemoteFontCatalog(data []byte, source RemoteFontSource) (remoteFontCatalog, error) {
	var envelope remoteFontCatalogEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return remoteFontCatalog{}, fmt.Errorf("parse remote font manifest: %w", err)
	}
	if len(envelope.Fonts) == 0 {
		return remoteFontCatalog{}, fmt.Errorf("remote font source %q is missing fonts data", source.Name)
	}

	var sourceInfo remoteFontSourceInfo
	_ = json.Unmarshal(envelope.SourceInfo, &sourceInfo)

	var fontGetFonts map[string]remoteFontSourceFile
	if err := json.Unmarshal(envelope.Fonts, &fontGetFonts); err != nil || len(fontGetFonts) == 0 {
		return remoteFontCatalog{}, fmt.Errorf("parse remote font manifest: expected FontGet source schema")
	}
	result := remoteFontCatalog{
		sourceInfo: sourceInfo,
		manifest: library.SubtitleStyleSourceManifest{
			SourceInfo: library.SubtitleStyleSourceManifestInfo{
				Name:        strings.TrimSpace(sourceInfo.Name),
				Description: strings.TrimSpace(sourceInfo.Description),
				URL:         strings.TrimSpace(sourceInfo.URL),
				APIEndpoint: strings.TrimSpace(sourceInfo.APIEndpoint),
				Version:     strings.TrimSpace(sourceInfo.Version),
				LastUpdated: strings.TrimSpace(sourceInfo.LastUpdated),
				TotalFonts:  sourceInfo.TotalFonts,
			},
			Fonts: make(map[string]library.SubtitleStyleSourceManifestFont, len(fontGetFonts)),
		},
		fonts: make(map[string]remoteFontCatalogEntry, len(fontGetFonts)),
	}
	for rawID, font := range fontGetFonts {
		manifestID := strings.TrimSpace(firstNonEmpty(font.ID, rawID))
		if manifestID != "" {
			result.manifest.Fonts[manifestID] = convertFontGetManifestFont(font)
		}
		entry := convertFontGetSourceFont(rawID, font)
		if entry.ID == "" {
			continue
		}
		result.fonts[entry.ID] = entry
	}
	if result.sourceInfo.Name == "" {
		result.sourceInfo.Name = strings.TrimSpace(source.Name)
	}
	if result.manifest.SourceInfo.Name == "" {
		result.manifest.SourceInfo.Name = strings.TrimSpace(firstNonEmpty(result.sourceInfo.Name, source.Name))
	}
	if result.manifest.SourceInfo.TotalFonts <= 0 {
		result.manifest.SourceInfo.TotalFonts = len(result.manifest.Fonts)
	}
	if result.sourceInfo.TotalFonts <= 0 {
		result.sourceInfo.TotalFonts = result.manifest.SourceInfo.TotalFonts
	}
	return result, nil
}

func convertFontGetSourceFont(rawID string, font remoteFontSourceFile) remoteFontCatalogEntry {
	entry := remoteFontCatalogEntry{
		ID:          strings.TrimSpace(firstNonEmpty(font.ID, rawID)),
		Name:        strings.TrimSpace(firstNonEmpty(font.Name, font.Family, rawID)),
		Family:      strings.TrimSpace(firstNonEmpty(font.Family, font.Name, rawID)),
		Aliases:     append([]string(nil), font.Aliases...),
		Description: strings.TrimSpace(font.Description),
		Popularity:  font.Popularity,
		Assets:      []remoteFontDownloadAsset{},
	}
	seen := make(map[string]struct{})
	for fileType, rawURL := range font.Files {
		if strings.TrimSpace(rawURL) == "" {
			continue
		}
		entry.DeclaredAssetCount++
		entry.Assets = appendRemoteFontAsset(entry.Assets, seen, buildRemoteFontAssetFileName(entry.Family, "", rawURL, fileType), rawURL, fileType)
	}
	for _, variant := range font.Variants {
		for fileType, rawURL := range variant.Files {
			if strings.TrimSpace(rawURL) == "" {
				continue
			}
			entry.DeclaredAssetCount++
			entry.Assets = appendRemoteFontAsset(entry.Assets, seen, buildRemoteFontAssetFileName(entry.Family, variant.Name, rawURL, fileType), rawURL, fileType)
		}
	}
	for variantName, fileTypes := range font.VariantFiles {
		for fileType, rawURL := range fileTypes {
			if strings.TrimSpace(rawURL) == "" {
				continue
			}
			entry.DeclaredAssetCount++
			entry.Assets = appendRemoteFontAsset(entry.Assets, seen, buildRemoteFontAssetFileName(entry.Family, variantName, rawURL, fileType), rawURL, fileType)
		}
	}
	if strings.TrimSpace(font.DownloadURL) != "" {
		entry.DeclaredAssetCount++
		entry.Assets = appendRemoteFontAsset(entry.Assets, seen, buildRemoteFontAssetFileName(entry.Family, "", font.DownloadURL, ""), font.DownloadURL, "")
	}
	switch {
	case entry.DeclaredAssetCount == 0:
		entry.UnavailableReason = remoteFontAvailabilityNoDownloadSource
	case len(entry.Assets) == 0:
		entry.UnavailableReason = remoteFontAvailabilityUnsupportedAsset
	}
	sort.Slice(entry.Assets, func(left, right int) bool {
		if entry.Assets[left].FileName == entry.Assets[right].FileName {
			return entry.Assets[left].URL < entry.Assets[right].URL
		}
		return entry.Assets[left].FileName < entry.Assets[right].FileName
	})
	return entry
}

func convertFontGetManifestFont(font remoteFontSourceFile) library.SubtitleStyleSourceManifestFont {
	variants := make([]library.SubtitleStyleSourceManifestVariant, 0, len(font.Variants))
	for _, variant := range font.Variants {
		files := make(map[string]string, len(variant.Files))
		for fileType, rawURL := range variant.Files {
			if strings.TrimSpace(fileType) == "" || strings.TrimSpace(rawURL) == "" {
				continue
			}
			files[strings.TrimSpace(fileType)] = strings.TrimSpace(rawURL)
		}
		variants = append(variants, library.SubtitleStyleSourceManifestVariant{
			Name:    strings.TrimSpace(variant.Name),
			Weight:  variant.Weight,
			Style:   strings.TrimSpace(variant.Style),
			Subsets: append([]string(nil), variant.Subsets...),
			Files:   files,
		})
	}
	return library.SubtitleStyleSourceManifestFont{
		Name:          strings.TrimSpace(font.Name),
		Family:        strings.TrimSpace(font.Family),
		License:       strings.TrimSpace(font.License),
		LicenseURL:    strings.TrimSpace(font.LicenseURL),
		Designer:      strings.TrimSpace(font.Designer),
		Foundry:       strings.TrimSpace(font.Foundry),
		Version:       strings.TrimSpace(font.Version),
		Description:   strings.TrimSpace(font.Description),
		Categories:    append([]string(nil), font.Categories...),
		Tags:          append([]string(nil), font.Tags...),
		Popularity:    font.Popularity,
		LastModified:  strings.TrimSpace(font.LastModified),
		MetadataURL:   strings.TrimSpace(font.MetadataURL),
		SourceURL:     strings.TrimSpace(font.SourceURL),
		Variants:      variants,
		UnicodeRanges: append([]string(nil), font.UnicodeRanges...),
		Languages:     append([]string(nil), font.Languages...),
		SampleText:    strings.TrimSpace(font.SampleText),
	}
}

func findRemoteFontCatalogMatches(
	source RemoteFontSource,
	sourceName string,
	catalog remoteFontCatalog,
	query string,
) []resolvedRemoteFontCandidate {
	normalizedQuery := normalizeFontFamilyKey(query)
	if normalizedQuery == "" {
		return nil
	}

	candidates := make([]resolvedRemoteFontCandidate, 0, len(catalog.fonts))
	for _, entry := range catalog.fonts {
		matchName, matchType, score := scoreRemoteFontCatalogEntry(normalizedQuery, entry)
		if score <= 0 {
			continue
		}
		candidates = append(candidates, resolvedRemoteFontCandidate{
			source:             source,
			sourceName:         sourceName,
			entry:              entry,
			matchName:          matchName,
			matchType:          matchType,
			score:              score,
			assetCount:         len(entry.Assets),
			declaredAssetCount: entry.DeclaredAssetCount,
		})
	}
	return candidates
}

func resolveRemoteFontCatalogEntryAvailability(entry remoteFontCatalogEntry) string {
	if len(entry.Assets) > 0 {
		return remoteFontAvailabilityInstallable
	}
	if strings.TrimSpace(entry.UnavailableReason) != "" {
		return entry.UnavailableReason
	}
	return remoteFontAvailabilityUnknownUnavailable
}

func buildRemoteFontUnavailableError(requestedFamily string, candidate resolvedRemoteFontCandidate) error {
	switch resolveRemoteFontCatalogEntryAvailability(candidate.entry) {
	case remoteFontAvailabilityNoDownloadSource:
		return fmt.Errorf("font %q was found in %s, but the source does not provide downloadable font files", requestedFamily, candidate.sourceName)
	case remoteFontAvailabilityUnsupportedAsset:
		return fmt.Errorf("font %q was found in %s, but the download format is not supported yet", requestedFamily, candidate.sourceName)
	default:
		return fmt.Errorf("font %q was found in %s, but it cannot be installed right now", requestedFamily, candidate.sourceName)
	}
}

func scoreRemoteFontCatalogEntry(query string, entry remoteFontCatalogEntry) (string, string, int) {
	scoreSingle := func(value string, exactScore int, prefixScore int, containsScore int, matchType string) (string, string, int) {
		trimmed := strings.TrimSpace(value)
		normalized := normalizeFontFamilyKey(trimmed)
		switch {
		case normalized != "" && normalized == query:
			return trimmed, "exact_" + matchType, exactScore
		case normalized != "" && strings.HasPrefix(normalized, query):
			return trimmed, "prefix_" + matchType, prefixScore
		case normalized != "" && strings.Contains(normalized, query):
			return trimmed, "contains_" + matchType, containsScore
		default:
			return "", "", 0
		}
	}

	bestName, bestType, bestScore := scoreSingle(entry.Family, 100, 80, 60, "family")
	if bestScore == 0 {
		bestName, bestType, bestScore = scoreSingle(entry.Name, 98, 78, 58, "name")
	}
	if bestScore == 0 {
		bestName, bestType, bestScore = scoreSingle(entry.ID, 95, 45, 30, "id")
	}
	for _, alias := range entry.Aliases {
		matchName, matchType, score := scoreSingle(alias, 92, 72, 52, "alias")
		if score > bestScore {
			bestName = matchName
			bestType = matchType
			bestScore = score
		}
	}
	return bestName, bestType, bestScore
}

func appendRemoteFontAsset(
	assets []remoteFontDownloadAsset,
	seen map[string]struct{},
	fileName string,
	rawURL string,
	fileType string,
) []remoteFontDownloadAsset {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return assets
	}
	trimmedFileName := strings.TrimSpace(fileName)
	if trimmedFileName == "" {
		trimmedFileName = path.Base(trimmedURL)
	}
	if trimmedFileName == "" {
		return assets
	}
	if !isFontAsset(trimmedFileName, fileType) {
		return assets
	}
	key := trimmedFileName + "\x00" + trimmedURL
	if _, exists := seen[key]; exists {
		return assets
	}
	seen[key] = struct{}{}
	return append(assets, remoteFontDownloadAsset{
		FileName: trimmedFileName,
		URL:      trimmedURL,
	})
}

func buildRemoteFontAssetFileName(family string, variantName string, rawURL string, fileType string) string {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return ""
	}
	if fileName := strings.TrimSpace(path.Base(trimmedURL)); fileName != "" && fileName != "." && fileName != "/" {
		if isFontFile(fileName) || strings.HasSuffix(strings.ToLower(fileName), ".zip") || strings.HasSuffix(strings.ToLower(fileName), ".tar.xz") {
			return fileName
		}
	}

	ext := inferRemoteFontAssetExtension(trimmedURL, fileType)
	if ext == "" {
		return strings.TrimSpace(path.Base(trimmedURL))
	}

	cleanName := sanitizeRemoteFontAssetName(firstNonEmpty(family, variantName, "remote-font"))
	cleanVariant := sanitizeRemoteFontAssetName(variantName)
	if cleanVariant != "" && cleanName != "" && strings.HasPrefix(strings.ToLower(cleanVariant), strings.ToLower(cleanName)) {
		cleanVariant = strings.TrimSpace(cleanVariant[len(cleanName):])
	}
	if cleanVariant != "" && !strings.EqualFold(cleanVariant, "regular") {
		return cleanName + "-" + cleanVariant + ext
	}
	if cleanName != "" {
		return cleanName + ext
	}
	return "remote-font" + ext
}

func sanitizeRemoteFontAssetName(value string) string {
	cleaned := strings.TrimSpace(value)
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "_", "")
	return cleaned
}

func inferRemoteFontAssetExtension(rawURL string, fileType string) string {
	trimmedURL := strings.TrimSpace(rawURL)
	loweredURL := strings.ToLower(trimmedURL)
	switch {
	case strings.HasSuffix(loweredURL, ".tar.xz"):
		return ".tar.xz"
	case isFontFile(trimmedURL):
		return strings.ToLower(filepath.Ext(trimmedURL))
	case strings.HasSuffix(loweredURL, ".zip"):
		return ".zip"
	}

	switch strings.ToLower(strings.TrimSpace(fileType)) {
	case "ttf", "otf", "ttc", "otc", "zip":
		return "." + strings.ToLower(strings.TrimSpace(fileType))
	case "tar.xz":
		return ".tar.xz"
	default:
		return ""
	}
}

func isFontAsset(fileName string, fileType string) bool {
	trimmedFileType := strings.ToLower(strings.TrimSpace(fileType))
	switch trimmedFileType {
	case "", "ttf", "otf", "ttc", "otc", "zip", "tar.xz":
	default:
		return false
	}
	normalizedName := strings.ToLower(strings.TrimSpace(fileName))
	switch {
	case isFontFile(normalizedName):
		return true
	case strings.HasSuffix(normalizedName, ".zip"):
		return true
	case strings.HasSuffix(normalizedName, ".tar.xz"):
		return true
	default:
		return false
	}
}

func fetchRemoteBinary(ctx context.Context, rawURL string, maxBytes int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", domainweb.DefaultBrowserRequestUserAgent)

	client := &http.Client{Timeout: remoteFontHTTPTimeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return nil, fmt.Errorf("remote request failed: %s %s", response.Status, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("remote asset exceeded size limit")
	}
	return body, nil
}

func downloadRemoteFontAsset(ctx context.Context, tempDir string, asset remoteFontDownloadAsset) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", domainweb.DefaultBrowserRequestUserAgent)

	client := &http.Client{Timeout: remoteFontHTTPTimeout}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if wafAction := strings.TrimSpace(response.Header.Get("x-amzn-waf-action")); wafAction != "" {
		return "", fmt.Errorf("remote font source blocked the download request (%s challenge)", wafAction)
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return "", fmt.Errorf("remote font download failed: %s %s", response.Status, strings.TrimSpace(string(body)))
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return "", fmt.Errorf("remote request failed: %s %s", response.Status, strings.TrimSpace(string(body)))
	}

	tempFile, err := os.CreateTemp(tempDir, "font-asset-*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	written, err := io.Copy(tempFile, io.LimitReader(response.Body, maxRemoteFontDownloadSize+1))
	if err != nil {
		return "", err
	}
	if written <= 0 || written > maxRemoteFontDownloadSize {
		return "", fmt.Errorf("remote font asset %q has invalid size", asset.FileName)
	}
	return tempFile.Name(), nil
}

func resolveDownloadedFontFiles(downloadedPath string, originalFileName string, tempDir string) ([]string, error) {
	if detectedArchiveType, ok := detectDownloadedArchiveType(downloadedPath); ok {
		switch detectedArchiveType {
		case ".zip":
			return extractZipFontFiles(downloadedPath, tempDir)
		case ".tar.xz":
			return extractTarXZFontFiles(downloadedPath, tempDir)
		}
	}

	normalizedName := strings.ToLower(strings.TrimSpace(originalFileName))
	switch {
	case isFontFile(normalizedName):
		return []string{downloadedPath}, nil
	case strings.HasSuffix(normalizedName, ".zip"):
		return extractZipFontFiles(downloadedPath, tempDir)
	case strings.HasSuffix(normalizedName, ".tar.xz"):
		return extractTarXZFontFiles(downloadedPath, tempDir)
	default:
		return nil, fmt.Errorf("unsupported remote font asset %q", originalFileName)
	}
}

func detectDownloadedArchiveType(downloadedPath string) (string, bool) {
	file, err := os.Open(downloadedPath)
	if err != nil {
		return "", false
	}
	defer file.Close()

	header := make([]byte, 8)
	n, err := io.ReadFull(file, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", false
	}
	header = header[:n]
	switch {
	case len(header) >= 4 && string(header[:4]) == "PK\x03\x04":
		return ".zip", true
	case len(header) >= 6 && string(header[:6]) == "\xFD7zXZ\x00":
		return ".tar.xz", true
	default:
		return "", false
	}
}

func extractZipFontFiles(archivePath string, tempDir string) ([]string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	extractDir := filepath.Join(tempDir, "zip-extracted-"+filepath.Base(archivePath))
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return nil, err
	}

	extractedFiles := make([]string, 0)
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || !isFontFile(file.Name) {
			continue
		}

		sourceFile, err := file.Open()
		if err != nil {
			return nil, err
		}

		targetPath := filepath.Join(extractDir, filepath.Base(file.Name))
		targetFile, err := os.Create(targetPath)
		if err != nil {
			_ = sourceFile.Close()
			return nil, err
		}

		if _, err := io.Copy(targetFile, sourceFile); err != nil {
			_ = targetFile.Close()
			_ = sourceFile.Close()
			return nil, err
		}
		_ = targetFile.Close()
		_ = sourceFile.Close()
		extractedFiles = append(extractedFiles, targetPath)
	}

	if len(extractedFiles) == 0 {
		return nil, fmt.Errorf("no font files found in archive")
	}
	return extractedFiles, nil
}

func extractTarXZFontFiles(archivePath string, tempDir string) ([]string, error) {
	tarPath, err := exec.LookPath("tar")
	if err != nil {
		return nil, fmt.Errorf("tar.xz archives are not supported on this system")
	}

	extractDir := filepath.Join(tempDir, "tarxz-extracted-"+filepath.Base(archivePath))
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return nil, err
	}

	command := exec.Command(tarPath, "-xJf", archivePath, "-C", extractDir)
	processutil.ConfigureCLI(command)
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("extract tar.xz archive: %w %s", err, strings.TrimSpace(string(output)))
	}

	extractedFiles := make([]string, 0, 8)
	err = filepath.WalkDir(extractDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isFontFile(path) {
			return nil
		}
		extractedFiles = append(extractedFiles, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(extractedFiles) == 0 {
		return nil, fmt.Errorf("no font files found in archive")
	}
	return extractedFiles, nil
}

func normalizeFontInstallTarget(value FontInstallTarget) FontInstallTarget {
	switch strings.ToLower(strings.TrimSpace(string(value))) {
	case "", string(FontInstallTargetUser):
		return FontInstallTargetUser
	case string(FontInstallTargetMachine):
		return FontInstallTargetMachine
	default:
		return ""
	}
}

func installDirectoryForTarget(target FontInstallTarget) (string, error) {
	switch normalizeFontInstallTarget(target) {
	case FontInstallTargetUser:
		return systemUserFontInstallDirectory()
	case FontInstallTargetMachine:
		return systemMachineFontInstallDirectory()
	default:
		return "", fmt.Errorf("unsupported font install target %q", target)
	}
}

func systemUserFontInstallDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Fonts"), nil
	case "windows":
		localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		if localAppData != "" {
			return filepath.Join(localAppData, "Microsoft", "Windows", "Fonts"), nil
		}
		return filepath.Join(home, "AppData", "Local", "Microsoft", "Windows", "Fonts"), nil
	default:
		dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
		if dataHome == "" {
			dataHome = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(dataHome, "fonts"), nil
	}
}

func systemMachineFontInstallDirectory() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Fonts", nil
	case "windows":
		windowsDir := strings.TrimSpace(os.Getenv("WINDIR"))
		if windowsDir == "" {
			windowsDir = `C:\Windows`
		}
		return filepath.Join(windowsDir, "Fonts"), nil
	default:
		return "/usr/local/share/fonts", nil
	}
}

func refreshInstalledFontCache(target FontInstallTarget, installDir string) error {
	switch normalizeFontInstallTarget(target) {
	case FontInstallTargetUser, FontInstallTargetMachine:
	default:
		return nil
	}

	switch runtime.GOOS {
	case "darwin":
		if err := refreshDarwinFontCache(); err != nil {
			return err
		}
	case "linux":
		if err := refreshLinuxFontCache(installDir); err != nil {
			return err
		}
	case "windows":
		return nil
	}
	return nil
}

func refreshDarwinFontCache() error {
	for _, command := range [][]string{{"pkill", "fontd"}, {"killall", "fontd"}} {
		path, err := exec.LookPath(command[0])
		if err != nil {
			continue
		}
		runCmd := exec.Command(path, command[1:]...)
		processutil.ConfigureCLI(runCmd)
		result, runErr := runCmd.CombinedOutput()
		if runErr == nil {
			return nil
		}
		output := strings.ToLower(strings.TrimSpace(string(result)))
		if strings.Contains(output, "no matching processes") || strings.Contains(output, "no process found") {
			return nil
		}
	}
	return nil
}

func refreshLinuxFontCache(installDir string) error {
	fcCachePath, err := exec.LookPath("fc-cache")
	if err != nil {
		return nil
	}
	command := exec.Command(fcCachePath, "-f", installDir)
	processutil.ConfigureCLI(command)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("refresh font cache: %w %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func writeAtomicFile(path string, data []byte, perm os.FileMode) error {
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".font-install-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(perm); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeRemoteSourcePrefix(value string, fallbackName string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		trimmed = strings.ToLower(strings.TrimSpace(fallbackName))
	}
	trimmed = strings.ReplaceAll(trimmed, "_", "-")
	trimmed = strings.Join(strings.Fields(trimmed), "-")
	var builder strings.Builder
	lastHyphen := false
	for _, r := range trimmed {
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLower || isDigit {
			builder.WriteRune(r)
			lastHyphen = false
			continue
		}
		if r == '-' && !lastHyphen && builder.Len() > 0 {
			builder.WriteRune(r)
			lastHyphen = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

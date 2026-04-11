package wails

import (
	"context"
	"encoding/base64"

	"dreamcreator/internal/application/events"
	"dreamcreator/internal/application/fonts/service"
	"dreamcreator/internal/domain/library"
)

type SystemHandler struct {
	fonts *service.FontService
	bus   events.Bus
}

func NewSystemHandler(fonts *service.FontService, bus events.Bus) *SystemHandler {
	return &SystemHandler{
		fonts: fonts,
		bus:   bus,
	}
}

func (handler *SystemHandler) publishFontsUpdated(ctx context.Context, payload map[string]interface{}) {
	if handler.bus == nil {
		return
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	_ = handler.bus.Publish(ctx, events.Event{
		Topic:   "system.fonts",
		Type:    "updated",
		Payload: payload,
	})
}

func (handler *SystemHandler) ServiceName() string {
	return "SystemHandler"
}

func (handler *SystemHandler) ListFontFamilies(ctx context.Context) ([]string, error) {
	families, err := handler.fonts.ListFontFamilies(ctx)
	if err != nil {
		return nil, err
	}
	if families == nil {
		return []string{}, nil
	}
	return families, nil
}

type FontCatalogFace struct {
	Name           string `json:"name"`
	FullName       string `json:"fullName,omitempty"`
	PostScriptName string `json:"postScriptName,omitempty"`
	Weight         int    `json:"weight,omitempty"`
	Italic         bool   `json:"italic,omitempty"`
}

type FontCatalogFamily struct {
	Family string            `json:"family"`
	Faces  []FontCatalogFace `json:"faces"`
}

func (handler *SystemHandler) ListFontCatalog(ctx context.Context) ([]FontCatalogFamily, error) {
	catalog, err := handler.fonts.ListFontCatalog(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]FontCatalogFamily, 0, len(catalog))
	for _, family := range catalog {
		faces := make([]FontCatalogFace, 0, len(family.Faces))
		for _, face := range family.Faces {
			faces = append(faces, FontCatalogFace{
				Name:           face.Name,
				FullName:       face.FullName,
				PostScriptName: face.PostScriptName,
				Weight:         face.Weight,
				Italic:         face.Italic,
			})
		}
		result = append(result, FontCatalogFamily{
			Family: family.Family,
			Faces:  faces,
		})
	}
	if result == nil {
		return []FontCatalogFamily{}, nil
	}
	return result, nil
}

type RefreshFontCatalogResult struct {
	FamilyCount int `json:"familyCount"`
}

func (handler *SystemHandler) RefreshFontCatalog(ctx context.Context) (RefreshFontCatalogResult, error) {
	if err := handler.fonts.RefreshCatalog(ctx); err != nil {
		return RefreshFontCatalogResult{}, err
	}
	families, err := handler.fonts.ListFontFamilies(ctx)
	if err != nil {
		return RefreshFontCatalogResult{}, err
	}
	handler.publishFontsUpdated(ctx, map[string]interface{}{
		"reason": "manual_refresh",
	})
	return RefreshFontCatalogResult{
		FamilyCount: len(families),
	}, nil
}

type CurrentUserProfile struct {
	Username     string `json:"username"`
	DisplayName  string `json:"displayName"`
	Initials     string `json:"initials,omitempty"`
	AvatarPath   string `json:"avatarPath,omitempty"`
	AvatarBase64 string `json:"avatarBase64,omitempty"`
	AvatarMime   string `json:"avatarMime,omitempty"`
}

func (handler *SystemHandler) GetCurrentUserProfile(ctx context.Context) (CurrentUserProfile, error) {
	return loadCurrentUserProfile(ctx)
}

type ExportedFontFamilyAsset struct {
	FileName      string `json:"fileName"`
	ContentBase64 string `json:"contentBase64"`
}

type ExportedFontFamily struct {
	Family string                    `json:"family"`
	Assets []ExportedFontFamilyAsset `json:"assets"`
}

func (handler *SystemHandler) ExportFontFamily(ctx context.Context, family string) (ExportedFontFamily, error) {
	exported, err := handler.fonts.ExportFontFamily(ctx, family)
	if err != nil {
		return ExportedFontFamily{}, err
	}

	result := ExportedFontFamily{
		Family: exported.Family,
		Assets: make([]ExportedFontFamilyAsset, 0, len(exported.Assets)),
	}
	for _, asset := range exported.Assets {
		result.Assets = append(result.Assets, ExportedFontFamilyAsset{
			FileName:      asset.FileName,
			ContentBase64: base64.StdEncoding.EncodeToString(asset.Content),
		})
	}
	if result.Assets == nil {
		result.Assets = []ExportedFontFamilyAsset{}
	}
	return result, nil
}

func (handler *SystemHandler) ExportFontFamilies(ctx context.Context, families []string) ([]ExportedFontFamily, error) {
	result := make([]ExportedFontFamily, 0, len(families))
	for _, family := range families {
		exported, err := handler.ExportFontFamily(ctx, family)
		if err != nil {
			return nil, err
		}
		result = append(result, exported)
	}
	return result, nil
}

type InstallRemoteFontSource struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Provider string `json:"provider,omitempty"`
	URL      string `json:"url,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Filename string `json:"filename,omitempty"`
	Priority int    `json:"priority,omitempty"`
	BuiltIn  bool   `json:"builtIn,omitempty"`
	Enabled  bool   `json:"enabled"`
}

type InstallRemoteFontFamilyRequest struct {
	Family   string                    `json:"family"`
	Target   string                    `json:"target,omitempty"`
	SourceID string                    `json:"sourceId,omitempty"`
	Sources  []InstallRemoteFontSource `json:"sources"`
}

type InstallRemoteFontFamilyResult struct {
	Family         string   `json:"family"`
	InstalledFiles []string `json:"installedFiles"`
	Target         string   `json:"target"`
}

type SearchRemoteFontFamilyRequest struct {
	Family  string                    `json:"family"`
	Sources []InstallRemoteFontSource `json:"sources"`
}

type RemoteFontSearchCandidate struct {
	SourceID           string `json:"sourceId"`
	SourceName         string `json:"sourceName"`
	FontID             string `json:"fontId"`
	Family             string `json:"family"`
	MatchName          string `json:"matchName"`
	MatchType          string `json:"matchType"`
	AssetCount         int    `json:"assetCount"`
	DeclaredAssetCount int    `json:"declaredAssetCount"`
	Installable        bool   `json:"installable"`
	Availability       string `json:"availability"`
	UnavailableReason  string `json:"unavailableReason,omitempty"`
}

type SyncRemoteFontSourceRequest struct {
	Source InstallRemoteFontSource `json:"source"`
}

type SyncRemoteFontSourceResult struct {
	SourceID           string                       `json:"sourceId"`
	SourceName         string                       `json:"sourceName"`
	RemoteFontManifest SyncRemoteFontSourceManifest `json:"remoteFontManifest,omitempty"`
	FontCount          int                          `json:"fontCount"`
	SyncStatus         string                       `json:"syncStatus"`
	LastSyncedAt       string                       `json:"lastSyncedAt,omitempty"`
	LastError          string                       `json:"lastError,omitempty"`
}

type SyncRemoteFontSourceManifest struct {
	SourceInfo SyncRemoteFontSourceManifestInfo    `json:"sourceInfo,omitempty"`
	Fonts      map[string]SyncRemoteFontSourceFont `json:"fonts,omitempty"`
}

type SyncRemoteFontSourceManifestInfo struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	APIEndpoint string `json:"apiEndpoint,omitempty"`
	Version     string `json:"version,omitempty"`
	LastUpdated string `json:"lastUpdated,omitempty"`
	TotalFonts  int    `json:"totalFonts,omitempty"`
}

type SyncRemoteFontSourceFont struct {
	Name          string                            `json:"name,omitempty"`
	Family        string                            `json:"family,omitempty"`
	License       string                            `json:"license,omitempty"`
	LicenseURL    string                            `json:"licenseUrl,omitempty"`
	Designer      string                            `json:"designer,omitempty"`
	Foundry       string                            `json:"foundry,omitempty"`
	Version       string                            `json:"version,omitempty"`
	Description   string                            `json:"description,omitempty"`
	Categories    []string                          `json:"categories,omitempty"`
	Tags          []string                          `json:"tags,omitempty"`
	Popularity    int                               `json:"popularity,omitempty"`
	LastModified  string                            `json:"lastModified,omitempty"`
	MetadataURL   string                            `json:"metadataUrl,omitempty"`
	SourceURL     string                            `json:"sourceUrl,omitempty"`
	Variants      []SyncRemoteFontSourceFontVariant `json:"variants,omitempty"`
	UnicodeRanges []string                          `json:"unicodeRanges,omitempty"`
	Languages     []string                          `json:"languages,omitempty"`
	SampleText    string                            `json:"sampleText,omitempty"`
}

type SyncRemoteFontSourceFontVariant struct {
	Name    string            `json:"name,omitempty"`
	Weight  int               `json:"weight,omitempty"`
	Style   string            `json:"style,omitempty"`
	Subsets []string          `json:"subsets,omitempty"`
	Files   map[string]string `json:"files,omitempty"`
}

func toSyncRemoteFontSourceManifest(value library.SubtitleStyleSourceManifest) SyncRemoteFontSourceManifest {
	fonts := make(map[string]SyncRemoteFontSourceFont, len(value.Fonts))
	for id, font := range value.Fonts {
		variants := make([]SyncRemoteFontSourceFontVariant, 0, len(font.Variants))
		for _, variant := range font.Variants {
			files := make(map[string]string, len(variant.Files))
			for fileType, rawURL := range variant.Files {
				files[fileType] = rawURL
			}
			variants = append(variants, SyncRemoteFontSourceFontVariant{
				Name:    variant.Name,
				Weight:  variant.Weight,
				Style:   variant.Style,
				Subsets: append([]string(nil), variant.Subsets...),
				Files:   files,
			})
		}
		fonts[id] = SyncRemoteFontSourceFont{
			Name:          font.Name,
			Family:        font.Family,
			License:       font.License,
			LicenseURL:    font.LicenseURL,
			Designer:      font.Designer,
			Foundry:       font.Foundry,
			Version:       font.Version,
			Description:   font.Description,
			Categories:    append([]string(nil), font.Categories...),
			Tags:          append([]string(nil), font.Tags...),
			Popularity:    font.Popularity,
			LastModified:  font.LastModified,
			MetadataURL:   font.MetadataURL,
			SourceURL:     font.SourceURL,
			Variants:      variants,
			UnicodeRanges: append([]string(nil), font.UnicodeRanges...),
			Languages:     append([]string(nil), font.Languages...),
			SampleText:    font.SampleText,
		}
	}
	return SyncRemoteFontSourceManifest{
		SourceInfo: SyncRemoteFontSourceManifestInfo{
			Name:        value.SourceInfo.Name,
			Description: value.SourceInfo.Description,
			URL:         value.SourceInfo.URL,
			APIEndpoint: value.SourceInfo.APIEndpoint,
			Version:     value.SourceInfo.Version,
			LastUpdated: value.SourceInfo.LastUpdated,
			TotalFonts:  value.SourceInfo.TotalFonts,
		},
		Fonts: fonts,
	}
}

func toRemoteFontSource(source InstallRemoteFontSource) service.RemoteFontSource {
	return service.RemoteFontSource{
		ID:       source.ID,
		Name:     source.Name,
		Provider: source.Provider,
		URL:      source.URL,
		Prefix:   source.Prefix,
		Filename: source.Filename,
		Priority: source.Priority,
		BuiltIn:  source.BuiltIn,
	}
}

func (handler *SystemHandler) SearchRemoteFontFamily(
	ctx context.Context,
	request SearchRemoteFontFamilyRequest,
) ([]RemoteFontSearchCandidate, error) {
	sources := make([]service.RemoteFontSource, 0, len(request.Sources))
	for _, source := range request.Sources {
		if !source.Enabled {
			continue
		}
		if source.Kind != "" && source.Kind != "font" {
			continue
		}
		sources = append(sources, toRemoteFontSource(source))
	}

	candidates, err := handler.fonts.SearchRemoteFontFamily(ctx, request.Family, sources)
	if err != nil {
		return nil, err
	}

	result := make([]RemoteFontSearchCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, RemoteFontSearchCandidate{
			SourceID:           candidate.SourceID,
			SourceName:         candidate.SourceName,
			FontID:             candidate.FontID,
			Family:             candidate.Family,
			MatchName:          candidate.MatchName,
			MatchType:          candidate.MatchType,
			AssetCount:         candidate.AssetCount,
			DeclaredAssetCount: candidate.DeclaredAssetCount,
			Installable:        candidate.Installable,
			Availability:       candidate.Availability,
			UnavailableReason:  candidate.UnavailableReason,
		})
	}
	if result == nil {
		return []RemoteFontSearchCandidate{}, nil
	}
	return result, nil
}

func (handler *SystemHandler) InstallRemoteFontFamily(
	ctx context.Context,
	request InstallRemoteFontFamilyRequest,
) (InstallRemoteFontFamilyResult, error) {
	sources := make([]service.RemoteFontSource, 0, len(request.Sources))
	for _, source := range request.Sources {
		if !source.Enabled {
			continue
		}
		if source.Kind != "" && source.Kind != "font" {
			continue
		}
		sources = append(sources, toRemoteFontSource(source))
	}

	installed, err := handler.fonts.InstallRemoteFontFamily(
		ctx,
		request.Family,
		sources,
		service.FontInstallTarget(request.Target),
		request.SourceID,
	)
	if err != nil {
		return InstallRemoteFontFamilyResult{}, err
	}

	handler.publishFontsUpdated(ctx, map[string]interface{}{
		"family":         installed.Family,
		"installedFiles": installed.InstalledFiles,
	})

	return InstallRemoteFontFamilyResult{
		Family:         installed.Family,
		InstalledFiles: installed.InstalledFiles,
		Target:         string(installed.Target),
	}, nil
}

func (handler *SystemHandler) SyncRemoteFontSource(
	ctx context.Context,
	request SyncRemoteFontSourceRequest,
) (SyncRemoteFontSourceResult, error) {
	status, err := handler.fonts.SyncRemoteFontSource(ctx, toRemoteFontSource(request.Source))
	if err != nil {
		return SyncRemoteFontSourceResult{
			SourceID:           status.SourceID,
			SourceName:         status.SourceName,
			RemoteFontManifest: toSyncRemoteFontSourceManifest(status.Manifest),
			FontCount:          status.FontCount,
			SyncStatus:         "error",
			LastSyncedAt:       status.LastSyncedAt,
			LastError:          status.LastError,
		}, nil
	}
	return SyncRemoteFontSourceResult{
		SourceID:           status.SourceID,
		SourceName:         status.SourceName,
		RemoteFontManifest: toSyncRemoteFontSourceManifest(status.Manifest),
		FontCount:          status.FontCount,
		SyncStatus:         status.SyncStatus,
		LastSyncedAt:       status.LastSyncedAt,
		LastError:          status.LastError,
	}, nil
}

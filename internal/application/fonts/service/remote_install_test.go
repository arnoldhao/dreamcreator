package service

import (
	"archive/zip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeFontInstallTarget(t *testing.T) {
	t.Parallel()

	if got := normalizeFontInstallTarget(""); got != FontInstallTargetUser {
		t.Fatalf("expected empty target to normalize to user, got %q", got)
	}
	if got := normalizeFontInstallTarget(FontInstallTargetMachine); got != FontInstallTargetMachine {
		t.Fatalf("expected machine target to stay machine, got %q", got)
	}
	if got := normalizeFontInstallTarget("unknown"); got != "" {
		t.Fatalf("expected unknown target to normalize to empty, got %q", got)
	}
}

func TestNormalizeRemoteFontSourcesIncludesFontGetFields(t *testing.T) {
	t.Parallel()

	sources := normalizeRemoteFontSources([]RemoteFontSource{{
		Name:     "Google Fonts",
		Provider: "fontget",
		URL:      "https://example.com/google-fonts.json",
		Prefix:   "google",
		Filename: "google-fonts.json",
		Priority: 1,
		BuiltIn:  true,
	}})
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].ID != "google" {
		t.Fatalf("expected derived source id from prefix, got %q", sources[0].ID)
	}
	if sources[0].Provider != "fontget" {
		t.Fatalf("expected fontget provider, got %q", sources[0].Provider)
	}
	if sources[0].Priority != 1 {
		t.Fatalf("expected priority to survive normalization, got %d", sources[0].Priority)
	}
}

func TestNormalizeRemoteFontSourcesRejectsNonFontGetProvider(t *testing.T) {
	t.Parallel()

	sources := normalizeRemoteFontSources([]RemoteFontSource{{
		Provider: "github",
		URL:      "https://example.com/fonts.json",
	}})
	if len(sources) != 0 {
		t.Fatalf("expected non-fontget providers to be rejected, got %#v", sources)
	}
}

func TestScoreRemoteFontCatalogEntryPrefersExactFamilyThenAlias(t *testing.T) {
	t.Parallel()

	entry := remoteFontCatalogEntry{
		ID:      "google.noto-sans-sc",
		Name:    "Noto Sans SC",
		Family:  "Noto Sans SC",
		Aliases: []string{"NotoSansSC", "Noto Sans CJK SC"},
		Assets: []remoteFontDownloadAsset{{
			FileName: "NotoSansSC-Regular.ttf",
			URL:      "https://example.com/NotoSansSC-Regular.ttf",
		}},
	}

	matchName, matchType, score := scoreRemoteFontCatalogEntry(normalizeFontFamilyKey("Noto Sans SC"), entry)
	if matchType != "exact_family" || matchName != "Noto Sans SC" || score != 100 {
		t.Fatalf("expected exact family match, got %q %q %d", matchName, matchType, score)
	}

	matchName, matchType, score = scoreRemoteFontCatalogEntry(normalizeFontFamilyKey("NotoSansSC"), entry)
	if matchType != "exact_alias" || matchName != "NotoSansSC" || score != 92 {
		t.Fatalf("expected exact alias match, got %q %q %d", matchName, matchType, score)
	}
}

func TestParseRemoteFontCatalogSupportsFontGetSchema(t *testing.T) {
	t.Parallel()

	source := RemoteFontSource{
		ID:       "google",
		Name:     "Google Fonts",
		Provider: "fontget",
		URL:      "https://example.com/google-fonts.json",
		Prefix:   "google",
	}
	raw := []byte(`{
		"source_info": {
			"name": "Google Fonts",
			"description": "Google font catalog",
			"api_endpoint": "https://fonts.google.com/metadata/fonts",
			"version": "1.0.0",
			"last_updated": "2026-03-25T05:23:08.830701Z",
			"total_fonts": 1037
		},
		"fonts": {
			"roboto": {
				"name": "Roboto",
				"family": "Roboto",
				"license": "OFL",
				"license_url": "https://example.com/license",
				"designer": "Google",
				"foundry": "Google",
				"version": "1.0",
				"popularity": 87,
				"categories": ["Sans Serif"],
				"tags": ["ui"],
				"metadata_url": "https://example.com/roboto",
				"source_url": "https://example.com/roboto-source",
				"unicode_ranges": ["U+0000-00FF"],
				"languages": ["Latin"],
				"sample_text": "The quick brown fox jumps over the lazy dog",
				"variants": [
					{
						"name": "Regular",
						"weight": 400,
						"style": "normal",
						"subsets": ["latin"],
						"files": {
							"ttf": "https://example.com/Roboto-Regular.ttf"
						}
					}
				]
			}
		}
	}`)

	catalog, err := parseRemoteFontCatalog(raw, source)
	if err != nil {
		t.Fatalf("expected fontget schema to parse, got %v", err)
	}
	entry, exists := catalog.fonts["roboto"]
	if !exists {
		t.Fatalf("expected roboto entry, got %#v", catalog.fonts)
	}
	if entry.Family != "Roboto" {
		t.Fatalf("expected family Roboto, got %q", entry.Family)
	}
	if len(entry.Assets) != 1 {
		t.Fatalf("expected 1 asset, got %#v", entry.Assets)
	}
	if entry.Assets[0].URL != "https://example.com/Roboto-Regular.ttf" {
		t.Fatalf("expected variant download url to survive, got %q", entry.Assets[0].URL)
	}
	if catalog.manifest.SourceInfo.TotalFonts != 1037 {
		t.Fatalf("expected total_fonts from source_info to survive, got %#v", catalog.manifest.SourceInfo)
	}
	if catalog.manifest.SourceInfo.APIEndpoint != "https://fonts.google.com/metadata/fonts" {
		t.Fatalf("expected api endpoint to survive, got %#v", catalog.manifest.SourceInfo)
	}
	if catalog.manifest.Fonts["roboto"].Variants[0].Subsets[0] != "latin" {
		t.Fatalf("expected raw manifest variants to survive, got %#v", catalog.manifest.Fonts["roboto"])
	}
}

func TestParseRemoteFontCatalogRejectsLegacySchema(t *testing.T) {
	t.Parallel()

	source := RemoteFontSource{
		ID:       "legacy",
		Name:     "Legacy Source",
		Provider: "fontget",
		URL:      "https://example.com/legacy-fonts.json",
	}
	raw := []byte(`{
		"fonts": [
			{
				"id": "noto-sans-sc",
				"family": "Noto Sans SC",
				"files": [
					{
						"path": "fonts/NotoSansSC-Regular.ttf"
					}
				]
			}
		]
	}`)

	if _, err := parseRemoteFontCatalog(raw, source); err == nil {
		t.Fatalf("expected legacy schema to be rejected")
	}
}

func TestParseRemoteFontCatalogSupportsFontGetFontfacekitURLWithoutExtension(t *testing.T) {
	t.Parallel()

	source := RemoteFontSource{
		ID:       "fontget-font-squirrel",
		Name:     "Font Squirrel",
		Provider: "fontget",
		URL:      "https://example.com/font-squirrel.json",
		Prefix:   "squirrel",
	}
	raw := []byte(`{
		"source_info": {
			"name": "Font Squirrel",
			"version": "1.0.0",
			"last_updated": "2026-03-25T05:23:08.830701Z",
			"total_fonts": 1
		},
		"fonts": {
			"1942-report": {
				"name": "1942 report",
				"family": "1942 report",
				"variants": [
					{
						"name": "1942 report Regular",
						"weight": 400,
						"style": "normal",
						"files": {
							"ttf": "https://www.fontsquirrel.com/fontfacekit/1942-report"
						}
					}
				]
			}
		}
	}`)

	catalog, err := parseRemoteFontCatalog(raw, source)
	if err != nil {
		t.Fatalf("expected fontfacekit schema to parse, got %v", err)
	}
	entry, exists := catalog.fonts["1942-report"]
	if !exists {
		t.Fatalf("expected 1942-report entry, got %#v", catalog.fonts)
	}
	if len(entry.Assets) != 1 {
		t.Fatalf("expected 1 installable asset, got %#v", entry.Assets)
	}
	if entry.Assets[0].FileName != "1942report.ttf" {
		t.Fatalf("expected synthetic .ttf filename, got %#v", entry.Assets[0])
	}
	if entry.DeclaredAssetCount != 1 {
		t.Fatalf("expected declared asset count to survive, got %#v", entry)
	}
	if entry.UnavailableReason != "" {
		t.Fatalf("expected installable entry to have no unavailable reason, got %#v", entry)
	}
}

func TestParseRemoteFontCatalogRetainsUnavailableMatchReason(t *testing.T) {
	t.Parallel()

	source := RemoteFontSource{
		ID:       "fontget-font-squirrel",
		Name:     "Font Squirrel",
		Provider: "fontget",
		URL:      "https://example.com/font-squirrel.json",
		Prefix:   "squirrel",
	}
	raw := []byte(`{
		"source_info": {
			"name": "Font Squirrel",
			"version": "1.0.0",
			"last_updated": "2026-03-25T05:23:08.830701Z",
			"total_fonts": 1
		},
		"fonts": {
			"1942-report": {
				"name": "1942 report",
				"family": "1942 report",
				"variants": [
					{
						"name": "1942 report Regular",
						"weight": 400,
						"style": "normal",
						"files": {
							"woff": "https://www.fontsquirrel.com/fontfacekit/1942-report"
						}
					}
				]
			}
		}
	}`)

	catalog, err := parseRemoteFontCatalog(raw, source)
	if err != nil {
		t.Fatalf("expected catalog to parse, got %v", err)
	}
	entry, exists := catalog.fonts["1942-report"]
	if !exists {
		t.Fatalf("expected 1942-report entry, got %#v", catalog.fonts)
	}
	if len(entry.Assets) != 0 {
		t.Fatalf("expected no installable assets, got %#v", entry.Assets)
	}
	if entry.DeclaredAssetCount != 1 {
		t.Fatalf("expected declared asset count to survive, got %#v", entry)
	}
	if entry.UnavailableReason != remoteFontAvailabilityUnsupportedAsset {
		t.Fatalf("expected unsupported asset reason, got %#v", entry)
	}
}

func TestResolveDownloadedFontFilesDetectsZipByContent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "font-asset.bin")
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}

	zipWriter := zip.NewWriter(archiveFile)
	fontEntry, err := zipWriter.Create("TestFont.ttf")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := fontEntry.Write([]byte("dummy-font-data")); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatalf("close archive file: %v", err)
	}

	fontFiles, err := resolveDownloadedFontFiles(archivePath, "1942report.ttf", tempDir)
	if err != nil {
		t.Fatalf("expected zip archive to be detected by content, got %v", err)
	}
	if len(fontFiles) != 1 || filepath.Base(fontFiles[0]) != "TestFont.ttf" {
		t.Fatalf("expected extracted TTF file, got %#v", fontFiles)
	}
}

func TestDownloadRemoteFontAssetReportsWAFChallenge(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("x-amzn-waf-action", "challenge")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	_, err := downloadRemoteFontAsset(context.Background(), t.TempDir(), remoteFontDownloadAsset{
		FileName: "1942report.ttf",
		URL:      server.URL,
	})
	if err == nil || !strings.Contains(err.Error(), "blocked the download request") {
		t.Fatalf("expected waf challenge error, got %v", err)
	}
}

func TestSyncRemoteFontSourceReportsFontCount(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{
			"source_info": {
				"name": "Google Fonts",
				"version": "1.0.0",
				"last_updated": "2026-03-25T05:23:08.830701Z",
				"total_fonts": 1037
			},
			"fonts": {
				"roboto": {
					"name": "Roboto",
					"family": "Roboto",
					"variants": [
						{
							"name": "Regular",
							"files": {
								"ttf": "https://example.com/Roboto-Regular.ttf"
							}
						}
					]
				}
			}
		}`))
	}))
	defer server.Close()

	service := NewFontService()
	status, err := service.SyncRemoteFontSource(context.Background(), RemoteFontSource{
		ID:       "fontget-google-fonts",
		Name:     "Google Fonts",
		Provider: "fontget",
		URL:      server.URL,
		Prefix:   "google",
		Filename: "google-fonts.json",
		Priority: 1,
		BuiltIn:  true,
	})
	if err != nil {
		t.Fatalf("expected sync to succeed, got %v", err)
	}
	if status.SourceID != "fontget-google-fonts" {
		t.Fatalf("expected source id to survive sync, got %#v", status)
	}
	if status.FontCount != 1037 {
		t.Fatalf("expected font count from source info, got %#v", status)
	}
	if status.SyncStatus != "synced" {
		t.Fatalf("expected sync status synced, got %#v", status)
	}
	if status.LastSyncedAt == "" {
		t.Fatalf("expected last synced at to be populated, got %#v", status)
	}
	if status.Manifest.SourceInfo.TotalFonts != 1037 {
		t.Fatalf("expected manifest source info to survive sync, got %#v", status)
	}
	if status.Manifest.SourceInfo.LastUpdated != "2026-03-25T05:23:08.830701Z" {
		t.Fatalf("expected manifest last updated to survive sync, got %#v", status)
	}
	if status.Manifest.Fonts["roboto"].Family != "Roboto" {
		t.Fatalf("expected manifest fonts to survive sync, got %#v", status)
	}
}

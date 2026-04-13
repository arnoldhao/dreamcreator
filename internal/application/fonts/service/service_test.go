package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

func TestFontNamesFromFontDataIncludesPostScriptAlias(t *testing.T) {
	entries := fontNamesFromFontData(goregular.TTF)
	if len(entries) != 1 {
		t.Fatalf("expected one font entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.family != "Go" {
		t.Fatalf("expected family Go, got %q", entry.family)
	}
	if !containsFontLookupKey(entry.lookupKeys, "Go") {
		t.Fatalf("expected family alias in lookup keys, got %v", entry.lookupKeys)
	}
	if !containsFontLookupKey(entry.lookupKeys, "Go Regular") {
		t.Fatalf("expected full name alias in lookup keys, got %v", entry.lookupKeys)
	}
	if !containsFontLookupKey(entry.lookupKeys, "GoRegular") {
		t.Fatalf("expected postscript alias in lookup keys, got %v", entry.lookupKeys)
	}
	if entry.faceIndex != -1 {
		t.Fatalf("expected standalone font face index -1, got %d", entry.faceIndex)
	}
	if entry.face != "Regular" {
		t.Fatalf("expected regular font face, got %q", entry.face)
	}
	if entry.weight != 400 {
		t.Fatalf("expected regular font weight 400, got %d", entry.weight)
	}
	if entry.italic {
		t.Fatalf("expected regular face to not be italic")
	}
}

func TestListFontCatalogReturnsFamilyFaces(t *testing.T) {
	service := NewFontService()
	service.loaded = true
	service.catalog = fontCatalog{
		familyCatalog: []FontCatalogFamily{{
			Family: "PingFang SC",
			Faces: []FontCatalogFace{
				{
					Name:           "Regular",
					FullName:       "PingFang SC Regular",
					PostScriptName: "PingFangSC-Regular",
					Weight:         400,
					Italic:         false,
				},
				{
					Name:           "Semibold",
					FullName:       "PingFang SC Semibold",
					PostScriptName: "PingFangSC-Semibold",
					Weight:         600,
					Italic:         false,
				},
			},
		}},
	}

	result, err := service.ListFontCatalog(context.Background())
	if err != nil {
		t.Fatalf("ListFontCatalog returned error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected one font family, got %#v", result)
	}
	if result[0].Family != "PingFang SC" {
		t.Fatalf("expected PingFang SC family, got %#v", result[0])
	}
	if len(result[0].Faces) != 2 {
		t.Fatalf("expected two faces, got %#v", result[0].Faces)
	}
	if result[0].Faces[1].Name != "Semibold" || result[0].Faces[1].Weight != 600 || result[0].Faces[1].PostScriptName != "PingFangSC-Semibold" {
		t.Fatalf("expected semibold face metadata, got %#v", result[0].Faces[1])
	}

	result[0].Faces[0].Name = "Mutated"
	again, err := service.ListFontCatalog(context.Background())
	if err != nil {
		t.Fatalf("ListFontCatalog second call returned error: %v", err)
	}
	if again[0].Faces[0].Name != "Regular" {
		t.Fatalf("expected ListFontCatalog to return a defensive copy, got %#v", again[0].Faces[0])
	}
}

func TestExportFontFamilyPreservesRequestedAlias(t *testing.T) {
	tempDir := t.TempDir()
	fontPath := filepath.Join(tempDir, "Go-Regular.ttf")
	if err := os.WriteFile(fontPath, goregular.TTF, 0o644); err != nil {
		t.Fatalf("write font file: %v", err)
	}

	service := NewFontService()
	service.loaded = true
	service.catalog = fontCatalog{
		families: []string{"Go"},
		filesByFamily: map[string][]fontFileEntry{
			normalizeFontFamilyKey("Go"): {
				{family: "Go", fileName: filepath.Base(fontPath), path: fontPath},
			},
			normalizeFontFamilyKey("Go Regular"): {
				{family: "Go", fileName: filepath.Base(fontPath), path: fontPath},
			},
			normalizeFontFamilyKey("GoRegular"): {
				{family: "Go", fileName: filepath.Base(fontPath), path: fontPath},
			},
		},
	}

	exported, err := service.ExportFontFamily(context.Background(), "Go-Regular")
	if err != nil {
		t.Fatalf("ExportFontFamily returned error: %v", err)
	}
	if exported.Family != "Go-Regular" {
		t.Fatalf("expected exported family Go-Regular, got %q", exported.Family)
	}
	if len(exported.Assets) != 1 {
		t.Fatalf("expected one exported asset, got %d", len(exported.Assets))
	}
}

func TestExportFontFamilyKeepsRequestedFamilyWhenCatalogIncludesVariantEntries(t *testing.T) {
	tempDir := t.TempDir()
	regularPath := filepath.Join(tempDir, "MicrosoftYaHei-Regular.ttf")
	lightPath := filepath.Join(tempDir, "MicrosoftYaHei-Light.ttf")
	if err := os.WriteFile(regularPath, goregular.TTF, 0o644); err != nil {
		t.Fatalf("write regular font file: %v", err)
	}
	if err := os.WriteFile(lightPath, goregular.TTF, 0o644); err != nil {
		t.Fatalf("write light font file: %v", err)
	}

	service := NewFontService()
	service.loaded = true
	service.catalog = fontCatalog{
		families: []string{"Microsoft YaHei"},
		filesByFamily: map[string][]fontFileEntry{
			normalizeFontFamilyKey("Microsoft YaHei"): {
				{
					family:   "Microsoft YaHei",
					fileName: filepath.Base(regularPath),
					path:     regularPath,
				},
				{
					family:   "Microsoft YaHei Light",
					fileName: filepath.Base(lightPath),
					path:     lightPath,
				},
			},
		},
	}

	exported, err := service.ExportFontFamily(context.Background(), "Microsoft YaHei")
	if err != nil {
		t.Fatalf("ExportFontFamily returned error: %v", err)
	}
	if exported.Family != "Microsoft YaHei" {
		t.Fatalf("expected exported family Microsoft YaHei, got %q", exported.Family)
	}
	if len(exported.Assets) != 2 {
		t.Fatalf("expected two exported assets, got %d", len(exported.Assets))
	}
}

func TestFontNamesFromFontDataPreservesCollectionFaces(t *testing.T) {
	collection := buildTestTTC(goregular.TTF, goregular.TTF)

	entries := fontNamesFromFontData(collection)
	if len(entries) != 2 {
		t.Fatalf("expected two collection font entries, got %d", len(entries))
	}
	if entries[0].faceIndex != 0 || entries[1].faceIndex != 1 {
		t.Fatalf("expected collection face indices [0 1], got [%d %d]", entries[0].faceIndex, entries[1].faceIndex)
	}
}

func TestExtractFontFaceFromCollectionBytes(t *testing.T) {
	collection := buildTestTTC(goregular.TTF, goregular.TTF)

	extracted, err := extractFontFaceFromCollectionBytes(collection, 1)
	if err != nil {
		t.Fatalf("extractFontFaceFromCollectionBytes returned error: %v", err)
	}
	if bytes.HasPrefix(extracted, []byte("ttcf")) {
		t.Fatalf("expected extracted font to be standalone sfnt data, got collection header")
	}
	if _, err := opentype.Parse(extracted); err != nil {
		t.Fatalf("expected extracted font to be parseable, got %v", err)
	}
}

func TestResolveCatalogFontFamilyPrefersPublicLegacyFamily(t *testing.T) {
	got := resolveCatalogFontFamily("Go UI", "", "Go", "Go Regular", "GoRegular")
	if got != "Go" {
		t.Fatalf("expected legacy family Go, got %q", got)
	}
}

func TestResolveCatalogFontFamilyFallsBackToTypographicWhenLegacyMissing(t *testing.T) {
	got := resolveCatalogFontFamily("Inter", "", "", "Inter Regular", "Inter-Regular")
	if got != "Inter" {
		t.Fatalf("expected typographic family Inter, got %q", got)
	}
}

func TestResolveCatalogFontFamilySuppressesHiddenSystemFamilies(t *testing.T) {
	got := resolveCatalogFontFamily("New York", "", ".New York", ".New York Regular", ".NewYork-Regular")
	if got != "" {
		t.Fatalf("expected hidden system family to be suppressed, got %q", got)
	}
}

func TestResolveCatalogFontFacePrefersPostScriptSuffixOverLocalizedSubfamily(t *testing.T) {
	got := resolveCatalogFontFace(
		"半粗體",
		"",
		"半粗體",
		"蘋方-繁 半粗體",
		"PingFang TC",
		"PingFangTC-Semibold",
	)
	if got != "Semibold" {
		t.Fatalf("expected postscript-derived face Semibold, got %q", got)
	}
}

func buildTestTTC(fonts ...[]byte) []byte {
	headerSize := 12 + len(fonts)*4
	offsets := make([]int, len(fonts))
	totalSize := alignTestTTCOffset(headerSize)
	for index, font := range fonts {
		offsets[index] = totalSize
		totalSize += alignTestTTCOffset(len(font))
	}

	data := make([]byte, totalSize)
	copy(data[:4], []byte("ttcf"))
	binary.BigEndian.PutUint32(data[4:8], 0x00010000)
	binary.BigEndian.PutUint32(data[8:12], uint32(len(fonts)))
	for index, offset := range offsets {
		binary.BigEndian.PutUint32(data[12+index*4:16+index*4], uint32(offset))
		copy(data[offset:offset+len(fonts[index])], fonts[index])
		patchStandaloneFontOffsetsToCollection(data[offset:offset+len(fonts[index])], uint32(offset))
	}
	return data
}

func alignTestTTCOffset(length int) int {
	if length <= 0 {
		return 0
	}
	return (length + 3) &^ 3
}

func patchStandaloneFontOffsetsToCollection(fontData []byte, baseOffset uint32) {
	if len(fontData) < 12 {
		return
	}
	numTables := int(binary.BigEndian.Uint16(fontData[4:6]))
	for index := 0; index < numTables; index++ {
		recordOffset := 12 + index*16 + 8
		if len(fontData) < recordOffset+4 {
			return
		}
		tableOffset := binary.BigEndian.Uint32(fontData[recordOffset : recordOffset+4])
		binary.BigEndian.PutUint32(fontData[recordOffset:recordOffset+4], tableOffset+baseOffset)
	}
}

func containsFontLookupKey(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

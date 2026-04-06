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
}

func TestExportFontFamilyMatchesHyphenatedAlias(t *testing.T) {
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
	if exported.Family != "Go" {
		t.Fatalf("expected exported family Go, got %q", exported.Family)
	}
	if len(exported.Assets) != 1 {
		t.Fatalf("expected one exported asset, got %d", len(exported.Assets))
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

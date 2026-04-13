//go:build windows

package service

import (
	"context"
	"encoding/binary"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/charmap"
)

const (
	windowsDefaultCharset = 1
	windowsFwRegular      = 400
	windowsLFSize         = 32
	windowsGDIError       = 0xffffffff

	windowsNameIDFamily            = 1
	windowsNameIDFull              = 4
	windowsNameIDPostScript        = 6
	windowsNameIDTypographicFamily = 16
	windowsNameIDWWSFamily         = 21
)

var (
	windowsGDI32                  = windows.NewLazySystemDLL("gdi32.dll")
	windowsProcCreateCompatibleDC = windowsGDI32.NewProc("CreateCompatibleDC")
	windowsProcCreateFontIndirect = windowsGDI32.NewProc("CreateFontIndirectW")
	windowsProcDeleteDC           = windowsGDI32.NewProc("DeleteDC")
	windowsProcDeleteObject       = windowsGDI32.NewProc("DeleteObject")
	windowsProcGetFontData        = windowsGDI32.NewProc("GetFontData")
	windowsProcGetTextFace        = windowsGDI32.NewProc("GetTextFaceW")
	windowsProcSelectObject       = windowsGDI32.NewProc("SelectObject")
)

type windowsLogFont struct {
	Height         int32
	Width          int32
	Escapement     int32
	Orientation    int32
	Weight         int32
	Italic         byte
	Underline      byte
	StrikeOut      byte
	CharSet        byte
	OutPrecision   byte
	ClipPrecision  byte
	Quality        byte
	PitchAndFamily byte
	FaceName       [windowsLFSize]uint16
}

func augmentPlatformFontCatalog(ctx context.Context, catalog *fontCatalog) error {
	if catalog == nil || len(catalog.filesByFamily) == 0 {
		return nil
	}

	seenEntries := make(map[string]struct{}, len(catalog.filesByFamily)*4)
	aliasCache := make(map[string][]string, len(catalog.families))
	for lookupKey, entries := range catalog.filesByFamily {
		for _, entry := range entries {
			seenEntries[buildWindowsFontEntryIdentity(lookupKey, entry)] = struct{}{}
		}
	}

	for _, family := range catalog.families {
		if err := ctx.Err(); err != nil {
			return err
		}
		normalizedFamily := normalizeFontFamilyKey(family)
		entries := append([]fontFileEntry(nil), catalog.filesByFamily[normalizedFamily]...)
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if strings.TrimSpace(entry.path) == "" {
				continue
			}

			cacheKey := entry.path + "\x00" + strconv.Itoa(entry.faceIndex)
			aliases, ok := aliasCache[cacheKey]
			if !ok {
				aliases = readWindowsFontAliases(entry)
				aliasCache[cacheKey] = aliases
			}

			for _, alias := range aliases {
				normalizedAlias := normalizeFontFamilyKey(alias)
				if normalizedAlias == "" || normalizedAlias == normalizedFamily {
					continue
				}
				seenKey := buildWindowsFontEntryIdentity(normalizedAlias, entry)
				if _, exists := seenEntries[seenKey]; exists {
					continue
				}
				seenEntries[seenKey] = struct{}{}
				catalog.filesByFamily[normalizedAlias] = append(catalog.filesByFamily[normalizedAlias], entry)
			}
		}
	}

	return nil
}

func exportPlatformFontFamily(ctx context.Context, candidates []string) (ExportedFontFamily, bool, error) {
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return ExportedFontFamily{}, false, err
		}
		normalizedCandidate := normalizeFontFamilyKey(candidate)
		if normalizedCandidate == "" {
			continue
		}
		if _, exists := seen[normalizedCandidate]; exists {
			continue
		}
		seen[normalizedCandidate] = struct{}{}

		exported, err := exportWindowsGDIFont(candidate)
		if err == nil {
			return exported, true, nil
		}
	}
	return ExportedFontFamily{}, false, nil
}

func buildWindowsFontEntryIdentity(lookupKey string, entry fontFileEntry) string {
	return lookupKey + "\x00" + entry.path + "\x00" + strconv.Itoa(entry.faceIndex)
}

func readWindowsFontAliases(entry fontFileEntry) []string {
	data, err := os.ReadFile(entry.path)
	if err != nil {
		return nil
	}
	if size := int64(len(data)); size <= 0 || size > maxFontFileSizeBytes {
		return nil
	}
	if entry.faceIndex >= 0 && isFontCollectionData(data) {
		data, err = extractFontFaceFromCollectionBytes(data, entry.faceIndex)
		if err != nil {
			return nil
		}
	}
	return extractWindowsFontAliasesFromData(data)
}

func extractWindowsFontAliasesFromData(data []byte) []string {
	result := extractWindowsSFNTAliases(data)
	if len(result) > 0 {
		return result
	}

	entries := fontNamesFromFontData(data)
	for _, entry := range entries {
		result = appendWindowsFontAlias(result, entry.family)
		result = appendWindowsFontAlias(result, entry.fullName)
		result = appendWindowsFontAlias(result, entry.postScript)
	}
	return result
}

func appendWindowsFontAlias(values []string, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return values
	}
	normalized := normalizeFontFamilyKey(trimmed)
	for _, existing := range values {
		if normalizeFontFamilyKey(existing) == normalized {
			return values
		}
	}
	return append(values, trimmed)
}

func extractWindowsSFNTAliases(data []byte) []string {
	if len(data) < 12 || isFontCollectionData(data) {
		return nil
	}

	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	if len(data) < 12+numTables*16 {
		return nil
	}

	nameTableOffset := -1
	nameTableLength := 0
	for i := 0; i < numTables; i++ {
		recordOffset := 12 + i*16
		if string(data[recordOffset:recordOffset+4]) != "name" {
			continue
		}
		nameTableOffset = int(binary.BigEndian.Uint32(data[recordOffset+8 : recordOffset+12]))
		nameTableLength = int(binary.BigEndian.Uint32(data[recordOffset+12 : recordOffset+16]))
		break
	}
	if nameTableOffset < 0 || nameTableLength <= 0 || len(data) < nameTableOffset+nameTableLength {
		return nil
	}

	table := data[nameTableOffset : nameTableOffset+nameTableLength]
	if len(table) < 6 {
		return nil
	}

	recordCount := int(binary.BigEndian.Uint16(table[2:4]))
	stringOffset := int(binary.BigEndian.Uint16(table[4:6]))
	if recordCount <= 0 || len(table) < 6+recordCount*12 || stringOffset < 0 || stringOffset > len(table) {
		return nil
	}

	result := make([]string, 0, recordCount)
	for i := 0; i < recordCount; i++ {
		record := table[6+i*12 : 6+(i+1)*12]
		nameID := binary.BigEndian.Uint16(record[6:8])
		if !isWindowsFontAliasNameID(nameID) {
			continue
		}

		valueLength := int(binary.BigEndian.Uint16(record[8:10]))
		valueOffset := int(binary.BigEndian.Uint16(record[10:12]))
		start := stringOffset + valueOffset
		end := start + valueLength
		if start < 0 || valueLength <= 0 || start > len(table) || end > len(table) {
			continue
		}

		decoded := decodeWindowsFontNameRecord(
			binary.BigEndian.Uint16(record[0:2]),
			binary.BigEndian.Uint16(record[2:4]),
			table[start:end],
		)
		result = appendWindowsFontAlias(result, decoded)
	}
	return result
}

func isWindowsFontAliasNameID(value uint16) bool {
	switch value {
	case windowsNameIDFamily,
		windowsNameIDFull,
		windowsNameIDPostScript,
		windowsNameIDTypographicFamily,
		windowsNameIDWWSFamily:
		return true
	default:
		return false
	}
}

func decodeWindowsFontNameRecord(platformID uint16, encodingID uint16, data []byte) string {
	switch platformID {
	case 0:
		return decodeWindowsUTF16BE(data)
	case 1:
		if encodingID == 0 {
			decoded, err := charmap.Macintosh.NewDecoder().Bytes(data)
			if err == nil {
				return strings.TrimSpace(string(decoded))
			}
		}
		return strings.TrimSpace(string(data))
	case 3:
		return decodeWindowsUTF16BE(data)
	default:
		return ""
	}
}

func decodeWindowsUTF16BE(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	if len(data) == 0 {
		return ""
	}
	u16 := make([]uint16, 0, len(data)/2)
	for offset := 0; offset < len(data); offset += 2 {
		u16 = append(u16, binary.BigEndian.Uint16(data[offset:offset+2]))
	}
	return strings.TrimSpace(syscall.UTF16ToString(u16))
}

func exportWindowsGDIFont(family string) (ExportedFontFamily, error) {
	trimmedFamily := strings.TrimSpace(family)
	if trimmedFamily == "" {
		return ExportedFontFamily{}, syscall.EINVAL
	}

	dc, _, createDCErr := windowsProcCreateCompatibleDC.Call(0)
	if dc == 0 {
		return ExportedFontFamily{}, createDCErr
	}
	defer windowsProcDeleteDC.Call(dc)

	var logFont windowsLogFont
	logFont.CharSet = windowsDefaultCharset
	logFont.Weight = windowsFwRegular
	if !copyWindowsFaceName(&logFont.FaceName, trimmedFamily) {
		return ExportedFontFamily{}, syscall.EINVAL
	}

	fontHandle, _, createFontErr := windowsProcCreateFontIndirect.Call(uintptr(unsafe.Pointer(&logFont)))
	if fontHandle == 0 {
		return ExportedFontFamily{}, createFontErr
	}
	defer windowsProcDeleteObject.Call(fontHandle)

	previousObject, _, _ := windowsProcSelectObject.Call(dc, fontHandle)
	if previousObject != 0 {
		defer windowsProcSelectObject.Call(dc, previousObject)
	}

	fontData, err := readWindowsGDIFontData(dc)
	if err != nil {
		return ExportedFontFamily{}, err
	}

	actualFamily := readWindowsTextFaceName(dc)
	aliases := extractWindowsFontAliasesFromData(fontData)
	if !windowsFontAliasesContain(aliases, trimmedFamily) && normalizeFontFamilyKey(actualFamily) != normalizeFontFamilyKey(trimmedFamily) {
		return ExportedFontFamily{}, syscall.ENOENT
	}

	entries := fontNamesFromFontData(fontData)
	resolvedFamily := firstNonEmpty(actualFamily, trimmedFamily)
	fileBase := resolvedFamily
	if len(entries) > 0 {
		resolvedFamily = firstNonEmpty(entries[0].family, resolvedFamily, trimmedFamily)
		fileBase = firstNonEmpty(entries[0].postScript, entries[0].fullName, entries[0].family, fileBase)
	}
	fileBase = sanitizeFontAssetFileName(fileBase)
	if fileBase == "" {
		fileBase = "font"
	}

	return ExportedFontFamily{
		Family: resolvedFamily,
		Assets: []ExportedFontAsset{{
			FileName: fileBase + inferWindowsFontFileExtension(fontData),
			Content:  fontData,
		}},
	}, nil
}

func copyWindowsFaceName(target *[windowsLFSize]uint16, value string) bool {
	encoded, err := windows.UTF16FromString(strings.TrimSpace(value))
	if err != nil || len(encoded) > len(target) {
		return false
	}
	for index := range target {
		target[index] = 0
	}
	copy(target[:], encoded)
	return true
}

func readWindowsGDIFontData(dc uintptr) ([]byte, error) {
	size, _, sizeErr := windowsProcGetFontData.Call(dc, 0, 0, 0, 0)
	if size == windowsGDIError || size == 0 {
		return nil, sizeErr
	}

	buffer := make([]byte, int(size))
	read, _, readErr := windowsProcGetFontData.Call(
		dc,
		0,
		0,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if read == windowsGDIError || int(read) != len(buffer) {
		return nil, readErr
	}
	return buffer, nil
}

func readWindowsTextFaceName(dc uintptr) string {
	buffer := make([]uint16, 256)
	length, _, _ := windowsProcGetTextFace.Call(
		dc,
		uintptr(len(buffer)),
		uintptr(unsafe.Pointer(&buffer[0])),
	)
	if length == 0 {
		return ""
	}
	return strings.TrimSpace(syscall.UTF16ToString(buffer))
}

func windowsFontAliasesContain(aliases []string, family string) bool {
	normalizedFamily := normalizeFontFamilyKey(family)
	if normalizedFamily == "" {
		return false
	}
	for _, alias := range aliases {
		if normalizeFontFamilyKey(alias) == normalizedFamily {
			return true
		}
	}
	return false
}

func inferWindowsFontFileExtension(data []byte) string {
	if len(data) >= 4 && string(data[:4]) == "OTTO" {
		return ".otf"
	}
	return ".ttf"
}

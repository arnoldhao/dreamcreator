package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

const maxFontFileSizeBytes int64 = 100 << 20 // 100 MiB

type FontService struct {
	mu sync.RWMutex

	loaded  bool
	catalog fontCatalog
	err     error
}

type FontCatalogFamily struct {
	Family string
	Faces  []FontCatalogFace
}

type FontCatalogFace struct {
	Name           string
	FullName       string
	PostScriptName string
	Weight         int
	Italic         bool
}

type ExportedFontFamily struct {
	Family string
	Assets []ExportedFontAsset
}

type ExportedFontAsset struct {
	FileName string
	Content  []byte
}

type fontCatalog struct {
	families      []string
	familyCatalog []FontCatalogFamily
	filesByFamily map[string][]fontFileEntry
}

type fontFileEntry struct {
	family    string
	fileName  string
	path      string
	priority  int
	faceIndex int
}

type fontNameEntry struct {
	family     string
	face       string
	fullName   string
	postScript string
	weight     int
	italic     bool
	lookupKeys []string
	faceIndex  int
}

func NewFontService() *FontService {
	return &FontService{}
}

func (service *FontService) ListFontFamilies(ctx context.Context) ([]string, error) {
	if err := service.ensureCatalog(ctx); err != nil {
		return nil, err
	}
	service.mu.RLock()
	defer service.mu.RUnlock()
	result := make([]string, 0, len(service.catalog.families))
	result = append(result, service.catalog.families...)
	return result, nil
}

func (service *FontService) ListFontCatalog(ctx context.Context) ([]FontCatalogFamily, error) {
	if err := service.ensureCatalog(ctx); err != nil {
		return nil, err
	}
	service.mu.RLock()
	defer service.mu.RUnlock()
	result := make([]FontCatalogFamily, 0, len(service.catalog.familyCatalog))
	for _, family := range service.catalog.familyCatalog {
		result = append(result, cloneFontCatalogFamily(family))
	}
	return result, nil
}

func (service *FontService) ExportFontFamily(ctx context.Context, family string) (ExportedFontFamily, error) {
	if err := service.ensureCatalog(ctx); err != nil {
		return ExportedFontFamily{}, err
	}

	trimmedFamily := strings.TrimSpace(family)
	if trimmedFamily == "" {
		return ExportedFontFamily{Assets: []ExportedFontAsset{}}, nil
	}

	service.mu.RLock()
	entries := append([]fontFileEntry(nil), service.catalog.filesByFamily[normalizeFontFamilyKey(trimmedFamily)]...)
	service.mu.RUnlock()
	result := ExportedFontFamily{
		Family: trimmedFamily,
		Assets: make([]ExportedFontAsset, 0, len(entries)),
	}
	if len(entries) == 0 {
		return result, nil
	}

	for _, entry := range entries {
		if ctx.Err() != nil {
			return ExportedFontFamily{}, ctx.Err()
		}

		data, err := os.ReadFile(entry.path)
		if err != nil {
			continue
		}
		if size := int64(len(data)); size <= 0 || size > maxFontFileSizeBytes {
			continue
		}

		fileName := entry.fileName
		if entry.faceIndex >= 0 && isFontCollectionData(data) {
			data, err = extractFontFaceFromCollectionBytes(data, entry.faceIndex)
			if err != nil {
				continue
			}
			if size := int64(len(data)); size <= 0 || size > maxFontFileSizeBytes {
				continue
			}
		}

		result.Family = entry.family
		result.Assets = append(result.Assets, ExportedFontAsset{
			FileName: fileName,
			Content:  data,
		})
	}

	return result, nil
}

func (service *FontService) ensureCatalog(ctx context.Context) error {
	service.mu.RLock()
	if service.loaded {
		err := service.err
		service.mu.RUnlock()
		return err
	}
	service.mu.RUnlock()

	service.mu.Lock()
	defer service.mu.Unlock()
	if service.loaded {
		return service.err
	}
	service.catalog, service.err = scanFontCatalog(ctx)
	service.loaded = true
	return service.err
}

func (service *FontService) RefreshCatalog(ctx context.Context) error {
	service.mu.Lock()
	service.loaded = false
	service.catalog = fontCatalog{}
	service.err = nil
	service.mu.Unlock()
	return service.ensureCatalog(ctx)
}

func scanFontCatalog(ctx context.Context) (fontCatalog, error) {
	dirs := fontDirectories()
	if len(dirs) == 0 {
		return fontCatalog{
			families:      []string{},
			filesByFamily: map[string][]fontFileEntry{},
		}, nil
	}

	filesByFamily := make(map[string][]fontFileEntry, 512)
	facesByFamily := make(map[string][]FontCatalogFace, 512)
	displayFamilies := make(map[string]string, 512)
	seenFamilyPath := make(map[string]struct{}, 1024)
	seenFacesByFamily := make(map[string]map[string]struct{}, 512)

	for dirIndex, dir := range dirs {
		if ctx.Err() != nil {
			return fontCatalog{}, ctx.Err()
		}

		if dir == "" {
			continue
		}
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}

		err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if entry.IsDir() {
				return nil
			}

			if !isFontFile(path) {
				return nil
			}

			stat, err := entry.Info()
			if err != nil {
				return nil
			}
			if stat.Size() <= 0 || stat.Size() > maxFontFileSizeBytes {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			fontNamesInFile := fontNamesFromFontData(data)
			for _, names := range fontNamesInFile {
				family := strings.TrimSpace(names.family)
				if family == "" {
					continue
				}
				if strings.HasPrefix(family, ".") {
					continue
				}

				normalizedFamily := normalizeFontFamilyKey(family)
				if normalizedFamily != "" {
					displayFamilies[normalizedFamily] = family
					faceKey := buildFontCatalogFaceKey(names)
					if faceKey != "" {
						seenFaces := seenFacesByFamily[normalizedFamily]
						if seenFaces == nil {
							seenFaces = make(map[string]struct{}, 8)
							seenFacesByFamily[normalizedFamily] = seenFaces
						}
						if _, exists := seenFaces[faceKey]; !exists {
							seenFaces[faceKey] = struct{}{}
							facesByFamily[normalizedFamily] = append(facesByFamily[normalizedFamily], FontCatalogFace{
								Name:           firstNonEmpty(strings.TrimSpace(names.face), "Regular"),
								FullName:       strings.TrimSpace(names.fullName),
								PostScriptName: strings.TrimSpace(names.postScript),
								Weight:         names.weight,
								Italic:         names.italic,
							})
						}
					}
				}

				for _, lookupKey := range names.lookupKeys {
					lookupKey = strings.TrimSpace(lookupKey)
					if lookupKey == "" || strings.HasPrefix(lookupKey, ".") {
						continue
					}
					normalizedLookupKey := normalizeFontFamilyKey(lookupKey)
					if normalizedLookupKey == "" {
						continue
					}
					seenKey := normalizedLookupKey + "\x00" + path + "\x00" + strconv.Itoa(names.faceIndex)
					if _, exists := seenFamilyPath[seenKey]; exists {
						continue
					}
					seenFamilyPath[seenKey] = struct{}{}
					filesByFamily[normalizedLookupKey] = append(filesByFamily[normalizedLookupKey], fontFileEntry{
						family:    family,
						fileName:  buildFontAssetFileName(path, names),
						path:      path,
						priority:  dirIndex,
						faceIndex: names.faceIndex,
					})
				}

				if normalizedFamily == "" {
					continue
				}
			}

			return nil
		})
		if err != nil {
			return fontCatalog{}, err
		}
	}

	families := make([]string, 0, len(displayFamilies))
	familyCatalog := make([]FontCatalogFamily, 0, len(displayFamilies))
	for _, entries := range filesByFamily {
		if len(entries) == 0 {
			continue
		}
		sort.Slice(entries, func(left, right int) bool {
			if entries[left].priority != entries[right].priority {
				return entries[left].priority < entries[right].priority
			}
			if entries[left].fileName == entries[right].fileName {
				return entries[left].path < entries[right].path
			}
			return entries[left].fileName < entries[right].fileName
		})
	}
	for _, family := range displayFamilies {
		families = append(families, family)
	}
	sort.Strings(families)
	for _, family := range families {
		normalizedFamily := normalizeFontFamilyKey(family)
		faces := append([]FontCatalogFace(nil), facesByFamily[normalizedFamily]...)
		sort.Slice(faces, func(left, right int) bool {
			if faces[left].Weight != faces[right].Weight {
				return faces[left].Weight < faces[right].Weight
			}
			if faces[left].Italic != faces[right].Italic {
				return !faces[left].Italic && faces[right].Italic
			}
			return faces[left].Name < faces[right].Name
		})
		if len(faces) == 0 {
			faces = []FontCatalogFace{{Name: "Regular", Weight: 400}}
		}
		familyCatalog = append(familyCatalog, FontCatalogFamily{
			Family: family,
			Faces:  faces,
		})
	}
	return fontCatalog{
		families:      families,
		familyCatalog: familyCatalog,
		filesByFamily: filesByFamily,
	}, nil
}

func fontNamesFromFontData(data []byte) []fontNameEntry {
	var results []fontNameEntry

	if isFontCollectionData(data) {
		collection, err := opentype.ParseCollection(data)
		if err != nil {
			return nil
		}
		for i := 0; i < collection.NumFonts(); i++ {
			font, err := collection.Font(i)
			if err != nil {
				continue
			}
			if entry := fontNames(font, i); entry.family != "" {
				results = append(results, entry)
			}
		}
		return results
	}

	font, err := opentype.Parse(data)
	if err != nil {
		return nil
	}
	if entry := fontNames(font, -1); entry.family != "" {
		results = append(results, entry)
	}
	return results
}

func fontNames(font *sfnt.Font, faceIndex int) fontNameEntry {
	var buf sfnt.Buffer
	lookupKeys := make([]string, 0, 6)
	push := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		for _, existing := range lookupKeys {
			if strings.EqualFold(existing, name) {
				return
			}
		}
		lookupKeys = append(lookupKeys, name)
	}

	typographicFamily := readFontName(font, &buf, sfnt.NameIDTypographicFamily)
	typographicSubfamily := readFontName(font, &buf, sfnt.NameIDTypographicSubfamily)
	wwsFamily := readFontName(font, &buf, sfnt.NameIDWWSFamily)
	wwsSubfamily := readFontName(font, &buf, sfnt.NameIDWWSSubfamily)
	legacyFamily := readFontName(font, &buf, sfnt.NameIDFamily)
	legacySubfamily := readFontName(font, &buf, sfnt.NameIDSubfamily)
	fullName := readFontName(font, &buf, sfnt.NameIDFull)
	postScript := readFontName(font, &buf, sfnt.NameIDPostScript)
	family := resolveCatalogFontFamily(typographicFamily, wwsFamily, legacyFamily, fullName, postScript)
	face := resolveCatalogFontFace(typographicSubfamily, wwsSubfamily, legacySubfamily, fullName, family, postScript)
	weight := resolveCatalogFontWeight(face, fullName, postScript)
	italic := resolveCatalogFontItalic(face, fullName, postScript)

	push(family)
	push(typographicFamily)
	push(wwsFamily)
	push(legacyFamily)
	push(fullName)
	push(postScript)

	return fontNameEntry{
		family:     family,
		face:       face,
		fullName:   fullName,
		postScript: postScript,
		weight:     weight,
		italic:     italic,
		lookupKeys: lookupKeys,
		faceIndex:  faceIndex,
	}
}

func readFontName(font *sfnt.Font, buf *sfnt.Buffer, id sfnt.NameID) string {
	name, err := font.Name(buf, id)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}

func resolveCatalogFontFamily(typographicFamily, wwsFamily, legacyFamily, fullName, postScript string) string {
	// macOS ships internal faces whose legacy family names are dot-prefixed.
	// Those names are not exposed to AppKit/WebView, so do not surface the
	// matching typographic alias in settings either.
	if isHiddenCatalogFontFamily(legacyFamily) || isHiddenCatalogFontFamily(wwsFamily) {
		return ""
	}
	return firstPublicCatalogFontFamily(legacyFamily, wwsFamily, typographicFamily, fullName, postScript)
}

func firstPublicCatalogFontFamily(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || isHiddenCatalogFontFamily(trimmed) {
			continue
		}
		return trimmed
	}
	return ""
}

func isHiddenCatalogFontFamily(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), ".")
}

func isFontFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".ttf", ".otf", ".ttc", ".otc":
		return true
	default:
		return false
	}
}

func fontDirectories() []string {
	home, _ := os.UserHomeDir()

	dirs := make([]string, 0, 4)
	// Scan the operating system font directories only.
	dirs = append(dirs, platformFontDirectories(home)...)
	return dirs
}

func normalizeFontFamilyKey(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.Trim(trimmed, `"'`)
	trimmed = strings.NewReplacer("-", " ", "_", " ").Replace(trimmed)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(strings.Join(strings.Fields(trimmed), " "))
}

func buildFontAssetFileName(path string, entry fontNameEntry) string {
	ext := strings.ToLower(filepath.Ext(path))
	if entry.faceIndex < 0 {
		return filepath.Base(path)
	}

	targetExt := ".ttf"
	if ext == ".otc" {
		targetExt = ".otf"
	}

	baseName := firstNonEmpty(
		entry.postScript,
		entry.fullName,
		entry.face,
		entry.family,
		fmt.Sprintf("face-%d", entry.faceIndex),
	)
	baseName = sanitizeFontAssetFileName(baseName)
	if baseName == "" {
		baseName = fmt.Sprintf("face-%d", entry.faceIndex)
	}
	return fmt.Sprintf("%s-face-%d%s", baseName, entry.faceIndex, targetExt)
}

func sanitizeFontAssetFileName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(trimmed))
	lastDash := false
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if lastDash {
				continue
			}
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func extractFontFaceFromCollectionBytes(data []byte, faceIndex int) ([]byte, error) {
	if len(data) < 12 || string(data[:4]) != "ttcf" {
		return nil, fmt.Errorf("font data is not a collection")
	}
	fontCount := int(binary.BigEndian.Uint32(data[8:12]))
	if faceIndex < 0 || faceIndex >= fontCount {
		return nil, fmt.Errorf("font face index %d out of range", faceIndex)
	}

	offsetTableOffset := 12 + faceIndex*4
	if len(data) < offsetTableOffset+4 {
		return nil, fmt.Errorf("font collection offset table is truncated")
	}
	fontOffset := int(binary.BigEndian.Uint32(data[offsetTableOffset : offsetTableOffset+4]))
	if fontOffset < 0 || len(data) < fontOffset+12 {
		return nil, fmt.Errorf("font face offset is invalid")
	}

	numTables := int(binary.BigEndian.Uint16(data[fontOffset+4 : fontOffset+6]))
	recordOffset := fontOffset + 12
	recordTableSize := numTables * 16
	if numTables <= 0 || len(data) < recordOffset+recordTableSize {
		return nil, fmt.Errorf("font face table directory is invalid")
	}

	type tableRecord struct {
		tag      [4]byte
		checksum uint32
		offset   uint32
		length   uint32
	}

	records := make([]tableRecord, 0, numTables)
	totalSize := 12 + recordTableSize
	for i := 0; i < numTables; i++ {
		start := recordOffset + i*16
		var record tableRecord
		copy(record.tag[:], data[start:start+4])
		record.checksum = binary.BigEndian.Uint32(data[start+4 : start+8])
		record.offset = binary.BigEndian.Uint32(data[start+8 : start+12])
		record.length = binary.BigEndian.Uint32(data[start+12 : start+16])
		end := int(record.offset + record.length)
		if int(record.offset) < 0 || end < int(record.offset) || end > len(data) {
			return nil, fmt.Errorf("font face table %q is out of range", string(record.tag[:]))
		}
		totalSize += alignFontDataLength(int(record.length))
		records = append(records, record)
	}

	output := make([]byte, totalSize)
	copy(output[:12], data[fontOffset:fontOffset+12])
	writeOffset := 12 + recordTableSize
	for i, record := range records {
		start := 12 + i*16
		copy(output[start:start+4], record.tag[:])
		binary.BigEndian.PutUint32(output[start+4:start+8], record.checksum)
		binary.BigEndian.PutUint32(output[start+8:start+12], uint32(writeOffset))
		binary.BigEndian.PutUint32(output[start+12:start+16], record.length)

		sourceStart := int(record.offset)
		sourceEnd := sourceStart + int(record.length)
		copy(output[writeOffset:writeOffset+int(record.length)], data[sourceStart:sourceEnd])
		writeOffset += alignFontDataLength(int(record.length))
	}

	return output, nil
}

func alignFontDataLength(length int) int {
	if length <= 0 {
		return 0
	}
	return (length + 3) &^ 3
}

func isFontCollectionData(data []byte) bool {
	return len(data) >= 4 && bytes.Equal(data[:4], []byte("ttcf"))
}

func cloneFontCatalogFamily(value FontCatalogFamily) FontCatalogFamily {
	result := FontCatalogFamily{
		Family: value.Family,
		Faces:  make([]FontCatalogFace, 0, len(value.Faces)),
	}
	result.Faces = append(result.Faces, value.Faces...)
	return result
}

func buildFontCatalogFaceKey(entry fontNameEntry) string {
	return firstNonEmpty(
		normalizeFontFamilyKey(entry.postScript),
		normalizeFontFamilyKey(entry.fullName),
		normalizeFontFamilyKey(entry.face),
		fmt.Sprintf("face-%d", entry.faceIndex),
	)
}

func resolveCatalogFontFace(typographicSubfamily, wwsSubfamily, legacySubfamily, fullName, family, postScript string) string {
	if derived := deriveCatalogFontFaceFromPostScript(postScript); derived != "" {
		return derived
	}
	for _, value := range []string{typographicSubfamily, wwsSubfamily, legacySubfamily} {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || strings.EqualFold(trimmed, family) {
			continue
		}
		return trimmed
	}
	if derived := deriveCatalogFontFaceFromFullName(fullName, family); derived != "" {
		return derived
	}
	return "Regular"
}

func deriveCatalogFontFaceFromFullName(fullName string, family string) string {
	trimmedFullName := strings.TrimSpace(fullName)
	trimmedFamily := strings.TrimSpace(family)
	if trimmedFullName == "" {
		return ""
	}
	if trimmedFamily != "" && strings.HasPrefix(strings.ToLower(trimmedFullName), strings.ToLower(trimmedFamily)) {
		trimmedFullName = strings.TrimSpace(strings.TrimLeft(trimmedFullName[len(trimmedFamily):], " -_"))
	}
	if trimmedFullName == "" || strings.EqualFold(trimmedFullName, family) {
		return ""
	}
	return trimmedFullName
}

func deriveCatalogFontFaceFromPostScript(postScript string) string {
	trimmed := strings.TrimSpace(postScript)
	if trimmed == "" {
		return ""
	}
	if separatorIndex := strings.LastIndex(trimmed, "-"); separatorIndex >= 0 && separatorIndex < len(trimmed)-1 {
		return strings.TrimSpace(trimmed[separatorIndex+1:])
	}
	return ""
}

func resolveCatalogFontWeight(face, fullName, postScript string) int {
	return deriveCatalogFontWeight(strings.ToLower(strings.TrimSpace(strings.Join([]string{face, fullName, postScript}, " "))))
}

func resolveCatalogFontItalic(face, fullName, postScript string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.Join([]string{face, fullName, postScript}, " ")))
	return strings.Contains(normalized, "italic") || strings.Contains(normalized, "oblique")
}

func deriveCatalogFontWeight(normalized string) int {
	switch {
	case strings.Contains(normalized, "thin"), strings.Contains(normalized, "hairline"):
		return 100
	case strings.Contains(normalized, "ultralight"), strings.Contains(normalized, "extra light"), strings.Contains(normalized, "extralight"):
		return 200
	case strings.Contains(normalized, "light"):
		return 300
	case strings.Contains(normalized, "semibold"), strings.Contains(normalized, "demibold"), strings.Contains(normalized, "demi bold"), strings.Contains(normalized, "demi"):
		return 600
	case strings.Contains(normalized, "extrabold"), strings.Contains(normalized, "extra bold"), strings.Contains(normalized, "ultrabold"), strings.Contains(normalized, "ultra bold"):
		return 800
	case strings.Contains(normalized, "black"), strings.Contains(normalized, "heavy"):
		return 900
	case strings.Contains(normalized, "medium"):
		return 500
	case strings.Contains(normalized, "bold"):
		return 700
	default:
		return 400
	}
}

// platformFontDirectories returns candidate font directories for current OS.
// Implemented in platform-specific files.

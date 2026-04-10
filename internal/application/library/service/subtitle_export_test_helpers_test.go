package service

import (
	"encoding/xml"
	"strings"
	"testing"
)

func parseRenderedFCPXML(t *testing.T, content string) fcpxmlRoot {
	t.Helper()

	var root fcpxmlRoot
	if err := xml.Unmarshal([]byte(content), &root); err != nil {
		t.Fatalf("unmarshal fcpxml: %v\ncontent=%s", err, content)
	}
	return root
}

func requireASSSectionValue(t *testing.T, content string, section string, key string) string {
	t.Helper()

	lines := strings.Split(normalizeSubtitleNewlines(content), "\n")
	inSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inSection = strings.EqualFold(trimmed, section)
			continue
		}
		if !inSection {
			continue
		}
		foundKey, value, ok := splitSubtitleExportStyleKeyValue(trimmed)
		if ok && strings.EqualFold(foundKey, key) {
			return strings.TrimSpace(value)
		}
	}

	t.Fatalf("missing key %q in section %q\ncontent=%s", key, section, content)
	return ""
}

func requireASSStyleField(t *testing.T, content string, section string, field string) string {
	t.Helper()

	lines := strings.Split(normalizeSubtitleNewlines(content), "\n")
	inSection := false
	format := []string(nil)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inSection = strings.EqualFold(trimmed, section)
			continue
		}
		if !inSection {
			continue
		}
		key, value, ok := splitSubtitleExportStyleKeyValue(trimmed)
		if !ok {
			continue
		}
		if strings.EqualFold(key, "Format") {
			format = parseSubtitleExportStyleFormat(value)
			continue
		}
		if strings.EqualFold(key, "Style") {
			values := splitSubtitleExportStyleFields(value, len(format))
			return findSubtitleExportStyleField(format, values, field)
		}
	}

	t.Fatalf("missing style field %q in section %q\ncontent=%s", field, section, content)
	return ""
}

func parseRenderedITTAttrs(t *testing.T, content string) ([]xml.Attr, []xml.Attr) {
	t.Helper()

	decoder := xml.NewDecoder(strings.NewReader(content))
	var rootAttrs []xml.Attr
	var styleAttrs []xml.Attr
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		startElement, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if startElement.Name.Local == "tt" && rootAttrs == nil {
			rootAttrs = append([]xml.Attr(nil), startElement.Attr...)
		}
		if startElement.Name.Local == "style" && styleAttrs == nil {
			styleAttrs = append([]xml.Attr(nil), startElement.Attr...)
		}
		if rootAttrs != nil && styleAttrs != nil {
			return rootAttrs, styleAttrs
		}
	}

	t.Fatalf("expected rendered itt xml to contain root and style attrs\ncontent=%s", content)
	return nil, nil
}

func requireFCPXMLTimeOnFrameBoundary(t *testing.T, frameDuration string, value string) int64 {
	t.Helper()

	timeNumerator, timeDenominator, ok := parseFCPXMLTimeValue(value)
	if !ok {
		t.Fatalf("invalid fcpxml time value %q", value)
	}
	frameNumerator, frameDenominator, ok := parseFCPXMLRationalTime(frameDuration)
	if !ok {
		t.Fatalf("invalid fcpxml frame duration %q", frameDuration)
	}
	framesNumerator := timeNumerator * frameDenominator
	framesDenominator := timeDenominator * frameNumerator
	if framesDenominator <= 0 || framesNumerator%framesDenominator != 0 {
		t.Fatalf("expected %q to align to frame duration %q", value, frameDuration)
	}
	return framesNumerator / framesDenominator
}

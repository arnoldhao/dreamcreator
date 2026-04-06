package service

import (
	"strings"
	"testing"

	"dreamcreator/internal/application/library/dto"
)

func TestDetectSubtitleFormatNormalizesAliases(t *testing.T) {
	t.Parallel()

	if got := detectSubtitleFormat("", "track.webvtt", ""); got != "vtt" {
		t.Fatalf("expected webvtt alias to normalize to vtt, got %q", got)
	}
	if got := detectSubtitleFormat("", "track.dfxp", ""); got != "itt" {
		t.Fatalf("expected dfxp alias to normalize to itt, got %q", got)
	}
	if got := detectSubtitleFormat("", "track.itt", ""); got != "itt" {
		t.Fatalf("expected itt extension to stay itt, got %q", got)
	}
}

func TestParseSubtitleDocumentVTT(t *testing.T) {
	t.Parallel()

	content := "WEBVTT Demo\n\nSTYLE\n::cue { color: white; }\n\nNOTE top level note\nkeep this block\n\ncue-1\n00:00:01.000 --> 00:00:02.500 line:90% position:50%\nHello\nWorld\n"
	document := parseSubtitleDocument(content, "webvtt")

	if document.Format != "vtt" {
		t.Fatalf("expected normalized vtt format, got %#v", document)
	}
	if len(document.Cues) != 1 {
		t.Fatalf("expected one cue, got %#v", document.Cues)
	}
	if document.Cues[0].Start != "00:00:01.000" || document.Cues[0].End != "00:00:02.500" {
		t.Fatalf("unexpected vtt timing, got %#v", document.Cues[0])
	}
	if document.Cues[0].Text != "Hello\nWorld" {
		t.Fatalf("expected multiline cue text, got %#v", document.Cues[0])
	}
	if subtitleDocumentSourceContent(document) != content {
		t.Fatalf("expected source content metadata to be preserved")
	}
}

func TestParseSubtitleDocumentASSIgnoresComments(t *testing.T) {
	t.Parallel()

	content := `[Script Info]
ScriptType: v4.00+

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Comment: 0,0:00:00.00,0:00:01.00,Default,,0,0,0,,comment
Dialogue: 0,0:00:01.00,0:00:02.50,Default,,0,0,0,,Hello\NWorld
`
	document := parseSubtitleDocument(content, "ass")

	if len(document.Cues) != 1 {
		t.Fatalf("expected one dialogue cue, got %#v", document.Cues)
	}
	if document.Cues[0].Text != "Hello\nWorld" {
		t.Fatalf("expected escaped ass newlines to normalize, got %#v", document.Cues[0])
	}
}

func TestParseSubtitleDocumentITTAndFCPXML(t *testing.T) {
	t.Parallel()

	itt := `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <head><styling><style xml:id="s1" /></styling></head>
  <body><div><p begin="00:00:01.000" end="00:00:02.000">Hello<br/>TTML</p></div></body>
</tt>`
	ittDocument := parseSubtitleDocument(itt, "ttml")
	if ittDocument.Format != "itt" || len(ittDocument.Cues) != 1 {
		t.Fatalf("expected itt cue parse, got %#v", ittDocument)
	}
	if ittDocument.Cues[0].Text != "Hello\nTTML" {
		t.Fatalf("expected ttml line breaks to normalize, got %#v", ittDocument.Cues[0])
	}

	fcpxml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>
<fcpxml version="1.11">
  <resources>
    <format id="r1" frameDuration="1/30s" width="1920" height="1080"/>
    <effect id="r2" name="Basic Title" uid="basic-title"/>
  </resources>
  <library location="file:///tmp/demo.fcpbundle">
    <event name="Demo">
      <project name="Project">
        <sequence duration="3s" format="r1">
          <spine>
            <gap offset="0s" start="3600s" duration="3s">
              <title ref="r2" lane="1" offset="1s" duration="3/2s" start="3600s">
                <text><text-style ref="ts1">Hello FCPXML</text-style></text>
                <text-style-def id="ts1"><text-style font="Helvetica"/></text-style-def>
              </title>
            </gap>
          </spine>
        </sequence>
      </project>
    </event>
  </library>
</fcpxml>`
	fcpxmlDocument := parseSubtitleDocument(fcpxml, "fcpxml")
	if len(fcpxmlDocument.Cues) != 1 {
		t.Fatalf("expected one fcpxml title cue, got %#v", fcpxmlDocument.Cues)
	}
	if fcpxmlDocument.Cues[0].Start != "00:00:01.000" || fcpxmlDocument.Cues[0].End != "00:00:02.500" {
		t.Fatalf("expected fcpxml rational timing to normalize, got %#v", fcpxmlDocument.Cues[0])
	}
}

func TestRenderSubtitleContentPreservingSourceVTTAndASS(t *testing.T) {
	t.Parallel()

	vttSource := "WEBVTT\n\ncue-1\n00:00:01.000 --> 00:00:02.000 line:80%\nHello\n"
	vttDocument := subtitleDocumentWithSource(vttSource, "vtt", []dto.SubtitleCue{{
		Index: 1,
		Start: "00:00:03.000",
		End:   "00:00:04.000",
		Text:  "Updated",
	}}, nil)
	vttContent, ok := renderSubtitleContentPreservingSource(vttDocument, "vtt", nil, "", vttSource)
	if !ok {
		t.Fatal("expected vtt source-preserving render to succeed")
	}
	if !strings.Contains(vttContent, "00:00:03.000 --> 00:00:04.000 line:80%") {
		t.Fatalf("expected vtt cue settings to be preserved, got %q", vttContent)
	}

	assSource := `[Script Info]
ScriptType: v4.00+

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Dialogue: 0,0:00:01.00,0:00:02.00,Primary,,12,34,56,fade,Hello\NWorld
`
	assDocument := subtitleDocumentWithSource(assSource, "ass", []dto.SubtitleCue{{
		Index: 1,
		Start: "0:00:05.00",
		End:   "0:00:06.00",
		Text:  "Updated",
	}}, nil)
	assContent, ok := renderSubtitleContentPreservingSource(assDocument, "ass", nil, "", assSource)
	if !ok {
		t.Fatal("expected ass source-preserving render to succeed")
	}
	if !strings.Contains(assContent, "Dialogue: 0,0:00:05.00,0:00:06.00,Primary,,12,34,56,fade,Updated") {
		t.Fatalf("expected ass style and margins to be preserved, got %q", assContent)
	}
}

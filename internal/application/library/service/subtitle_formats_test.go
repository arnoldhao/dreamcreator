package service

import (
	"encoding/xml"
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

	ittSMPTE := `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttp="http://www.w3.org/ns/ttml#parameter" ttp:timeBase="smpte" ttp:frameRate="30" ttp:frameRateMultiplier="1000 1001">
  <body><div><p begin="00:00:30:22" end="00:00:34:22">SMPTE cue</p></div></body>
</tt>`
	ittSMPTEDocument := parseSubtitleDocument(ittSMPTE, "itt")
	if ittSMPTEDocument.Format != "itt" || len(ittSMPTEDocument.Cues) != 1 {
		t.Fatalf("expected itt smpte cue parse, got %#v", ittSMPTEDocument)
	}
	if ittSMPTEDocument.Cues[0].Start != "00:00:30.764" || ittSMPTEDocument.Cues[0].End != "00:00:34.768" {
		t.Fatalf("expected smpte itt timing to normalize into media timestamps, got %#v", ittSMPTEDocument.Cues[0])
	}
	if ittSMPTEDocument.Cues[0].Text != "SMPTE cue" {
		t.Fatalf("expected smpte itt cue text to be preserved, got %#v", ittSMPTEDocument.Cues[0])
	}

	fcpxml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>
<fcpxml version="1.11">
  <resources>
    <format id="r1" frameDuration="1001/30000s" width="1920" height="1080"/>
    <effect id="r2" name="Basic Title" uid="basic-title"/>
  </resources>
  <library location="file:///tmp/demo.fcpbundle">
    <event name="Demo">
      <project name="Project">
        <sequence duration="3s" format="r1">
          <spine>
            <gap offset="0s" start="0s" duration="3s">
              <title ref="r2" lane="1" offset="1001/1000s" duration="3003/2000s" start="1001/1000s">
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
	if fcpxmlDocument.Cues[0].Start != "00:00:01.001" || fcpxmlDocument.Cues[0].End != "00:00:02.503" {
		t.Fatalf("expected fcpxml rational timing to normalize, got %#v", fcpxmlDocument.Cues[0])
	}
}

func TestRenderSubtitleContentPreservingSourceFCPXML(t *testing.T) {
	t.Parallel()

	source := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>
<fcpxml version="1.11">
  <resources>
    <format id="r1" frameDuration="1/30s" width="1920" height="1080" colorSpace="1-1-1 (Rec. 709)"/>
    <effect id="r2" name="Basic Title" uid="basic-title"/>
  </resources>
  <library location="file:///tmp/demo.fcpbundle">
    <event name="Demo">
      <project name="Project">
        <sequence duration="4s" format="r1">
          <spine>
            <gap offset="0s" start="3600s" duration="4s">
              <title ref="r2" lane="3" offset="1s" duration="3/2s" start="3600s" name="Hello FCPXML">
                <text><text-style ref="ts-main">Hello FCPXML</text-style></text>
                <text-style-def id="ts-main"><text-style font="Helvetica" fontSize="60" alignment="center"/></text-style-def>
              </title>
              <title ref="r2" lane="4" offset="3s" duration="1s" start="3600s" name="Second">
                <text><text-style ref="ts-secondary">Second</text-style></text>
              </title>
            </gap>
          </spine>
        </sequence>
      </project>
    </event>
  </library>
</fcpxml>`

	document := subtitleDocumentWithSource(source, "fcpxml", []dto.SubtitleCue{
		{Index: 1, Start: "00:00:05.000", End: "00:00:06.500", Text: "Updated title"},
		{Index: 2, Start: "00:00:07.000", End: "00:00:08.000", Text: "Second updated"},
	}, nil)

	content, ok := renderSubtitleContentPreservingSource(document, "fcpxml", nil, "", source)
	if !ok {
		t.Fatal("expected fcpxml source-preserving render to succeed")
	}

	var root fcpxmlRoot
	if err := xml.Unmarshal([]byte(content), &root); err != nil {
		t.Fatalf("unmarshal rendered fcpxml: %v\ncontent=%s", err, content)
	}

	frameGrid := newFCPXMLFrameGrid("1/30s")
	if len(root.Resources.Formats) != 1 || root.Resources.Formats[0].FrameDuration != frameGrid.formatFrames(1) {
		t.Fatalf("expected original format resource to normalize onto shared timebase, got %#v", root.Resources)
	}
	if len(root.Library.Events) != 1 || len(root.Library.Events[0].Projects) != 1 {
		t.Fatalf("expected original event/project structure to be preserved, got %#v", root.Library)
	}

	sequence := root.Library.Events[0].Projects[0].Sequence
	if sequence.Duration != frameGrid.formatFrames(frameGrid.roundMillisecondsToFrames(8000)) {
		t.Fatalf("expected sequence duration to follow the last cue end, got %#v", sequence)
	}
	gap := sequence.Spine.Gap
	if gap.Start != "3600s" || gap.Duration != frameGrid.formatFrames(frameGrid.roundMillisecondsToFrames(8000)) {
		t.Fatalf("expected gap timing to stay aligned with source metadata, got %#v", gap)
	}
	if len(gap.Titles) != 2 {
		t.Fatalf("expected two preserved fcpxml titles, got %#v", gap.Titles)
	}

	first := gap.Titles[0]
	if first.Lane != 3 || first.Ref != "r2" || first.Offset != frameGrid.formatFrames(frameGrid.roundMillisecondsToFrames(5000)) || first.Duration != frameGrid.formatFrames(frameGrid.roundMillisecondsToFrames(1500)) || first.Start != "3600s" {
		t.Fatalf("expected first title metadata to be preserved while timing updates, got %#v", first)
	}
	if fcpxmlTitleText(first) != "Updated title" {
		t.Fatalf("expected first title text to update, got %#v", first)
	}
	if len(first.TextStyleDef) != 1 || first.TextStyleDef[0].ID != "ts-main" {
		t.Fatalf("expected first title text-style-def to be preserved, got %#v", first)
	}
	if first.Text == nil || len(first.Text.TextStyle) != 1 || first.Text.TextStyle[0].Ref != "ts-main" {
		t.Fatalf("expected first title text-style ref to stay intact, got %#v", first)
	}

	second := gap.Titles[1]
	if second.Lane != 4 || second.Offset != frameGrid.formatFrames(frameGrid.roundMillisecondsToFrames(7000)) || second.Duration != frameGrid.formatFrames(frameGrid.roundMillisecondsToFrames(1000)) || second.Start != "3600s" {
		t.Fatalf("expected second title metadata to be preserved while timing updates, got %#v", second)
	}
	if fcpxmlTitleText(second) != "Second updated" {
		t.Fatalf("expected second title text to update, got %#v", second)
	}
	if second.Text == nil || len(second.Text.TextStyle) != 1 || second.Text.TextStyle[0].Ref != "ts-secondary" {
		t.Fatalf("expected second title text-style ref to stay intact, got %#v", second)
	}

	roundTrip := parseSubtitleDocument(content, "fcpxml")
	if len(roundTrip.Cues) != 2 {
		t.Fatalf("expected round-trip fcpxml parse to keep both cues, got %#v", roundTrip.Cues)
	}
	if roundTrip.Cues[0].Start != "00:00:05.000" || roundTrip.Cues[0].End != "00:00:06.500" || roundTrip.Cues[0].Text != "Updated title" {
		t.Fatalf("expected first round-trip cue to stay accurate, got %#v", roundTrip.Cues[0])
	}
	if roundTrip.Cues[1].Start != "00:00:07.000" || roundTrip.Cues[1].End != "00:00:08.000" || roundTrip.Cues[1].Text != "Second updated" {
		t.Fatalf("expected second round-trip cue to stay accurate, got %#v", roundTrip.Cues[1])
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

	ittSource := `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xml:lang="ja-JP" xmlns:ttp="http://www.w3.org/ns/ttml#parameter" ttp:frameRate="25">
  <head>
    <styling>
      <style xml:id="s77" tts:fontFamily="Hiragino Sans" xmlns:tts="http://www.w3.org/ns/ttml#styling"/>
    </styling>
  </head>
  <body style="s77">
    <div region="bottom">
      <p begin="00:00:01.000" end="00:00:02.000">こんにちは</p>
    </div>
  </body>
</tt>`
	ittDocument := subtitleDocumentWithSource(ittSource, "itt", []dto.SubtitleCue{{
		Index: 1,
		Start: "00:00:05.000",
		End:   "00:00:06.250",
		Text:  "更新\nしました",
	}}, nil)
	ittContent, ok := renderSubtitleContentPreservingSource(ittDocument, "itt", nil, "", ittSource)
	if !ok {
		t.Fatal("expected itt source-preserving render to succeed")
	}
	rootAttrs, styleAttrs := parseRenderedITTAttrs(t, ittContent)
	if got := xmlAttributeValue(rootAttrs, "lang"); got != "ja-JP" {
		t.Fatalf("expected itt root language to be preserved, got %q", got)
	}
	if got := xmlAttributeValue(rootAttrs, "frameRate"); got != "25" {
		t.Fatalf("expected itt frame rate to be preserved, got %q", got)
	}
	if got := xmlAttributeValue(styleAttrs, "id"); got != "s77" {
		t.Fatalf("expected itt style id to be preserved, got %q", got)
	}
	if got := xmlAttributeValue(styleAttrs, "fontFamily"); got != "Hiragino Sans" {
		t.Fatalf("expected itt style font to be preserved, got %q", got)
	}
	ittParagraphs := parseITTCueParagraphs(ittContent)
	if len(ittParagraphs) != 1 || ittParagraphs[0].Start != "00:00:05.000" || ittParagraphs[0].End != "00:00:06.250" || ittParagraphs[0].Text != "更新\nしました" {
		t.Fatalf("expected itt cue timing/text to update while preserving structure, got %#v", ittParagraphs)
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

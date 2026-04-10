package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dreamcreator/internal/application/library/dto"
)

func TestLibraryServiceExportSubtitleWritesConfiguredFormats(t *testing.T) {
	t.Parallel()

	service := &LibraryService{}
	sourceContent := "1\n00:00:01,000 --> 00:00:02,500\nLine A\nLine B\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond\n"
	cases := []struct {
		name     string
		filename string
		config   *dto.SubtitleExportConfig
		validate func(t *testing.T, content string)
	}{
		{
			name:     "ass",
			filename: "configured.ass",
			config: &dto.SubtitleExportConfig{
				ASS: &dto.SubtitleASSExportConfig{
					PlayResX: 1280,
					PlayResY: 720,
					Title:    "Service ASS",
				},
			},
			validate: func(t *testing.T, content string) {
				t.Helper()

				if got := requireASSSectionValue(t, content, "[Script Info]", "Title"); got != "Service ASS" {
					t.Fatalf("expected ass title from export config, got %q", got)
				}
				if got := requireASSSectionValue(t, content, "[Script Info]", "PlayResX"); got != "1280" {
					t.Fatalf("expected ass playResX from export config, got %q", got)
				}
				parsed := parseASSDocument(content)
				if len(parsed.Events) != 2 {
					t.Fatalf("expected two ass dialogue events, got %#v", parsed.Events)
				}
				if got := assEventValue(parsed.EventFormat, parsed.Events[0].Values, "text"); got != `Line A\NLine B` {
					t.Fatalf("expected ass multiline text to export with \\N, got %q", got)
				}
				if got := assEventValue(parsed.EventFormat, parsed.Events[1].Values, "start"); got != "0:00:03.00" {
					t.Fatalf("expected second ass cue start, got %q", got)
				}
			},
		},
		{
			name:     "itt",
			filename: "configured.itt",
			config: &dto.SubtitleExportConfig{
				ITT: &dto.SubtitleITTExportConfig{
					FrameRate:           24,
					FrameRateMultiplier: "1000 1001",
					Language:            "en-GB",
				},
			},
			validate: func(t *testing.T, content string) {
				t.Helper()

				rootAttrs, styleAttrs := parseRenderedITTAttrs(t, content)
				if got := xmlAttributeValue(rootAttrs, "lang"); got != "en-GB" {
					t.Fatalf("expected itt language from export config, got %q", got)
				}
				if got := xmlAttributeValue(rootAttrs, "frameRate"); got != "24" {
					t.Fatalf("expected itt frameRate from export config, got %q", got)
				}
				if got := xmlAttributeValue(rootAttrs, "frameRateMultiplier"); got != "1000 1001" {
					t.Fatalf("expected itt frameRateMultiplier from export config, got %q", got)
				}
				if got := xmlAttributeValue(styleAttrs, "fontFamily"); got != "Helvetica" {
					t.Fatalf("expected itt style to inherit subtitle font, got %q", got)
				}
				paragraphs := parseITTCueParagraphs(content)
				if len(paragraphs) != 2 {
					t.Fatalf("expected two itt paragraphs, got %#v", paragraphs)
				}
				if paragraphs[0].Text != "Line A\nLine B" || paragraphs[1].Text != "Second" {
					t.Fatalf("expected itt text content to round-trip, got %#v", paragraphs)
				}
			},
		},
		{
			name:     "fcpxml",
			filename: "configured.fcpxml",
			config: &dto.SubtitleExportConfig{
				FCPXML: &dto.SubtitleFCPXMLExportConfig{
					FrameDuration:        "1/24s",
					Width:                1920,
					Height:               1080,
					ColorSpace:           "Rec. 709",
					LibraryName:          "Service/Library",
					EventName:            "ServiceEvent",
					ProjectName:          "ServiceProject",
					DefaultLane:          5,
					StartTimecodeSeconds: 5400,
				},
			},
			validate: func(t *testing.T, content string) {
				t.Helper()

				root := parseRenderedFCPXML(t, content)
				if root.Library.Location != "file:///root/Movies/Service_Library.fcpbundle" {
					t.Fatalf("expected sanitized library path, got %#v", root.Library)
				}
				if len(root.Library.Events) != 1 || root.Library.Events[0].Name != "ServiceEvent" {
					t.Fatalf("expected fcpxml event metadata from export config, got %#v", root.Library)
				}
				project := root.Library.Events[0].Projects[0]
				if project.Name != "ServiceProject" {
					t.Fatalf("expected fcpxml project name from export config, got %#v", project)
				}
				gap := project.Sequence.Spine.Gap
				if len(gap.Titles) != 2 {
					t.Fatalf("expected two fcpxml title nodes, got %#v", gap.Titles)
				}
				frameGrid := newFCPXMLFrameGrid("1/24s")
				if root.Resources.Formats[0].FrameDuration != frameGrid.formatFrames(1) {
					t.Fatalf("expected fcpxml frame duration to use unified timebase, got %#v", root.Resources.Formats[0])
				}
				startFrames, _ := frameGrid.roundMillisecondsRangeToFrames(1000, 2500)
				if project.Sequence.TCStart != frameGrid.formatFrames(frameGrid.roundMillisecondsToFrames(5400*1000)) {
					t.Fatalf("expected fcpxml sequence tcStart from export config, got %#v", project.Sequence)
				}
				if gap.Start != "0s" {
					t.Fatalf("expected fcpxml gap start to use local timeline origin, got %#v", gap)
				}
				if gap.Titles[0].Lane != 5 || gap.Titles[0].Offset != frameGrid.formatFrames(startFrames) || gap.Titles[0].Start != frameGrid.formatFrames(startFrames) {
					t.Fatalf("expected fcpxml timing/lane metadata from export config, got %#v", gap.Titles[0])
				}
				if len(gap.Titles[0].Params) != 4 || gap.Titles[0].Params[0].Name != "Position" {
					t.Fatalf("expected fcpxml basic title params, got %#v", gap.Titles[0].Params)
				}
				if fcpxmlTitleText(gap.Titles[0]) != "Line A\nLine B" {
					t.Fatalf("expected fcpxml multiline cue text, got %#v", gap.Titles[0])
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			exportPath := filepath.Join(t.TempDir(), tc.filename)
			result, err := service.ExportSubtitle(context.Background(), dto.SubtitleExportRequest{
				ExportPath:           exportPath,
				Content:              sourceContent,
				Format:               "srt",
				StyleDocumentContent: testSubtitleExportStyleDocument,
				ExportConfig:         tc.config,
			})
			if err != nil {
				t.Fatalf("export subtitle: %v", err)
			}
			if result.ExportPath != exportPath {
				t.Fatalf("expected export path %q, got %#v", exportPath, result)
			}

			data, err := os.ReadFile(exportPath)
			if err != nil {
				t.Fatalf("read exported file: %v", err)
			}
			content := string(data)
			if result.Bytes != len(content) {
				t.Fatalf("expected byte count %d, got %#v", len(content), result)
			}

			tc.validate(t, content)
		})
	}
}

func TestLibraryServiceExportSubtitleAppliesStyleOverlayToSameFormatVTT(t *testing.T) {
	t.Parallel()

	service := &LibraryService{}
	exportPath := filepath.Join(t.TempDir(), "styled.vtt")
	document := subtitleDocumentWithSource(`WEBVTT

00:00:01.000 --> 00:00:02.000
Original
`, "vtt", []dto.SubtitleCue{{
		Index: 1,
		Start: "00:00:01.000",
		End:   "00:00:02.000",
		Text:  "Styled",
	}}, nil)

	result, err := service.ExportSubtitle(context.Background(), dto.SubtitleExportRequest{
		ExportPath:           exportPath,
		Document:             &document,
		StyleDocumentContent: testSubtitleExportStyleDocument,
	})
	if err != nil {
		t.Fatalf("export subtitle: %v", err)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("read exported file: %v", err)
	}
	content := string(data)
	if result.Bytes != len(content) {
		t.Fatalf("expected byte count %d, got %#v", len(content), result)
	}
	if !strings.Contains(content, "STYLE") || !strings.Contains(content, "::cue(.mono)") {
		t.Fatalf("expected same-format vtt export to re-render with style overlay, got %q", content)
	}
	if !strings.Contains(content, "<c.mono>Styled</c>") {
		t.Fatalf("expected styled cue text, got %q", content)
	}
}

func TestLibraryServiceExportSubtitlePreservesSourceStructuredFormats(t *testing.T) {
	t.Parallel()

	service := &LibraryService{}
	cases := []struct {
		name     string
		filename string
		document dto.SubtitleDocument
		validate func(t *testing.T, content string)
	}{
		{
			name:     "ass",
			filename: "preserved.ass",
			document: subtitleDocumentWithSource(`[Script Info]
Title: Original ASS
ScriptType: v4.00+

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Dialogue: 0,0:00:01.00,0:00:02.00,Primary,,12,34,56,fade,Hello\NWorld
`, "ass", []dto.SubtitleCue{{
				Index: 1,
				Start: "0:00:05.00",
				End:   "0:00:06.50",
				Text:  "Updated",
			}}, nil),
			validate: func(t *testing.T, content string) {
				t.Helper()

				parsed := parseASSDocument(content)
				if len(parsed.Events) != 1 {
					t.Fatalf("expected one preserved ass event, got %#v", parsed.Events)
				}
				entry := parsed.Events[0]
				if got := assEventValue(parsed.EventFormat, entry.Values, "style"); got != "Primary" {
					t.Fatalf("expected ass style to be preserved, got %q", got)
				}
				if got := assEventValue(parsed.EventFormat, entry.Values, "marginl"); got != "12" {
					t.Fatalf("expected ass left margin to be preserved, got %q", got)
				}
				if got := assEventValue(parsed.EventFormat, entry.Values, "marginr"); got != "34" {
					t.Fatalf("expected ass right margin to be preserved, got %q", got)
				}
				if got := assEventValue(parsed.EventFormat, entry.Values, "marginv"); got != "56" {
					t.Fatalf("expected ass vertical margin to be preserved, got %q", got)
				}
				if got := assEventValue(parsed.EventFormat, entry.Values, "effect"); got != "fade" {
					t.Fatalf("expected ass effect to be preserved, got %q", got)
				}
				if got := assEventValue(parsed.EventFormat, entry.Values, "start"); got != "0:00:05.00" {
					t.Fatalf("expected ass start to update, got %q", got)
				}
				if got := assEventValue(parsed.EventFormat, entry.Values, "end"); got != "0:00:06.50" {
					t.Fatalf("expected ass end to update, got %q", got)
				}
				if got := assEventValue(parsed.EventFormat, entry.Values, "text"); got != "Updated" {
					t.Fatalf("expected ass text to update, got %q", got)
				}
			},
		},
		{
			name:     "itt",
			filename: "preserved.itt",
			document: subtitleDocumentWithSource(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xml:lang="ko-KR" xmlns:ttp="http://www.w3.org/ns/ttml#parameter" ttp:frameRate="25">
  <head>
    <styling>
      <style xml:id="svc-style" xmlns:tts="http://www.w3.org/ns/ttml#styling" tts:fontFamily="Apple SD Gothic Neo"/>
    </styling>
  </head>
  <body style="svc-style">
    <div region="bottom">
      <p begin="00:00:01.000" end="00:00:02.000">안녕하세요</p>
    </div>
  </body>
</tt>`, "itt", []dto.SubtitleCue{{
				Index: 1,
				Start: "00:00:05.000",
				End:   "00:00:06.250",
				Text:  "업데이트",
			}}, nil),
			validate: func(t *testing.T, content string) {
				t.Helper()

				rootAttrs, styleAttrs := parseRenderedITTAttrs(t, content)
				if got := xmlAttributeValue(rootAttrs, "lang"); got != "ko-KR" {
					t.Fatalf("expected itt language to be preserved, got %q", got)
				}
				if got := xmlAttributeValue(rootAttrs, "frameRate"); got != "25" {
					t.Fatalf("expected itt frame rate to be preserved, got %q", got)
				}
				if got := xmlAttributeValue(styleAttrs, "id"); got != "svc-style" {
					t.Fatalf("expected itt style id to be preserved, got %q", got)
				}
				if got := xmlAttributeValue(styleAttrs, "fontFamily"); got != "Apple SD Gothic Neo" {
					t.Fatalf("expected itt style font to be preserved, got %q", got)
				}
				paragraphs := parseITTCueParagraphs(content)
				if len(paragraphs) != 1 || paragraphs[0].Start != "00:00:05.000" || paragraphs[0].End != "00:00:06.250" || paragraphs[0].Text != "업데이트" {
					t.Fatalf("expected itt cue timing/text to update, got %#v", paragraphs)
				}
			},
		},
		{
			name:     "fcpxml",
			filename: "preserved.fcpxml",
			document: subtitleDocumentWithSource(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>
<fcpxml version="1.11">
  <resources>
    <format id="r1" frameDuration="1/30s" width="1920" height="1080"/>
    <effect id="r2" name="Basic Title" uid="basic-title"/>
  </resources>
  <library location="file:///tmp/preserved.fcpbundle">
    <event name="PreservedEvent">
      <project name="PreservedProject">
        <sequence duration="4s" format="r1">
          <spine>
            <gap offset="0s" start="3600s" duration="4s">
              <title ref="r2" lane="7" offset="1s" duration="3/2s" start="3600s" name="First">
                <text><text-style ref="ts1">First</text-style></text>
                <text-style-def id="ts1"><text-style font="Helvetica"/></text-style-def>
              </title>
            </gap>
          </spine>
        </sequence>
      </project>
    </event>
  </library>
</fcpxml>`, "fcpxml", []dto.SubtitleCue{{
				Index: 1,
				Start: "00:00:05.000",
				End:   "00:00:06.500",
				Text:  "Updated fcpxml",
			}}, nil),
			validate: func(t *testing.T, content string) {
				t.Helper()

				root := parseRenderedFCPXML(t, content)
				if root.Library.Location != "file:///tmp/preserved.fcpbundle" {
					t.Fatalf("expected fcpxml library path to be preserved, got %#v", root.Library)
				}
				if len(root.Resources.Formats) != 1 || root.Resources.Formats[0].FrameDuration != newFCPXMLFrameGrid("1/30s").formatFrames(1) {
					t.Fatalf("expected preserved fcpxml format frame duration to normalize onto shared timebase, got %#v", root.Resources)
				}
				project := root.Library.Events[0].Projects[0]
				if project.Name != "PreservedProject" {
					t.Fatalf("expected fcpxml project name to be preserved, got %#v", project)
				}
				frameGrid := newFCPXMLFrameGrid("1/30s")
				gap := project.Sequence.Spine.Gap
				if gap.Duration != frameGrid.formatFrames(frameGrid.roundMillisecondsToFrames(6500)) {
					t.Fatalf("expected fcpxml gap duration to update from cue end, got %#v", gap)
				}
				if len(gap.Titles) != 1 {
					t.Fatalf("expected one preserved fcpxml title, got %#v", gap.Titles)
				}
				title := gap.Titles[0]
				if title.Lane != 7 || title.Ref != "r2" || title.Start != "3600s" || title.Offset != frameGrid.formatFrames(frameGrid.roundMillisecondsToFrames(5000)) || title.Duration != frameGrid.formatFrames(frameGrid.roundMillisecondsToFrames(1500)) {
					t.Fatalf("expected fcpxml title metadata to be preserved while timing updates, got %#v", title)
				}
				if fcpxmlTitleText(title) != "Updated fcpxml" {
					t.Fatalf("expected fcpxml title text to update, got %#v", title)
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			exportPath := filepath.Join(t.TempDir(), tc.filename)
			result, err := service.ExportSubtitle(context.Background(), dto.SubtitleExportRequest{
				ExportPath: exportPath,
				Document:   &tc.document,
			})
			if err != nil {
				t.Fatalf("export subtitle: %v", err)
			}

			data, err := os.ReadFile(exportPath)
			if err != nil {
				t.Fatalf("read exported file: %v", err)
			}
			content := string(data)
			if result.Bytes != len(content) {
				t.Fatalf("expected byte count %d, got %#v", len(content), result)
			}

			tc.validate(t, content)
		})
	}
}

func TestLibraryServiceExportSubtitleConvertsITTSMPTEToFrameAlignedFCPXML(t *testing.T) {
	t.Parallel()

	service := &LibraryService{}
	exportPath := filepath.Join(t.TempDir(), "from-itt.fcpxml")

	sourceContent := `<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttp="http://www.w3.org/ns/ttml#parameter" ttp:timeBase="smpte" ttp:frameRate="30" ttp:frameRateMultiplier="1000 1001">
  <body>
    <div>
      <p begin="00:00:30:22" end="00:00:34:22">First cue</p>
      <p begin="00:00:35:15" end="00:00:38:00">Second cue</p>
    </div>
  </body>
</tt>`

	result, err := service.ExportSubtitle(context.Background(), dto.SubtitleExportRequest{
		ExportPath:   exportPath,
		Content:      sourceContent,
		Format:       "itt",
		TargetFormat: "fcpxml",
		ExportConfig: &dto.SubtitleExportConfig{
			FCPXML: &dto.SubtitleFCPXMLExportConfig{
				FrameDuration:        "1001/30000s",
				StartTimecodeSeconds: 3600,
				ProjectName:          "ITT to FCPXML",
			},
		},
	})
	if err != nil {
		t.Fatalf("export subtitle: %v", err)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("read exported file: %v", err)
	}
	content := string(data)
	if result.Bytes != len(content) {
		t.Fatalf("expected byte count %d, got %#v", len(content), result)
	}

	root := parseRenderedFCPXML(t, content)
	if len(root.Resources.Formats) != 1 {
		t.Fatalf("expected one fcpxml format resource, got %#v", root.Resources)
	}
	format := root.Resources.Formats[0]
	if format.FrameDuration != newFCPXMLFrameGrid("1001/30000s").formatFrames(1) {
		t.Fatalf("expected fcpxml export to keep configured frame duration, got %#v", format)
	}

	project := root.Library.Events[0].Projects[0]
	gap := project.Sequence.Spine.Gap
	if project.Sequence.TCStart == "0s" {
		t.Fatalf("expected sequence tcStart to capture configured start timecode, got %#v", project.Sequence)
	}
	if gap.Start != "0s" {
		t.Fatalf("expected gap start to use local title timeline origin, got %#v", gap)
	}
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, project.Sequence.TCStart)
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, project.Sequence.Duration)
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, gap.Offset)
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, gap.Duration)
	if len(gap.Titles) != 2 {
		t.Fatalf("expected two exported title nodes, got %#v", gap.Titles)
	}
	for _, title := range gap.Titles {
		requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, title.Offset)
		requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, title.Duration)
		requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, title.Start)
		if title.Start != title.Offset {
			t.Fatalf("expected title start to track title offset for basic title timing, got %#v", title)
		}
		if len(title.Params) != 4 || title.Params[0].Name != "Position" {
			t.Fatalf("expected basic title params in exported fcpxml, got %#v", title.Params)
		}
	}

	roundTrip := parseSubtitleDocument(content, "fcpxml")
	if len(roundTrip.Cues) != 2 {
		t.Fatalf("expected round-trip fcpxml parse to keep both cues, got %#v", roundTrip.Cues)
	}
	if roundTrip.Cues[0].Text != "First cue" || roundTrip.Cues[1].Text != "Second cue" {
		t.Fatalf("expected round-trip fcpxml cue text to stay accurate, got %#v", roundTrip.Cues)
	}
	if cueDurationMS(roundTrip.Cues[0]) <= 0 || cueDurationMS(roundTrip.Cues[1]) <= 0 {
		t.Fatalf("expected round-trip fcpxml cue durations to stay positive, got %#v", roundTrip.Cues)
	}
}

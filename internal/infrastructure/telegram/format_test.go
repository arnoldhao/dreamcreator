package telegram

import (
	"strings"
	"testing"
)

func TestRenderTelegramHTML_CoreFormatting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "inline styles",
			input: "hi _there_ **boss** `code`",
			want:  "hi <i>there</i> <b>boss</b> <code>code</code>",
		},
		{
			name:  "link",
			input: "see [docs](https://example.com)",
			want:  "see <a href=\"https://example.com\">docs</a>",
		},
		{
			name:  "paragraph break",
			input: "first\n\nsecond",
			want:  "first\n\nsecond",
		},
		{
			name:  "ordered list keeps start index",
			input: "2. two\n3. three",
			want:  "2. two\n3. three",
		},
		{
			name:  "heading flattened",
			input: "# Title",
			want:  "Title",
		},
		{
			name:  "blockquote trimmed",
			input: "> first\n> second",
			want:  "<blockquote>first\nsecond</blockquote>",
		},
		{
			name:  "fenced code block",
			input: "```js\nconst x = 1;\n```",
			want:  "<pre><code class=\"language-js\">const x = 1;\n</code></pre>",
		},
		{
			name:  "spoiler",
			input: "the answer is ||42||",
			want:  "the answer is <tg-spoiler>42</tg-spoiler>",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := RenderTelegramHTML(tc.input)
			if got != tc.want {
				t.Fatalf("unexpected html:\ninput: %q\ngot:   %q\nwant:  %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestRenderTelegramHTML_WrapsFileReferences(t *testing.T) {
	t.Parallel()
	got := RenderTelegramHTML("See README.md. Also (backup.sh).")
	if got != "See <code>README.md</code>. Also (<code>backup.sh</code>)." {
		t.Fatalf("unexpected html: %q", got)
	}
}

func TestNormalizeTelegramHTML_SanitizesAnchorScheme(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "drops javascript scheme",
			input: `<a href="javascript:alert(1)">x</a>`,
			want:  "x",
		},
		{
			name:  "keeps https scheme",
			input: `<a href="https://example.com">x</a>`,
			want:  `<a href="https://example.com">x</a>`,
		},
		{
			name:  "keeps tg scheme",
			input: `<a href="tg://user?id=1">x</a>`,
			want:  `<a href="tg://user?id=1">x</a>`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeTelegramHTML(tc.input); got != tc.want {
				t.Fatalf("unexpected normalized html:\ninput: %q\ngot:   %q\nwant:  %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeTelegramHTML_EmojiAndUnderline(t *testing.T) {
	t.Parallel()
	got := normalizeTelegramHTML(`<ins>hi</ins> <tg-emoji emoji-id="5368324170671202286"></tg-emoji>`)
	want := `<u>hi</u> <tg-emoji emoji-id="5368324170671202286"></tg-emoji>`
	if got != want {
		t.Fatalf("unexpected normalized html:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestRenderTelegramHTML_SourceLinesAvoidDuplicatedURLText(t *testing.T) {
	t.Parallel()
	urlValue := "https://example.com/s?wd=%E5%A4%A9%E6%B0%94&from=telegram"
	markdown := "来源:\n[1] <" + urlValue + ">\n[2] [百度搜索](<" + urlValue + ">)"
	got := RenderTelegramHTML(markdown)
	escapedURL := strings.ReplaceAll(urlValue, "&", "&amp;")
	if strings.Count(got, "<a href=\""+escapedURL+"\">") != 2 {
		t.Fatalf("expected two source links in html, got %q", got)
	}
	if strings.Contains(got, "("+escapedURL+")") {
		t.Fatalf("expected no duplicated parenthesized URL, got %q", got)
	}
}

func TestRenderTelegramHTML_TableFallbackToPre(t *testing.T) {
	t.Parallel()
	markdown := "| 城市 | 温度 |\n| --- | --- |\n| 东京 | 18°C |\n| 北京 | 12°C |"
	got := RenderTelegramHTML(markdown)
	if strings.Contains(strings.ToLower(got), "<table") {
		t.Fatalf("expected table tag to be downgraded, got %q", got)
	}
	if !strings.HasPrefix(got, "<pre>") || !strings.HasSuffix(got, "</pre>") {
		t.Fatalf("expected preformatted table output, got %q", got)
	}
	if !strings.Contains(got, "| 城市") || !strings.Contains(got, "| 东京") || !strings.Contains(got, "| 北京") {
		t.Fatalf("expected rendered table rows in pre output, got %q", got)
	}
}

func TestNormalizeTelegramHTML_PreservesExpandableBlockquoteAndSpanSpoiler(t *testing.T) {
	t.Parallel()
	input := `<blockquote expandable>line1` + "\n" + `line2</blockquote> <span class="tg-spoiler">secret</span>`
	got := normalizeTelegramHTML(input)
	want := `<blockquote expandable>line1` + "\n" + `line2</blockquote> <tg-spoiler>secret</tg-spoiler>`
	if got != want {
		t.Fatalf("unexpected normalized html:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestRenderTelegramHTML_TaskListFallback(t *testing.T) {
	t.Parallel()
	markdown := "- [x] done\n- [ ] todo"
	got := RenderTelegramHTML(markdown)
	if !strings.Contains(got, "• [x]") || !strings.Contains(got, "done") {
		t.Fatalf("expected checked task item marker, got %q", got)
	}
	if !strings.Contains(got, "• [ ]") || !strings.Contains(got, "todo") {
		t.Fatalf("expected unchecked task item marker, got %q", got)
	}
}

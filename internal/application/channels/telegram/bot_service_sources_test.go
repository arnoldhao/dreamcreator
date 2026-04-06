package telegram

import (
	"encoding/json"
	"strings"
	"testing"

	"dreamcreator/internal/application/chatevent"
)

func TestBuildTelegramReply_AppendsSources(t *testing.T) {
	marshal := func(value any) json.RawMessage {
		encoded, _ := json.Marshal(value)
		return encoded
	}
	parts := []chatevent.MessagePart{
		{
			Type: "source",
			Data: marshal(map[string]any{
				"url":   "https://openai.com",
				"title": "OpenAI",
			}),
		},
		{
			Type: "source",
			Data: marshal(map[string]any{
				"url":   "https://openai.com",
				"title": "Duplicate",
			}),
		},
	}

	reply := buildTelegramReply("结果如下", parts)
	if !strings.Contains(reply, "来源:") {
		t.Fatalf("expected reply to include sources section, got %q", reply)
	}
	if strings.Count(reply, "https://openai.com") != 1 {
		t.Fatalf("expected deduplicated source url, got %q", reply)
	}
}

func TestBuildTelegramSourceLine_UsesAutolinkWhenLabelEqualsURL(t *testing.T) {
	t.Parallel()
	urlValue := "https://example.com/search?q=%E5%A4%A9%E6%B0%94&from=bot"
	line := buildTelegramSourceLine(1, telegramSourceItem{URL: urlValue, Title: urlValue})
	if line != "[1] <"+urlValue+">" {
		t.Fatalf("unexpected source line, got %q", line)
	}
	if strings.Contains(line, "("+urlValue+")") {
		t.Fatalf("source line should not duplicate URL in parentheses, got %q", line)
	}
}

func TestBuildTelegramSourceLine_UsesMarkdownLinkWhenLabelDiffers(t *testing.T) {
	t.Parallel()
	urlValue := "https://example.com/path?a=1&b=2"
	line := buildTelegramSourceLine(2, telegramSourceItem{
		URL:   urlValue,
		Title: "示例来源 [带括号]",
	})
	want := "[2] [示例来源 \\[带括号\\]](<" + urlValue + ">)"
	if line != want {
		t.Fatalf("unexpected source line:\ngot:  %q\nwant: %q", line, want)
	}
}

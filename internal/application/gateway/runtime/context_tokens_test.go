package runtime

import (
	"strings"
	"testing"

	skillsdto "dreamcreator/internal/application/skills/dto"
	domainassistant "dreamcreator/internal/domain/assistant"
)

func TestLimitSkillPromptItemsRespectsMaxCount(t *testing.T) {
	t.Parallel()

	items := []skillsdto.SkillPromptItem{
		{Name: "skill-c", Description: "desc-c"},
		{Name: "skill-a", Description: "desc-a"},
		{Name: "skill-b", Description: "desc-b"},
	}
	limited := limitSkillPromptItems(items, domainassistant.AssistantSkills{
		MaxSkillsInPrompt: 2,
		MaxPromptChars:    10_000,
	})
	if len(limited) != 2 {
		t.Fatalf("expected 2 items, got %d", len(limited))
	}
	if limited[0].Name != "skill-a" || limited[1].Name != "skill-b" {
		t.Fatalf("expected sorted truncation [skill-a, skill-b], got [%s, %s]", limited[0].Name, limited[1].Name)
	}
}

func TestLimitSkillPromptItemsRespectsMaxChars(t *testing.T) {
	t.Parallel()

	items := []skillsdto.SkillPromptItem{
		{Name: "alpha-skill", Description: "first skill"},
		{Name: "beta-skill", Description: "second skill"},
	}
	preambleChars := len(strings.Join(skillsSectionPreambleLines(), "\n")) + 1
	firstLineChars := len(buildSkillPromptLine(items[0])) + 1
	limit := preambleChars + firstLineChars

	limited := limitSkillPromptItems(items, domainassistant.AssistantSkills{
		MaxSkillsInPrompt: 10,
		MaxPromptChars:    limit,
	})
	if len(limited) != 1 {
		t.Fatalf("expected 1 item within max chars, got %d", len(limited))
	}
	if limited[0].Name != "alpha-skill" {
		t.Fatalf("expected alpha-skill retained, got %s", limited[0].Name)
	}
}

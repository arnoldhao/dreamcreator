package assistant

import "testing"

func TestNormalizeCallSkillsConfigPreservesModeAndAllowList(t *testing.T) {
	item, err := NewAssistant(AssistantParams{
		ID: "assistant-1",
		Call: AssistantCall{
			Skills: CallSkillsConfig{
				Mode:      "custom",
				AllowList: []string{" skill-a ", "", "skill-a"},
			},
		},
	})
	if err != nil {
		t.Fatalf("new assistant failed: %v", err)
	}

	if item.Call.Skills.Mode != CallModeCustom {
		t.Fatalf("expected mode=%q, got %q", CallModeCustom, item.Call.Skills.Mode)
	}
	if len(item.Call.Skills.AllowList) != 1 || item.Call.Skills.AllowList[0] != "skill-a" {
		t.Fatalf("unexpected allow list: %#v", item.Call.Skills.AllowList)
	}
}

func TestNormalizeCallToolsConfigPreservesListsAndDisabledMode(t *testing.T) {
	item, err := NewAssistant(AssistantParams{
		ID: "assistant-2",
		Call: AssistantCall{
			Tools: CallToolsConfig{
				Mode:      "disabled",
				AllowList: []string{" tool-a ", "tool-b", "tool-a"},
				DenyList:  []string{" tool-c ", "", "tool-c"},
			},
		},
	})
	if err != nil {
		t.Fatalf("new assistant failed: %v", err)
	}

	if item.Call.Tools.Mode != "off" {
		t.Fatalf("expected mode=off, got %q", item.Call.Tools.Mode)
	}
	if len(item.Call.Tools.AllowList) != 2 {
		t.Fatalf("unexpected allow list: %#v", item.Call.Tools.AllowList)
	}
	if len(item.Call.Tools.DenyList) != 1 || item.Call.Tools.DenyList[0] != "tool-c" {
		t.Fatalf("unexpected deny list: %#v", item.Call.Tools.DenyList)
	}
}

func TestNormalizeAssistantSkillsDefaultsAndLimits(t *testing.T) {
	item, err := NewAssistant(AssistantParams{
		ID: "assistant-3",
		Skills: AssistantSkills{
			Mode:              "disabled",
			MaxSkillsInPrompt: -1,
			MaxPromptChars:    0,
		},
	})
	if err != nil {
		t.Fatalf("new assistant failed: %v", err)
	}

	if item.Skills.Mode != SkillsModeOff {
		t.Fatalf("expected normalized skills mode %q, got %q", SkillsModeOff, item.Skills.Mode)
	}
	if item.Skills.MaxSkillsInPrompt != DefaultAssistantMaxSkillsInPrompt {
		t.Fatalf("unexpected max skills in prompt: %d", item.Skills.MaxSkillsInPrompt)
	}
	if item.Skills.MaxPromptChars != DefaultAssistantSkillsPromptChars {
		t.Fatalf("unexpected max prompt chars: %d", item.Skills.MaxPromptChars)
	}
}

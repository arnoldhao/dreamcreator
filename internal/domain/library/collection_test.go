package library

import "testing"

func TestNormalizeModuleConfigPreservesNewGlossaryProfile(t *testing.T) {
	t.Parallel()

	config := DefaultModuleConfig()
	config.LanguageAssets.GlossaryProfiles = []GlossaryProfile{{
		Category:       "all",
		SourceLanguage: "all",
		TargetLanguage: "all",
	}}

	got := NormalizeModuleConfig(config)
	if len(got.LanguageAssets.GlossaryProfiles) != 1 {
		t.Fatalf("expected 1 glossary profile, got %d", len(got.LanguageAssets.GlossaryProfiles))
	}

	profile := got.LanguageAssets.GlossaryProfiles[0]
	if profile.ID == "" {
		t.Fatal("expected glossary profile id to be generated")
	}
	if profile.Name != "Glossary 1" {
		t.Fatalf("expected fallback glossary name, got %q", profile.Name)
	}
	if profile.Category != "all" {
		t.Fatalf("expected glossary category all, got %q", profile.Category)
	}
	if profile.SourceLanguage != "all" {
		t.Fatalf("expected glossary source language all, got %q", profile.SourceLanguage)
	}
	if profile.TargetLanguage != "all" {
		t.Fatalf("expected glossary target language all, got %q", profile.TargetLanguage)
	}
}

func TestNormalizeModuleConfigCollapsesLegacyThinkingLevelsToOn(t *testing.T) {
	t.Parallel()

	config := DefaultModuleConfig()
	config.TaskRuntime.Translate.ThinkingMode = "high"
	config.TaskRuntime.Proofread.ThinkingMode = "minimal"

	got := NormalizeModuleConfig(config)
	if got.TaskRuntime.Translate.ThinkingMode != "on" {
		t.Fatalf("expected translate thinking mode on, got %q", got.TaskRuntime.Translate.ThinkingMode)
	}
	if got.TaskRuntime.Proofread.ThinkingMode != "on" {
		t.Fatalf("expected proofread thinking mode on, got %q", got.TaskRuntime.Proofread.ThinkingMode)
	}
}

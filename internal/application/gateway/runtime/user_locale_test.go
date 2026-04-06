package runtime

import (
	"testing"

	domainassistant "dreamcreator/internal/domain/assistant"
)

func TestHydrateRuntimeLocaleCurrent_AutoUsesFallback(t *testing.T) {
	locale, needsPersist := hydrateRuntimeLocaleCurrent(domainassistant.UserLocale{Mode: "auto"}, "zh-CN", false)
	if locale.Current != "zh-CN" {
		t.Fatalf("expected current zh-CN, got %q", locale.Current)
	}
	if !needsPersist {
		t.Fatalf("expected needsPersist=true")
	}
}

func TestMarkRuntimeLocaleNeedsRefresh_AutoMissingCurrent(t *testing.T) {
	_, needsPersist := markRuntimeLocaleNeedsRefresh(domainassistant.UserLocale{Mode: "auto", Current: ""}, false)
	if !needsPersist {
		t.Fatalf("expected missing current to trigger refresh")
	}
}

func TestMarkRuntimeLocaleNeedsRefresh_ManualSkipsRefresh(t *testing.T) {
	_, needsPersist := markRuntimeLocaleNeedsRefresh(domainassistant.UserLocale{Mode: "manual", Value: "US"}, false)
	if needsPersist {
		t.Fatalf("expected manual locale not to trigger refresh")
	}
}

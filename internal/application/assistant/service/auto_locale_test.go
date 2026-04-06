package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"dreamcreator/internal/domain/assistant"
)

func TestPopulateAutoUserLocale_LocationAutoUsesResolver(t *testing.T) {
	called := 0
	svc := &AssistantService{
		resolveLocation: func(_ context.Context) string {
			called++
			return "San Francisco, California, United States"
		},
		now: time.Now,
	}
	user := assistant.AssistantUser{
		Language: assistant.UserLocale{Mode: "manual", Value: "en-US"},
		Timezone: assistant.UserLocale{Mode: "manual", Value: "America/Los_Angeles"},
		Location: assistant.UserLocale{Mode: "auto"},
	}

	updated, changed := svc.populateAutoUserLocale(context.Background(), user)
	if !changed {
		t.Fatalf("expected user locale changed")
	}
	if called != 1 {
		t.Fatalf("expected resolver called once, got %d", called)
	}
	if updated.Location.Current != "San Francisco, California, United States" {
		t.Fatalf("unexpected location current %q", updated.Location.Current)
	}
}

func TestPopulateAutoUserLocale_LocationManualSkipsResolver(t *testing.T) {
	called := 0
	svc := &AssistantService{
		resolveLocation: func(_ context.Context) string {
			called++
			return "ignored"
		},
		now: time.Now,
	}
	user := assistant.AssistantUser{
		Language: assistant.UserLocale{Mode: "manual", Value: "en-US"},
		Timezone: assistant.UserLocale{Mode: "manual", Value: "America/Los_Angeles"},
		Location: assistant.UserLocale{Mode: "manual", Value: "Japan"},
	}

	updated, changed := svc.populateAutoUserLocale(context.Background(), user)
	if changed {
		t.Fatalf("expected no changes for manual locale")
	}
	if called != 0 {
		t.Fatalf("expected resolver not called, got %d", called)
	}
	if updated.Location.Current != "" {
		t.Fatalf("expected empty location current, got %q", updated.Location.Current)
	}
}

func TestResolveCurrentLocation_FallbackToSecondProvider(t *testing.T) {
	originalIPWho := autoLocaleIPWhoEndpoint
	originalIPAPICo := autoLocaleIPAPICoEndpoint
	defer func() {
		autoLocaleIPWhoEndpoint = originalIPWho
		autoLocaleIPAPICoEndpoint = originalIPAPICo
	}()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/ipwho":
			_, _ = writer.Write([]byte(`{"success":false}`))
		case "/ipapi":
			_, _ = writer.Write([]byte(`{"city":"Seattle","region":"Washington","country_name":"United States"}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	autoLocaleIPWhoEndpoint = server.URL + "/ipwho"
	autoLocaleIPAPICoEndpoint = server.URL + "/ipapi"

	svc := &AssistantService{
		httpClient: server.Client(),
		now:        time.Now,
	}
	value := svc.resolveCurrentLocation(context.Background())
	if value != "Seattle, Washington, United States" {
		t.Fatalf("unexpected fallback location %q", value)
	}
}

func TestResolveCurrentLocation_UsesCache(t *testing.T) {
	originalIPWho := autoLocaleIPWhoEndpoint
	originalIPAPICo := autoLocaleIPAPICoEndpoint
	defer func() {
		autoLocaleIPWhoEndpoint = originalIPWho
		autoLocaleIPAPICoEndpoint = originalIPAPICo
	}()

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/ipwho":
			atomic.AddInt32(&requestCount, 1)
			_, _ = writer.Write([]byte(`{"success":true,"city":"Austin","region":"Texas","country":"United States"}`))
		case "/ipapi":
			_, _ = writer.Write([]byte(`{"city":"Backup","region":"NA","country_name":"Nowhere"}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	autoLocaleIPWhoEndpoint = server.URL + "/ipwho"
	autoLocaleIPAPICoEndpoint = server.URL + "/ipapi"

	fixedNow := time.Unix(1700000000, 0)
	svc := &AssistantService{
		httpClient: server.Client(),
		now: func() time.Time {
			return fixedNow
		},
	}

	first := svc.resolveCurrentLocation(context.Background())
	second := svc.resolveCurrentLocation(context.Background())

	if first != "Austin, Texas, United States" {
		t.Fatalf("unexpected first location %q", first)
	}
	if second != first {
		t.Fatalf("expected second location to use cache, got %q", second)
	}
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Fatalf("expected one outbound request, got %d", requestCount)
	}
}

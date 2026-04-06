package update

import (
	"context"
	"errors"
	"testing"
	"time"

	"dreamcreator/internal/application/softwareupdate"
	domainupdate "dreamcreator/internal/domain/update"
)

type catalogProviderStub struct {
	fetchCount int
	catalog    softwareupdate.Catalog
	err        error
}

type downloaderStub struct {
	path string
	err  error
}

func (stub *downloaderStub) Download(_ context.Context, _ string, progress func(int)) (string, error) {
	if progress != nil {
		progress(100)
	}
	if stub.err != nil {
		return "", stub.err
	}
	return stub.path, nil
}

type installerStub struct {
	installErr error
	restartErr error
	restarted  bool
}

func (stub installerStub) Install(_ context.Context, _ string) error {
	return stub.installErr
}

func (stub *installerStub) RestartToApply(_ context.Context) error {
	stub.restarted = true
	return stub.restartErr
}

func (stub *catalogProviderStub) FetchCatalog(_ context.Context, _ softwareupdate.Request) (softwareupdate.Catalog, error) {
	stub.fetchCount++
	if stub.err != nil {
		return softwareupdate.Catalog{}, stub.err
	}
	return stub.catalog, nil
}

func newCatalogService(provider *catalogProviderStub) *softwareupdate.Service {
	return softwareupdate.NewService(softwareupdate.ServiceParams{
		CatalogProvider: provider,
	})
}

func buildCatalog(version string, downloadURL string) softwareupdate.Catalog {
	return softwareupdate.Catalog{
		App: &softwareupdate.AppRelease{
			Version: version,
			Asset: softwareupdate.Asset{
				Sources: []softwareupdate.DownloadSource{
					{
						Name:     "test",
						URL:      downloadURL,
						Priority: 10,
						Enabled:  true,
					},
				},
			},
		},
	}
}

func TestSafeCheckAlwaysRunsWhenUpdateAlreadyAvailableToday(t *testing.T) {
	t.Parallel()

	provider := &catalogProviderStub{
		catalog: buildCatalog("1.2.4", "https://example.com/download.zip"),
	}
	service := NewService(ServiceParams{Catalog: newCatalogService(provider)})
	now := time.Date(2026, time.April, 1, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	service.now = func() time.Time { return now }
	service.state = domainupdate.Info{
		Kind:           domainupdate.KindApp,
		CurrentVersion: "1.2.3",
		LatestVersion:  "1.2.4",
		DownloadURL:    "https://example.com/download.zip",
		Status:         domainupdate.StatusAvailable,
		CheckedAt:      now.Add(-2 * time.Hour),
	}

	service.safeCheck(context.Background(), "1.2.3")

	if provider.fetchCount != 1 {
		t.Fatalf("expected auto-check to run, got %d fetches", provider.fetchCount)
	}
}

func TestManualCheckStillRunsWhenUpdateAlreadyAvailableToday(t *testing.T) {
	t.Parallel()

	provider := &catalogProviderStub{
		catalog: buildCatalog("1.2.4", "https://example.com/download.zip"),
	}
	service := NewService(ServiceParams{Catalog: newCatalogService(provider)})
	now := time.Date(2026, time.April, 1, 18, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	service.now = func() time.Time { return now }
	service.state = domainupdate.Info{
		Kind:           domainupdate.KindApp,
		CurrentVersion: "1.2.3",
		LatestVersion:  "1.2.4",
		DownloadURL:    "https://example.com/download.zip",
		Status:         domainupdate.StatusAvailable,
		CheckedAt:      now.Add(-1 * time.Hour),
	}

	if _, err := service.CheckForUpdate(context.Background(), "1.2.3"); err != nil {
		t.Fatalf("manual check failed: %v", err)
	}

	if provider.fetchCount != 1 {
		t.Fatalf("expected manual check to bypass auto-check skip, got %d fetches", provider.fetchCount)
	}
}

func TestCheckForUpdateReturnsNoUpdateWhenCurrentVersionIsNewerThanLatest(t *testing.T) {
	t.Parallel()

	provider := &catalogProviderStub{
		catalog: buildCatalog("1.3.0", "https://example.com/download.zip"),
	}
	service := NewService(ServiceParams{Catalog: newCatalogService(provider)})

	info, err := service.CheckForUpdate(context.Background(), "2.0.0")
	if err != nil {
		t.Fatalf("check for update failed: %v", err)
	}
	if info.Status != domainupdate.StatusNoUpdate {
		t.Fatalf("expected no_update status, got %q", info.Status)
	}
	if info.CurrentVersion != "2.0.0" {
		t.Fatalf("expected current version 2.0.0, got %q", info.CurrentVersion)
	}
	if info.LatestVersion != "1.3.0" {
		t.Fatalf("expected latest version 1.3.0, got %q", info.LatestVersion)
	}
}

func TestDownloadUpdatePublishesErrorWhenInstallerUnavailable(t *testing.T) {
	t.Parallel()

	installerErr := errors.New("installer not implemented")
	service := NewService(ServiceParams{
		Downloader: &downloaderStub{path: "/tmp/dreamcreator-update.exe"},
		Installer:  &installerStub{installErr: installerErr},
	})
	service.state = domainupdate.Info{
		Kind:        domainupdate.KindApp,
		Status:      domainupdate.StatusAvailable,
		DownloadURL: "https://example.com/dreamcreator-update.exe",
	}

	info, err := service.DownloadUpdate(context.Background())
	if !errors.Is(err, installerErr) {
		t.Fatalf("expected installer error, got %v", err)
	}
	if info.Status != domainupdate.StatusError {
		t.Fatalf("expected error status, got %q", info.Status)
	}
	if info.Message != installerErr.Error() {
		t.Fatalf("expected error message %q, got %q", installerErr.Error(), info.Message)
	}
}

func TestRestartToApplyInvokesInstallerAndResetsState(t *testing.T) {
	t.Parallel()

	installer := &installerStub{}
	service := NewService(ServiceParams{Installer: installer})
	service.state = domainupdate.Info{
		Kind:          domainupdate.KindApp,
		Status:        domainupdate.StatusReadyToRestart,
		LatestVersion: "1.2.4",
		Progress:      100,
	}

	info, err := service.RestartToApply(context.Background())
	if err != nil {
		t.Fatalf("restart to apply failed: %v", err)
	}
	if !installer.restarted {
		t.Fatal("expected installer restart hook to be called")
	}
	if info.Status != domainupdate.StatusIdle {
		t.Fatalf("expected idle status, got %q", info.Status)
	}
	if info.Progress != 0 {
		t.Fatalf("expected progress to reset, got %d", info.Progress)
	}
}

func TestRestartToApplyPublishesErrorWhenInstallerFails(t *testing.T) {
	t.Parallel()

	restartErr := errors.New("launch helper failed")
	installer := &installerStub{restartErr: restartErr}
	service := NewService(ServiceParams{Installer: installer})
	service.state = domainupdate.Info{
		Kind:   domainupdate.KindApp,
		Status: domainupdate.StatusReadyToRestart,
	}

	info, err := service.RestartToApply(context.Background())
	if !errors.Is(err, restartErr) {
		t.Fatalf("expected restart error, got %v", err)
	}
	if info.Status != domainupdate.StatusError {
		t.Fatalf("expected error status, got %q", info.Status)
	}
	if info.Message != restartErr.Error() {
		t.Fatalf("expected error message %q, got %q", restartErr.Error(), info.Message)
	}
}

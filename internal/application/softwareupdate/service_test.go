package softwareupdate

import (
	"context"
	"testing"
	"time"

	"dreamcreator/internal/domain/externaltools"
)

type catalogProviderStub struct {
	catalog Catalog
	err     error
	calls   int
}

func (stub *catalogProviderStub) FetchCatalog(_ context.Context, _ Request) (Catalog, error) {
	stub.calls++
	if stub.err != nil {
		return Catalog{}, stub.err
	}
	return stub.catalog, nil
}

type toolFallbackProviderStub struct {
	release ToolRelease
	err     error
	calls   int
}

func (stub *toolFallbackProviderStub) FetchToolRelease(_ context.Context, _ ToolRequest) (ToolRelease, error) {
	stub.calls++
	if stub.err != nil {
		return ToolRelease{}, stub.err
	}
	return stub.release, nil
}

func TestResolveToolReleaseUsesManifestCatalogFirst(t *testing.T) {
	t.Parallel()

	service := NewService(ServiceParams{
		CatalogProvider: &catalogProviderStub{
			catalog: Catalog{
				Tools: map[externaltools.ToolName]ToolRelease{
					externaltools.ToolYTDLP: {
						Name:               externaltools.ToolYTDLP,
						RecommendedVersion: "2026.03.17",
					},
				},
			},
		},
		ToolFallbackProvider: &toolFallbackProviderStub{
			release: ToolRelease{
				Name:               externaltools.ToolYTDLP,
				RecommendedVersion: "2025.12.01",
			},
		},
	})

	release, err := service.ResolveToolRelease(context.Background(), ToolRequest{Name: externaltools.ToolYTDLP})
	if err != nil {
		t.Fatalf("resolve tool release failed: %v", err)
	}
	if release.ResolvedBy != SourceManifest {
		t.Fatalf("expected manifest source, got %q", release.ResolvedBy)
	}
	if release.TargetVersion() != "2026.03.17" {
		t.Fatalf("unexpected target version: %s", release.TargetVersion())
	}
}

func TestResolveToolReleaseFallsBackAndCaches(t *testing.T) {
	t.Parallel()

	catalogProvider := &catalogProviderStub{catalog: Catalog{}}
	fallbackProvider := &toolFallbackProviderStub{
		release: ToolRelease{
			Name:               externaltools.ToolClawHub,
			RecommendedVersion: "0.9.0",
		},
	}
	service := NewService(ServiceParams{
		CatalogProvider:      catalogProvider,
		ToolFallbackProvider: fallbackProvider,
		FallbackTTL:          time.Hour,
	})

	for i := 0; i < 2; i++ {
		release, err := service.ResolveToolRelease(context.Background(), ToolRequest{Name: externaltools.ToolClawHub})
		if err != nil {
			t.Fatalf("resolve tool release failed: %v", err)
		}
		if release.ResolvedBy != SourceFallback {
			t.Fatalf("expected fallback source, got %q", release.ResolvedBy)
		}
	}

	if fallbackProvider.calls != 1 {
		t.Fatalf("expected fallback provider to be cached, got %d calls", fallbackProvider.calls)
	}
}

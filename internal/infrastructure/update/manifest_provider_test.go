package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"dreamcreator/internal/application/softwareupdate"
)

func TestManifestCatalogProviderSelectsCurrentPlatformAssets(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"appId":"cc.dreamapp.dreamcreator",
			"manifestVersion":"2026.04.06.1",
			"defaultChannel":"stable",
			"updatedAt":"2026-04-06T02:42:11Z",
			"channels":{
				"stable":{
					"app":{
						"source":{"provider":"github-release","owner":"arnoldhao","repo":"dreamcreator"},
						"version":"1.3.0",
						"publishedAt":"2026-04-06T00:00:00Z",
						"platforms":{
							"darwin-arm64":{
								"artifactName":"DreamCreator_1.3.0_mac_arm64.dmg",
								"sources":[{"name":"github","kind":"origin","url":"https://example.com/app.dmg","priority":20,"enabled":true}],
								"installStrategy":"app-installer",
								"artifactType":"dmg"
							},
							"windows-amd64":{
								"artifactName":"DreamCreator.Setup_1.3.0.exe",
								"sources":[{"name":"github","kind":"origin","url":"https://example.com/app.exe","priority":20,"enabled":true}],
								"installStrategy":"app-installer",
								"artifactType":"exe"
							}
						}
					},
					"tools":{
						"ffmpeg":{
							"displayName":"FFmpeg",
							"kind":"external-tool",
							"source":{"provider":"github-release","owner":"jellyfin","repo":"jellyfin-ffmpeg"},
							"upstreamVersion":"7.1.3-5",
							"recommendedVersion":"7.1.3-5",
							"publishedAt":"2026-04-06T00:00:00Z",
							"platforms":{
								"darwin-arm64":{
									"artifactName":"jellyfin-ffmpeg_7.1.3-5_portable_macarm64-gpl.tar.xz",
									"sources":[{"name":"github","kind":"origin","url":"https://example.com/ffmpeg.tar.xz","priority":20,"enabled":true}],
									"installStrategy":"archive",
									"artifactType":"tar.xz",
									"binaries":["ffmpeg","ffprobe"]
								}
							}
						}
					}
				}
			}
		}`))
	}))
	defer server.Close()

	provider := NewManifestCatalogProvider(server.Client(), server.URL)
	provider.goos = "darwin"
	provider.goarch = "arm64"

	catalog, err := provider.FetchCatalog(context.Background(), softwareupdate.Request{})
	if err != nil {
		t.Fatalf("fetch catalog failed: %v", err)
	}
	if catalog.App == nil {
		t.Fatal("expected app release")
	}
	if catalog.App.Asset.ArtifactName != "DreamCreator_1.3.0_mac_arm64.dmg" {
		t.Fatalf("unexpected app asset: %s", catalog.App.Asset.ArtifactName)
	}
	ffmpeg, ok := catalog.Tools["ffmpeg"]
	if !ok {
		t.Fatal("expected ffmpeg release")
	}
	if ffmpeg.Asset.ArtifactType != "tar.xz" {
		t.Fatalf("unexpected ffmpeg artifact type: %s", ffmpeg.Asset.ArtifactType)
	}
	if ffmpeg.Asset.PrimaryExecutableName() != "ffmpeg" {
		t.Fatalf("unexpected primary executable: %s", ffmpeg.Asset.PrimaryExecutableName())
	}
}

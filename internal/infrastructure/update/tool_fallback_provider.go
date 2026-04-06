package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"dreamcreator/internal/application/softwareupdate"
	"dreamcreator/internal/domain/externaltools"
)

type ToolFallbackProvider struct {
	client *http.Client
	goos   string
	goarch string
}

func NewToolFallbackProvider(client *http.Client) *ToolFallbackProvider {
	return &ToolFallbackProvider{
		client: client,
		goos:   runtime.GOOS,
		goarch: runtime.GOARCH,
	}
}

type githubReleaseResponse struct {
	TagName     string                    `json:"tag_name"`
	Body        string                    `json:"body"`
	HTMLURL     string                    `json:"html_url"`
	PublishedAt string                    `json:"published_at"`
	Assets      []githubReleaseAssetEntry `json:"assets"`
}

type githubReleaseAssetEntry struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
}

type npmLatestResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (provider *ToolFallbackProvider) FetchToolRelease(ctx context.Context, request softwareupdate.ToolRequest) (softwareupdate.ToolRelease, error) {
	if provider == nil || provider.client == nil {
		return softwareupdate.ToolRelease{}, fmt.Errorf("tool fallback client not configured")
	}
	switch request.Name {
	case externaltools.ToolYTDLP:
		return provider.fetchGitHubToolRelease(ctx, request.Name, "yt-dlp", "yt-dlp")
	case externaltools.ToolFFmpeg:
		return provider.fetchGitHubToolRelease(ctx, request.Name, "jellyfin", "jellyfin-ffmpeg")
	case externaltools.ToolBun:
		return provider.fetchGitHubToolRelease(ctx, request.Name, "oven-sh", "bun")
	case externaltools.ToolClawHub:
		return provider.fetchNPMPackageRelease(ctx, request.Name, "clawhub")
	case externaltools.ToolPlaywright:
		return softwareupdate.ToolRelease{}, softwareupdate.ErrReleaseNotFound
	default:
		return softwareupdate.ToolRelease{}, externaltools.ErrInvalidTool
	}
}

func (provider *ToolFallbackProvider) fetchGitHubToolRelease(ctx context.Context, name externaltools.ToolName, owner, repo string) (softwareupdate.ToolRelease, error) {
	release, err := provider.fetchLatestGitHubRelease(ctx, owner, repo)
	if err != nil {
		return softwareupdate.ToolRelease{}, err
	}
	asset, version, err := provider.buildToolReleaseAsset(name, release)
	if err != nil {
		return softwareupdate.ToolRelease{}, err
	}
	displayName := string(name)
	if name == externaltools.ToolFFmpeg {
		displayName = "FFmpeg"
	} else if name == externaltools.ToolBun {
		displayName = "Bun"
	}
	return softwareupdate.ToolRelease{
		Name:               name,
		DisplayName:        displayName,
		Kind:               "external-tool",
		Source:             softwareupdate.SourceRef{Provider: "github-release", Owner: owner, Repo: repo},
		UpstreamVersion:    version,
		RecommendedVersion: version,
		PublishedAt:        parseManifestTime(release.PublishedAt),
		AutoUpdate:         name == externaltools.ToolYTDLP,
		Required:           name != externaltools.ToolBun,
		Notes:              strings.TrimSpace(release.Body),
		ReleasePage:        strings.TrimSpace(release.HTMLURL),
		Asset:              asset,
	}, nil
}

func (provider *ToolFallbackProvider) fetchNPMPackageRelease(ctx context.Context, name externaltools.ToolName, packageName string) (softwareupdate.ToolRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://registry.npmjs.org/%s/latest", packageName), nil)
	if err != nil {
		return softwareupdate.ToolRelease{}, err
	}
	resp, err := provider.client.Do(req)
	if err != nil {
		return softwareupdate.ToolRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return softwareupdate.ToolRelease{}, fmt.Errorf("npm latest request failed: http %d", resp.StatusCode)
	}
	var payload npmLatestResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return softwareupdate.ToolRelease{}, err
	}
	version := strings.TrimSpace(payload.Version)
	if version == "" {
		return softwareupdate.ToolRelease{}, softwareupdate.ErrReleaseNotFound
	}
	return softwareupdate.ToolRelease{
		Name:               name,
		DisplayName:        "ClawHub",
		Kind:               "external-tool",
		Source:             softwareupdate.SourceRef{Provider: "npm-registry", Repo: packageName},
		UpstreamVersion:    version,
		RecommendedVersion: version,
		PublishedAt:        time.Time{},
		AutoUpdate:         false,
		Required:           false,
		ReleasePage:        fmt.Sprintf("https://www.npmjs.com/package/%s/v/%s", packageName, version),
	}, nil
}

func (provider *ToolFallbackProvider) fetchLatestGitHubRelease(ctx context.Context, owner, repo string) (githubReleaseResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo), nil)
	if err != nil {
		return githubReleaseResponse{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := provider.client.Do(req)
	if err != nil {
		return githubReleaseResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return githubReleaseResponse{}, fmt.Errorf("github latest release request failed: http %d", resp.StatusCode)
	}
	var release githubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubReleaseResponse{}, err
	}
	return release, nil
}

func (provider *ToolFallbackProvider) buildToolReleaseAsset(name externaltools.ToolName, release githubReleaseResponse) (softwareupdate.Asset, string, error) {
	switch name {
	case externaltools.ToolYTDLP:
		assetName := "yt-dlp"
		executableName := "yt-dlp"
		if provider.goos == "windows" {
			assetName = "yt-dlp.exe"
			executableName = "yt-dlp.exe"
		} else if provider.goos == "darwin" {
			assetName = "yt-dlp_macos"
		}
		asset, err := selectAssetByName(release.Assets, assetName)
		if err != nil {
			return softwareupdate.Asset{}, "", err
		}
		version := strings.TrimSpace(release.TagName)
		return toFallbackAsset(asset, softwareupdate.Asset{
			InstallStrategy: "binary",
			ArtifactType:    "raw-binary",
			ExecutableName:  executableName,
		}), version, nil
	case externaltools.ToolBun:
		assetName, version := provider.bunAssetName(strings.TrimSpace(release.TagName))
		asset, err := selectAssetByName(release.Assets, assetName)
		if err != nil {
			return softwareupdate.Asset{}, "", err
		}
		executableName := "bun"
		if provider.goos == "windows" {
			executableName = "bun.exe"
		}
		return toFallbackAsset(asset, softwareupdate.Asset{
			InstallStrategy: "archive",
			ArtifactType:    "zip",
			Binaries:        []string{executableName},
		}), version, nil
	case externaltools.ToolFFmpeg:
		assetName, version := provider.ffmpegAssetName(strings.TrimSpace(release.TagName))
		asset, err := selectAssetByName(release.Assets, assetName)
		if err != nil {
			return softwareupdate.Asset{}, "", err
		}
		binaries := []string{"ffmpeg", "ffprobe"}
		artifactType := "tar.xz"
		if provider.goos == "windows" {
			binaries = []string{"ffmpeg.exe", "ffprobe.exe"}
			artifactType = "zip"
		}
		return toFallbackAsset(asset, softwareupdate.Asset{
			InstallStrategy: "archive",
			ArtifactType:    artifactType,
			Binaries:        binaries,
		}), version, nil
	default:
		return softwareupdate.Asset{}, "", externaltools.ErrInvalidTool
	}
}

func (provider *ToolFallbackProvider) ffmpegAssetName(tag string) (string, string) {
	version := normalizeFFmpegVersion(tag)
	switch provider.goos {
	case "windows":
		return fmt.Sprintf("jellyfin-ffmpeg_%s_portable_win64-clang-gpl.zip", version), version
	case "darwin":
		if provider.goarch == "arm64" {
			return fmt.Sprintf("jellyfin-ffmpeg_%s_portable_macarm64-gpl.tar.xz", version), version
		}
		return fmt.Sprintf("jellyfin-ffmpeg_%s_portable_mac64-gpl.tar.xz", version), version
	default:
		return "", version
	}
}

func (provider *ToolFallbackProvider) bunAssetName(tag string) (string, string) {
	version := normalizeBunVersion(tag)
	switch provider.goos {
	case "windows":
		return "bun-windows-x64.zip", version
	case "darwin":
		if provider.goarch == "arm64" {
			return "bun-darwin-aarch64.zip", version
		}
		return "bun-darwin-x64.zip", version
	default:
		if provider.goarch == "arm64" {
			return "bun-linux-aarch64.zip", version
		}
		return "bun-linux-x64.zip", version
	}
}

func selectAssetByName(assets []githubReleaseAssetEntry, name string) (githubReleaseAssetEntry, error) {
	for _, asset := range assets {
		if strings.EqualFold(strings.TrimSpace(asset.Name), strings.TrimSpace(name)) {
			return asset, nil
		}
	}
	return githubReleaseAssetEntry{}, softwareupdate.ErrReleaseNotFound
}

func toFallbackAsset(asset githubReleaseAssetEntry, template softwareupdate.Asset) softwareupdate.Asset {
	sha256 := strings.TrimSpace(asset.Digest)
	sha256 = strings.TrimPrefix(sha256, "sha256:")
	template.ArtifactName = strings.TrimSpace(asset.Name)
	template.ContentType = strings.TrimSpace(asset.ContentType)
	template.Size = asset.Size
	template.SHA256 = sha256
	template.Sources = []softwareupdate.DownloadSource{
		{
			Name:     "gh-proxy",
			Kind:     "proxy",
			URL:      "https://gh-proxy.com/" + strings.TrimSpace(asset.BrowserDownloadURL),
			Priority: 10,
			Enabled:  true,
		},
		{
			Name:     "github",
			Kind:     "origin",
			URL:      strings.TrimSpace(asset.BrowserDownloadURL),
			Priority: 20,
			Enabled:  true,
		},
	}
	return template
}

func normalizeFFmpegVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	trimmed = strings.TrimPrefix(trimmed, "v")
	trimmed = strings.TrimPrefix(trimmed, "V")
	return strings.TrimSpace(trimmed)
}

func normalizeBunVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	trimmed = strings.TrimPrefix(trimmed, "bun-v")
	trimmed = strings.TrimPrefix(trimmed, "bun-V")
	trimmed = strings.TrimPrefix(trimmed, "v")
	trimmed = strings.TrimPrefix(trimmed, "V")
	return strings.TrimSpace(trimmed)
}

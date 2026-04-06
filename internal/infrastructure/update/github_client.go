package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	defaultReleasesURL = "https://api.github.com/repos/arnoldhao/dreamcreator/releases"
	ghProxyPrefix      = "https://gh-proxy.com/"
)

type GithubReleaseClient struct {
	client *http.Client
}

func NewGithubReleaseClient(client *http.Client) *GithubReleaseClient {
	return &GithubReleaseClient{client: client}
}

type githubRelease struct {
	TagName string          `json:"tag_name"`
	Body    string          `json:"body"`
	HTMLURL string          `json:"html_url"`
	Assets  []githubAsset   `json:"assets"`
	Draft   bool            `json:"draft"`
	Pre     bool            `json:"prerelease"`
	Created time.Time       `json:"created_at"`
	Publish time.Time       `json:"published_at"`
	Raw     json.RawMessage `json:"-"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (client *GithubReleaseClient) FetchLatestRelease(ctx context.Context) (githubRelease, error) {
	if client == nil || client.client == nil {
		return githubRelease{}, fmt.Errorf("http client not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, defaultReleasesURL+"/latest", nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.client.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("github latest release http %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, err
	}
	if strings.TrimSpace(release.TagName) == "" {
		return githubRelease{}, fmt.Errorf("no latest release found")
	}
	return release, nil
}

func selectAsset(assets []githubAsset) string {
	return selectAssetForPlatform(runtime.GOOS, runtime.GOARCH, assets)
}

func selectAssetForPlatform(osName string, arch string, assets []githubAsset) string {
	if len(assets) == 0 {
		return ""
	}

	var candidates []githubAsset
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		switch osName {
		case "darwin":
			if strings.Contains(name, "mac") || strings.Contains(name, "darwin") || strings.Contains(name, "macos") {
				candidates = append(candidates, asset)
			}
		case "windows":
			if strings.Contains(name, "win") || strings.Contains(name, "windows") {
				candidates = append(candidates, asset)
			}
		default:
			if strings.Contains(name, "linux") {
				candidates = append(candidates, asset)
			}
		}
	}

	if len(candidates) == 0 {
		candidates = assets
	}

	bestScore := -1
	bestURL := ""
	for _, asset := range candidates {
		score := scoreAssetForPlatform(osName, arch, asset.Name)
		if score > bestScore {
			bestScore = score
			bestURL = asset.BrowserDownloadURL
		}
	}
	return bestURL
}

func scoreAssetForPlatform(osName string, arch string, name string) int {
	normalized := strings.ToLower(strings.TrimSpace(name))
	score := 0

	switch osName {
	case "windows":
		switch {
		case strings.Contains(normalized, "installer") && strings.HasSuffix(normalized, ".exe"):
			score += 500
		case strings.HasSuffix(normalized, ".exe"):
			score += 420
		case strings.HasSuffix(normalized, ".zip"):
			score += 240
		}
	case "darwin":
		switch {
		case strings.HasSuffix(normalized, ".zip"):
			score += 420
		case strings.HasSuffix(normalized, ".dmg"):
			score += 320
		}
	default:
		if strings.HasSuffix(normalized, ".tar.gz") {
			score += 320
		}
		if strings.HasSuffix(normalized, ".zip") {
			score += 220
		}
	}

	if matchesPreferredArch(normalized, arch) {
		score += 120
	}
	if matchesOppositeArch(normalized, arch) {
		score -= 120
	}
	return score
}

func matchesPreferredArch(name string, arch string) bool {
	switch arch {
	case "arm64":
		return strings.Contains(name, "arm64") || strings.Contains(name, "aarch64")
	case "amd64":
		return strings.Contains(name, "x64") || strings.Contains(name, "amd64")
	default:
		return false
	}
}

func matchesOppositeArch(name string, arch string) bool {
	switch arch {
	case "arm64":
		return strings.Contains(name, "x64") || strings.Contains(name, "amd64")
	case "amd64":
		return strings.Contains(name, "arm64") || strings.Contains(name, "aarch64")
	default:
		return false
	}
}

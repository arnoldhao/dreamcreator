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

const defaultManifestURL = "https://updates.dreamapp.cc/dreamcreator/manifest.json"

type ManifestCatalogProvider struct {
	client      *http.Client
	manifestURL string
	goos        string
	goarch      string
}

func NewManifestCatalogProvider(client *http.Client, manifestURL string) *ManifestCatalogProvider {
	if strings.TrimSpace(manifestURL) == "" {
		manifestURL = defaultManifestURL
	}
	return &ManifestCatalogProvider{
		client:      client,
		manifestURL: manifestURL,
		goos:        runtime.GOOS,
		goarch:      runtime.GOARCH,
	}
}

type manifestDocument struct {
	AppID           string                     `json:"appId"`
	ManifestVersion string                     `json:"manifestVersion"`
	DefaultChannel  string                     `json:"defaultChannel"`
	UpdatedAt       string                     `json:"updatedAt"`
	Channels        map[string]manifestChannel `json:"channels"`
}

type manifestChannel struct {
	App   *manifestApp               `json:"app"`
	Tools map[string]manifestToolRef `json:"tools"`
}

type manifestSourceRef struct {
	Provider string `json:"provider"`
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
}

type manifestDownloadSource struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	URL      string `json:"url"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

type manifestPlatformAsset struct {
	ArtifactName    string                   `json:"artifactName"`
	ContentType     string                   `json:"contentType"`
	Size            int64                    `json:"size"`
	SHA256          string                   `json:"sha256"`
	Signature       string                   `json:"signature"`
	Sources         []manifestDownloadSource `json:"sources"`
	InstallStrategy string                   `json:"installStrategy"`
	ArtifactType    string                   `json:"artifactType"`
	Binaries        []string                 `json:"binaries"`
	ExecutableName  string                   `json:"executableName"`
}

type manifestApp struct {
	Source      manifestSourceRef                `json:"source"`
	Version     string                           `json:"version"`
	PublishedAt string                           `json:"publishedAt"`
	Notes       string                           `json:"notes"`
	ReleasePage string                           `json:"releasePage"`
	Platforms   map[string]manifestPlatformAsset `json:"platforms"`
}

type manifestCompatibility struct {
	MinAppVersion string `json:"minAppVersion"`
	MaxAppVersion string `json:"maxAppVersion"`
}

type manifestToolRef struct {
	DisplayName        string                           `json:"displayName"`
	Kind               string                           `json:"kind"`
	Source             manifestSourceRef                `json:"source"`
	UpstreamVersion    string                           `json:"upstreamVersion"`
	RecommendedVersion string                           `json:"recommendedVersion"`
	PublishedAt        string                           `json:"publishedAt"`
	AutoUpdate         bool                             `json:"autoUpdate"`
	Required           bool                             `json:"required"`
	Notes              string                           `json:"notes"`
	ReleasePage        string                           `json:"releasePage"`
	Compatibility      manifestCompatibility            `json:"compatibility"`
	Platforms          map[string]manifestPlatformAsset `json:"platforms"`
}

func (provider *ManifestCatalogProvider) FetchCatalog(ctx context.Context, request softwareupdate.Request) (softwareupdate.Catalog, error) {
	if provider == nil || provider.client == nil {
		return softwareupdate.Catalog{}, fmt.Errorf("manifest http client not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.manifestURL, nil)
	if err != nil {
		return softwareupdate.Catalog{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := provider.client.Do(req)
	if err != nil {
		return softwareupdate.Catalog{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return softwareupdate.Catalog{}, fmt.Errorf("manifest request failed: http %d", resp.StatusCode)
	}

	var document manifestDocument
	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return softwareupdate.Catalog{}, err
	}

	channelName := strings.TrimSpace(request.Channel)
	if channelName == "" {
		channelName = strings.TrimSpace(document.DefaultChannel)
	}
	if channelName == "" {
		channelName = "stable"
	}

	channel, ok := document.Channels[channelName]
	if !ok {
		return softwareupdate.Catalog{}, fmt.Errorf("manifest channel %q not found", channelName)
	}

	catalog := softwareupdate.Catalog{
		AppID:           strings.TrimSpace(document.AppID),
		ManifestVersion: strings.TrimSpace(document.ManifestVersion),
		Channel:         channelName,
		UpdatedAt:       parseManifestTime(document.UpdatedAt),
		Tools:           make(map[externaltools.ToolName]softwareupdate.ToolRelease),
	}

	platformKey := provider.platformKey()
	if channel.App != nil {
		if asset, ok := channel.App.Platforms[platformKey]; ok {
			catalog.App = &softwareupdate.AppRelease{
				Version:     strings.TrimSpace(channel.App.Version),
				PublishedAt: parseManifestTime(channel.App.PublishedAt),
				Notes:       strings.TrimSpace(channel.App.Notes),
				ReleasePage: strings.TrimSpace(channel.App.ReleasePage),
				Source:      toSourceRef(channel.App.Source),
				Asset:       toAsset(asset),
			}
		}
	}

	for name, tool := range channel.Tools {
		platformAsset, ok := tool.Platforms[platformKey]
		if !ok {
			continue
		}
		toolName := externaltools.ToolName(strings.TrimSpace(name))
		if toolName == "" {
			continue
		}
		catalog.Tools[toolName] = softwareupdate.ToolRelease{
			Name:               toolName,
			DisplayName:        strings.TrimSpace(tool.DisplayName),
			Kind:               strings.TrimSpace(tool.Kind),
			Source:             toSourceRef(tool.Source),
			UpstreamVersion:    strings.TrimSpace(tool.UpstreamVersion),
			RecommendedVersion: strings.TrimSpace(tool.RecommendedVersion),
			PublishedAt:        parseManifestTime(tool.PublishedAt),
			AutoUpdate:         tool.AutoUpdate,
			Required:           tool.Required,
			Notes:              strings.TrimSpace(tool.Notes),
			ReleasePage:        strings.TrimSpace(tool.ReleasePage),
			Compatibility: softwareupdate.Compatibility{
				MinAppVersion: strings.TrimSpace(tool.Compatibility.MinAppVersion),
				MaxAppVersion: strings.TrimSpace(tool.Compatibility.MaxAppVersion),
			},
			Asset: toAsset(platformAsset),
		}
	}

	return catalog, nil
}

func (provider *ManifestCatalogProvider) platformKey() string {
	switch provider.goos {
	case "darwin":
		if provider.goarch == "arm64" {
			return "darwin-arm64"
		}
		return "darwin-amd64"
	case "windows":
		return "windows-amd64"
	default:
		return provider.goos + "-" + provider.goarch
	}
}

func toSourceRef(source manifestSourceRef) softwareupdate.SourceRef {
	return softwareupdate.SourceRef{
		Provider: strings.TrimSpace(source.Provider),
		Owner:    strings.TrimSpace(source.Owner),
		Repo:     strings.TrimSpace(source.Repo),
	}
}

func toAsset(asset manifestPlatformAsset) softwareupdate.Asset {
	sources := make([]softwareupdate.DownloadSource, 0, len(asset.Sources))
	for _, source := range asset.Sources {
		sources = append(sources, softwareupdate.DownloadSource{
			Name:     strings.TrimSpace(source.Name),
			Kind:     strings.TrimSpace(source.Kind),
			URL:      strings.TrimSpace(source.URL),
			Priority: source.Priority,
			Enabled:  source.Enabled,
		})
	}
	return softwareupdate.Asset{
		ArtifactName:    strings.TrimSpace(asset.ArtifactName),
		ContentType:     strings.TrimSpace(asset.ContentType),
		Size:            asset.Size,
		SHA256:          strings.TrimSpace(asset.SHA256),
		Signature:       strings.TrimSpace(asset.Signature),
		Sources:         sources,
		InstallStrategy: strings.TrimSpace(asset.InstallStrategy),
		ArtifactType:    strings.TrimSpace(asset.ArtifactType),
		Binaries:        append([]string(nil), asset.Binaries...),
		ExecutableName:  strings.TrimSpace(asset.ExecutableName),
	}
}

func parseManifestTime(raw string) time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}
	}
	ts, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}
	}
	return ts
}

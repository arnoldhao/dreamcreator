package softwareupdate

import (
	"errors"
	"sort"
	"strings"
	"time"

	"dreamcreator/internal/domain/externaltools"
)

var ErrReleaseNotFound = errors.New("software update release not found")

type Request struct {
	Channel    string
	AppVersion string
}

type AppRequest struct {
	Channel        string
	CurrentVersion string
}

type ToolRequest struct {
	Channel    string
	AppVersion string
	Name       externaltools.ToolName
}

type SourceRef struct {
	Provider string
	Owner    string
	Repo     string
}

type DownloadSource struct {
	Name     string
	Kind     string
	URL      string
	Priority int
	Enabled  bool
}

type Asset struct {
	ArtifactName    string
	ContentType     string
	Size            int64
	SHA256          string
	Signature       string
	Sources         []DownloadSource
	InstallStrategy string
	ArtifactType    string
	Binaries        []string
	ExecutableName  string
}

func (asset Asset) SortedSources() []DownloadSource {
	if len(asset.Sources) == 0 {
		return nil
	}
	sources := make([]DownloadSource, 0, len(asset.Sources))
	for _, source := range asset.Sources {
		if !source.Enabled || strings.TrimSpace(source.URL) == "" {
			continue
		}
		sources = append(sources, source)
	}
	sort.SliceStable(sources, func(i, j int) bool {
		if sources[i].Priority == sources[j].Priority {
			return sources[i].Name < sources[j].Name
		}
		return sources[i].Priority < sources[j].Priority
	})
	return sources
}

func (asset Asset) DownloadURLs() []string {
	sources := asset.SortedSources()
	if len(sources) == 0 {
		return nil
	}
	urls := make([]string, 0, len(sources))
	for _, source := range sources {
		urls = append(urls, source.URL)
	}
	return urls
}

func (asset Asset) PrimaryDownloadURL() string {
	urls := asset.DownloadURLs()
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

func (asset Asset) PrimaryExecutableName() string {
	if strings.TrimSpace(asset.ExecutableName) != "" {
		return strings.TrimSpace(asset.ExecutableName)
	}
	if len(asset.Binaries) > 0 {
		return strings.TrimSpace(asset.Binaries[0])
	}
	return ""
}

type Compatibility struct {
	MinAppVersion string
	MaxAppVersion string
}

type AppRelease struct {
	Version     string
	PublishedAt time.Time
	Notes       string
	ReleasePage string
	Source      SourceRef
	Asset       Asset
	ResolvedBy  string
}

type ToolRelease struct {
	Name               externaltools.ToolName
	DisplayName        string
	Kind               string
	Source             SourceRef
	UpstreamVersion    string
	RecommendedVersion string
	PublishedAt        time.Time
	AutoUpdate         bool
	Required           bool
	Notes              string
	ReleasePage        string
	Compatibility      Compatibility
	Asset              Asset
	ResolvedBy         string
}

func (release ToolRelease) TargetVersion() string {
	if strings.TrimSpace(release.RecommendedVersion) != "" {
		return strings.TrimSpace(release.RecommendedVersion)
	}
	return strings.TrimSpace(release.UpstreamVersion)
}

type Catalog struct {
	AppID           string
	ManifestVersion string
	Channel         string
	UpdatedAt       time.Time
	App             *AppRelease
	Tools           map[externaltools.ToolName]ToolRelease
}

func (catalog Catalog) Tool(name externaltools.ToolName) (ToolRelease, bool) {
	if catalog.Tools == nil {
		return ToolRelease{}, false
	}
	release, ok := catalog.Tools[name]
	return release, ok
}

type Snapshot struct {
	Catalog    Catalog
	CheckedAt  time.Time
	LastError  string
	LastSource string
}

package update

import (
	"context"
	"strings"

	"dreamcreator/internal/application/softwareupdate"
)

func (client *GithubReleaseClient) FetchAppRelease(ctx context.Context, request softwareupdate.AppRequest) (softwareupdate.AppRelease, error) {
	release, err := client.FetchLatestRelease(ctx)
	if err != nil {
		return softwareupdate.AppRelease{}, err
	}
	return client.toAppRelease(release), nil
}

func (client *GithubReleaseClient) FetchAppReleaseByVersion(ctx context.Context, version string) (softwareupdate.AppRelease, error) {
	release, err := client.FetchReleaseByVersion(ctx, version)
	if err != nil {
		return softwareupdate.AppRelease{}, err
	}
	return client.toAppRelease(release), nil
}

func (client *GithubReleaseClient) toAppRelease(release githubRelease) softwareupdate.AppRelease {
	assetURL := selectAsset(release.Assets)
	sources := make([]softwareupdate.DownloadSource, 0, 2)
	if strings.TrimSpace(assetURL) != "" {
		sources = append(sources, softwareupdate.DownloadSource{
			Name:     "manifest-fallback",
			Kind:     "proxy",
			URL:      ghProxyPrefix + strings.TrimSpace(assetURL),
			Priority: 10,
			Enabled:  true,
		})
		sources = append(sources, softwareupdate.DownloadSource{
			Name:     "github",
			Kind:     "origin",
			URL:      strings.TrimSpace(assetURL),
			Priority: 20,
			Enabled:  true,
		})
	}
	return softwareupdate.AppRelease{
		Version:     strings.TrimSpace(release.TagName),
		Notes:       strings.TrimSpace(release.Body),
		ReleasePage: strings.TrimSpace(release.HTMLURL),
		Source: softwareupdate.SourceRef{
			Provider: "github-release",
			Owner:    "arnoldhao",
			Repo:     "dreamcreator",
		},
		Asset: softwareupdate.Asset{
			Sources: sources,
		},
	}
}

package update

import "testing"

func TestSelectAssetForPlatformPrefersWindowsInstaller(t *testing.T) {
	t.Parallel()

	url := selectAssetForPlatform("windows", "amd64", []githubAsset{
		{Name: "dreamcreator-windows-x64-1.2.3.zip", BrowserDownloadURL: "zip"},
		{Name: "dreamcreator-windows-x64-1.2.3-installer.exe", BrowserDownloadURL: "installer"},
	})

	if url != "installer" {
		t.Fatalf("expected installer asset, got %q", url)
	}
}

func TestSelectAssetForPlatformPrefersMatchingMacArchitecture(t *testing.T) {
	t.Parallel()

	url := selectAssetForPlatform("darwin", "arm64", []githubAsset{
		{Name: "dreamcreator-macos-x64-1.2.3.zip", BrowserDownloadURL: "x64"},
		{Name: "dreamcreator-macos-arm64-1.2.3.zip", BrowserDownloadURL: "arm64"},
	})

	if url != "arm64" {
		t.Fatalf("expected arm64 asset, got %q", url)
	}
}

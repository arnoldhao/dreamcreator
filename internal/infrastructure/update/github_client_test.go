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

func TestWindowsDownloadURLPreferences(t *testing.T) {
	t.Parallel()

	installerURL := "https://example.com/dreamcreator-windows-x64-1.2.3-installer.exe"
	portableURL := "https://example.com/dreamcreator-windows-x64-1.2.3.zip"

	installed := preferWindowsInstallerDownloadURLs([]string{portableURL})
	if len(installed) != 1 {
		t.Fatalf("expected derived installer URL, got %#v", installed)
	}
	if installed[0] != installerURL {
		t.Fatalf("expected installer URL first, got %#v", installed)
	}

	portable := preferWindowsPortableDownloadURLs([]string{installerURL})
	if len(portable) != 1 {
		t.Fatalf("expected derived portable URL, got %#v", portable)
	}
	if portable[0] != portableURL {
		t.Fatalf("expected portable URL first, got %#v", portable)
	}
}

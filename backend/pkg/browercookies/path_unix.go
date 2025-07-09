//go:build darwin || linux

package browercookies

import (
	"path/filepath"
	"runtime"
	"CanMe/backend/consts"
)

func getCookieFilePath(browserType consts.BrowserType, homeDir string) string {
	if runtime.GOOS == "darwin" {
		// macOS 路径
		switch browserType {
		case consts.Safari:
			return filepath.Join(homeDir, "Library", "Cookies", "Cookies.binarycookies")
		case consts.Chrome:
			return filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "Default", "Cookies")
		case consts.Edge:
			return filepath.Join(homeDir, "Library", "Application Support", "Microsoft Edge", "Default", "Cookies")
		case consts.Chromium:
			return filepath.Join(homeDir, "Library", "Application Support", "Chromium", "Default", "Cookies")
		case consts.Firefox:
			return filepath.Join(homeDir, "Library", "Application Support", "Firefox", "Profiles")
		default:
			return ""
		}
	} else {
		// Linux 路径
		switch browserType {
		case consts.Chrome:
			return filepath.Join(homeDir, ".config", "google-chrome", "Default", "Cookies")
		case consts.Edge:
			return filepath.Join(homeDir, ".config", "microsoft-edge", "Default", "Cookies")
		case consts.Chromium:
			return filepath.Join(homeDir, ".config", "chromium", "Default", "Cookies")
		case consts.Firefox:
			return filepath.Join(homeDir, ".mozilla", "firefox")
		default:
			return ""
		}
	}
}
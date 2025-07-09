//go:build windows

package browercookies

import (
	"CanMe/backend/consts"
	"path/filepath"
)

func getCookieFilePath(browserType consts.BrowserType, homeDir string) string {
	switch browserType {
	case consts.Chrome:
		return filepath.Join(homeDir, "AppData", "Local", "Google", "Chrome", "User Data", "Default", "Network", "Cookies")
	case consts.Edge:
		return filepath.Join(homeDir, "AppData", "Local", "Microsoft", "Edge", "User Data", "Default", "Network", "Cookies")
	case consts.Chromium:
		return filepath.Join(homeDir, "AppData", "Local", "Chromium", "User Data", "Default", "Network", "Cookies")
	case consts.Firefox:
		// Firefox 需要动态查找 profile 文件夹，这里返回基础路径
		return filepath.Join(homeDir, "AppData", "Roaming", "Mozilla", "Firefox", "Profiles")
	default:
		return ""
	}
}

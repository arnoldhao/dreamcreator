//go:build windows

package browercookies

import (
	"dreamcreator/backend/consts"
	"os"
	"path/filepath"
	"strings"
)

func getCookieFilePath(browserType consts.BrowserType, homeDir string) string {
	switch browserType {
	case consts.Chrome:
		return filepath.Join(homeDir, "AppData", "Local", "Google", "Chrome", "User Data", "Default", "Network", "Cookies")
	case consts.Brave:
		return filepath.Join(homeDir, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data", "Default", "Network", "Cookies")
	case consts.Edge:
		return filepath.Join(homeDir, "AppData", "Local", "Microsoft", "Edge", "User Data", "Default", "Network", "Cookies")
	case consts.Chromium:
		return filepath.Join(homeDir, "AppData", "Local", "Chromium", "User Data", "Default", "Network", "Cookies")
	case consts.Vivaldi:
		return filepath.Join(homeDir, "AppData", "Local", "Vivaldi", "User Data", "Default", "Network", "Cookies")
	case consts.Opera:
		return filepath.Join(homeDir, "AppData", "Roaming", "Opera Software", "Opera Stable", "Network", "Cookies")
	case consts.Firefox:
		// Firefox 需要动态查找 profile 文件夹，这里返回基础路径
		return filepath.Join(homeDir, "AppData", "Roaming", "Mozilla", "Firefox", "Profiles")
	default:
		return ""
	}
}

// listCandidateCookiePaths 返回可能存在的 cookie 路径（包含多 Profile）
func listCandidateCookiePaths(browserType consts.BrowserType, homeDir string) []string {
	var out []string
	switch browserType {
	case consts.Chrome:
		root := filepath.Join(homeDir, "AppData", "Local", "Google", "Chrome", "User Data")
		out = append(out, enumerateChromiumProfiles(root)...)
	case consts.Brave:
		root := filepath.Join(homeDir, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data")
		out = append(out, enumerateChromiumProfiles(root)...)
	case consts.Edge:
		root := filepath.Join(homeDir, "AppData", "Local", "Microsoft", "Edge", "User Data")
		out = append(out, enumerateChromiumProfiles(root)...)
	case consts.Chromium:
		root := filepath.Join(homeDir, "AppData", "Local", "Chromium", "User Data")
		out = append(out, enumerateChromiumProfiles(root)...)
	case consts.Vivaldi:
		root := filepath.Join(homeDir, "AppData", "Local", "Vivaldi", "User Data")
		out = append(out, enumerateChromiumProfiles(root)...)
	case consts.Opera:
		base := filepath.Join(homeDir, "AppData", "Roaming", "Opera Software")
		if entries, err := os.ReadDir(base); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					for _, sub := range []string{
						filepath.Join(base, e.Name(), "Network", "Cookies"),
						filepath.Join(base, e.Name(), "Cookies"),
					} {
						if _, err := os.Stat(sub); err == nil {
							out = append(out, sub)
						}
					}
				}
			}
		}
		// Opera Stable 常驻路径兜底
		for _, sub := range []string{
			filepath.Join(base, "Opera Stable", "Network", "Cookies"),
			filepath.Join(base, "Opera Stable", "Cookies"),
		} {
			if _, err := os.Stat(sub); err == nil {
				out = append(out, sub)
			}
		}
	case consts.Firefox:
		root := filepath.Join(homeDir, "AppData", "Roaming", "Mozilla", "Firefox", "Profiles")
		if entries, err := os.ReadDir(root); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					p := filepath.Join(root, e.Name(), "cookies.sqlite")
					if _, err := os.Stat(p); err == nil {
						out = append(out, p)
					}
				}
			}
		}
	default:
		// Safari 不支持 Windows
	}
	return out
}

// enumerateChromiumProfiles 列举所有包含 Cookie 的 Profile 数据库
func enumerateChromiumProfiles(userDataRoot string) []string {
	var out []string
	// 默认 profile
	defaults := []string{"Default"}
	if entries, err := os.ReadDir(userDataRoot); err == nil {
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() && (name == "Default" || strings.HasPrefix(name, "Profile ")) {
				defaults = append(defaults, name)
			}
		}
	}
	for _, prof := range defaults {
		// Prefer Network/Cookies, fall back to Cookies
		p1 := filepath.Join(userDataRoot, prof, "Network", "Cookies")
		p2 := filepath.Join(userDataRoot, prof, "Cookies")
		if _, err := os.Stat(p1); err == nil {
			out = append(out, p1)
		} else if _, err := os.Stat(p2); err == nil {
			out = append(out, p2)
		}
	}
	return out
}

// deriveChromiumProfile 从 Cookies 路径提取 Profile 名称（如 Default, Profile 1）
func deriveChromiumProfile(cookiePath string) string {
	// 支持 .../<Profile>/Network/Cookies 以及 .../<Profile>/Cookies 结构
	dir := filepath.Dir(cookiePath)
	base := filepath.Base(dir)
	if strings.EqualFold(base, "Network") || strings.EqualFold(base, "Cookies") {
		return filepath.Base(filepath.Dir(dir))
	}
	return base
}

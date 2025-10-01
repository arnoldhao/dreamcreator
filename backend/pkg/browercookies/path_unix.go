//go:build darwin || linux

package browercookies

import (
	"dreamcreator/backend/consts"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func getCookieFilePath(browserType consts.BrowserType, homeDir string) string {
	if runtime.GOOS == "darwin" {
		// macOS 路径
		switch browserType {
		case consts.Safari:
			// 新路径（优先）
			p := filepath.Join(homeDir, "Library", "Containers", "com.apple.Safari", "Data", "Library", "Cookies", "Cookies.binarycookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			// 旧路径（回退）
			return filepath.Join(homeDir, "Library", "Cookies", "Cookies.binarycookies")
		case consts.Chrome:
			// Chrome 在较新版本中 Cookies 迁移到 Network 子目录
			p := filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "Default", "Network", "Cookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "Default", "Cookies")
		case consts.Brave:
			p := filepath.Join(homeDir, "Library", "Application Support", "BraveSoftware", "Brave-Browser", "Default", "Network", "Cookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return filepath.Join(homeDir, "Library", "Application Support", "BraveSoftware", "Brave-Browser", "Default", "Cookies")
		case consts.Edge:
			p := filepath.Join(homeDir, "Library", "Application Support", "Microsoft Edge", "Default", "Network", "Cookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return filepath.Join(homeDir, "Library", "Application Support", "Microsoft Edge", "Default", "Cookies")
		case consts.Chromium:
			p := filepath.Join(homeDir, "Library", "Application Support", "Chromium", "Default", "Network", "Cookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return filepath.Join(homeDir, "Library", "Application Support", "Chromium", "Default", "Cookies")
		case consts.Opera:
			candidates := []string{
				filepath.Join(homeDir, "Library", "Application Support", "com.operasoftware.Opera", "Network", "Cookies"),
				filepath.Join(homeDir, "Library", "Application Support", "com.operasoftware.Opera", "Cookies"),
				filepath.Join(homeDir, "Library", "Application Support", "Opera Software", "Opera Stable", "Network", "Cookies"),
				filepath.Join(homeDir, "Library", "Application Support", "Opera Software", "Opera Stable", "Cookies"),
			}
			for _, cand := range candidates {
				if _, err := os.Stat(cand); err == nil {
					return cand
				}
			}
			return ""
		case consts.Vivaldi:
			p := filepath.Join(homeDir, "Library", "Application Support", "Vivaldi", "Default", "Network", "Cookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return filepath.Join(homeDir, "Library", "Application Support", "Vivaldi", "Default", "Cookies")
		case consts.Firefox:
			return filepath.Join(homeDir, "Library", "Application Support", "Firefox", "Profiles")
		default:
			return ""
		}
	} else {
		// Linux 路径（新优先 Network/Cookies，回退 Cookies）
		switch browserType {
		case consts.Chrome:
			p := filepath.Join(homeDir, ".config", "google-chrome", "Default", "Network", "Cookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return filepath.Join(homeDir, ".config", "google-chrome", "Default", "Cookies")
		case consts.Brave:
			p := filepath.Join(homeDir, ".config", "BraveSoftware", "Brave-Browser", "Default", "Network", "Cookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return filepath.Join(homeDir, ".config", "BraveSoftware", "Brave-Browser", "Default", "Cookies")
		case consts.Edge:
			p := filepath.Join(homeDir, ".config", "microsoft-edge", "Default", "Network", "Cookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return filepath.Join(homeDir, ".config", "microsoft-edge", "Default", "Cookies")
		case consts.Chromium:
			p := filepath.Join(homeDir, ".config", "chromium", "Default", "Network", "Cookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return filepath.Join(homeDir, ".config", "chromium", "Default", "Cookies")
		case consts.Opera:
			p := filepath.Join(homeDir, ".config", "opera", "Network", "Cookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return filepath.Join(homeDir, ".config", "opera", "Cookies")
		case consts.Vivaldi:
			p := filepath.Join(homeDir, ".config", "vivaldi", "Default", "Network", "Cookies")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return filepath.Join(homeDir, ".config", "vivaldi", "Default", "Cookies")
		case consts.Firefox:
			return filepath.Join(homeDir, ".mozilla", "firefox")
		default:
			return ""
		}
	}
}

// listCandidateCookiePaths 返回可能存在的 cookie 路径（包含多 Profile）
func listCandidateCookiePaths(browserType consts.BrowserType, homeDir string) []string {
	var out []string
	// macOS/Linux 支持多 profile 扫描
	switch runtime.GOOS {
	case "darwin":
		switch browserType {
		case consts.Safari:
			cands := []string{
				filepath.Join(homeDir, "Library", "Containers", "com.apple.Safari", "Data", "Library", "Cookies", "Cookies.binarycookies"),
				filepath.Join(homeDir, "Library", "Cookies", "Cookies.binarycookies"),
			}
			for _, p := range cands {
				if _, err := os.Stat(p); err == nil {
					out = append(out, p)
				}
			}
		case consts.Chrome:
			root := filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome")
			profiles := []string{"Default"}
			// 扫描 Profile *
			if entries, err := os.ReadDir(root); err == nil {
				for _, e := range entries {
					name := e.Name()
					if e.IsDir() && (name == "Default" || strings.HasPrefix(name, "Profile ")) {
						profiles = append(profiles, name)
					}
				}
			}
			for _, prof := range profiles {
				for _, sub := range []string{
					filepath.Join(root, prof, "Network", "Cookies"),
					filepath.Join(root, prof, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
			}
		case consts.Brave:
			root := filepath.Join(homeDir, "Library", "Application Support", "BraveSoftware", "Brave-Browser")
			profiles := []string{"Default"}
			if entries, err := os.ReadDir(root); err == nil {
				for _, e := range entries {
					name := e.Name()
					if e.IsDir() && (name == "Default" || strings.HasPrefix(name, "Profile ")) {
						profiles = append(profiles, name)
					}
				}
			}
			for _, prof := range profiles {
				for _, sub := range []string{
					filepath.Join(root, prof, "Network", "Cookies"),
					filepath.Join(root, prof, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
			}
		case consts.Edge:
			root := filepath.Join(homeDir, "Library", "Application Support", "Microsoft Edge")
			profiles := []string{"Default"}
			if entries, err := os.ReadDir(root); err == nil {
				for _, e := range entries {
					name := e.Name()
					if e.IsDir() && (name == "Default" || strings.HasPrefix(name, "Profile ")) {
						profiles = append(profiles, name)
					}
				}
			}
			for _, prof := range profiles {
				for _, sub := range []string{
					filepath.Join(root, prof, "Network", "Cookies"),
					filepath.Join(root, prof, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
			}
		case consts.Chromium:
			root := filepath.Join(homeDir, "Library", "Application Support", "Chromium")
			profiles := []string{"Default"}
			if entries, err := os.ReadDir(root); err == nil {
				for _, e := range entries {
					name := e.Name()
					if e.IsDir() && (name == "Default" || strings.HasPrefix(name, "Profile ")) {
						profiles = append(profiles, name)
					}
				}
			}
			for _, prof := range profiles {
				for _, sub := range []string{
					filepath.Join(root, prof, "Network", "Cookies"),
					filepath.Join(root, prof, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
			}
		case consts.Opera:
			roots := []string{
				filepath.Join(homeDir, "Library", "Application Support", "com.operasoftware.Opera"),
				filepath.Join(homeDir, "Library", "Application Support", "Opera Software"),
			}
			for _, root := range roots {
				if _, err := os.Stat(root); err != nil {
					continue
				}
				// 根目录自身也可能存储 Cookies
				for _, sub := range []string{
					filepath.Join(root, "Network", "Cookies"),
					filepath.Join(root, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
				if entries, err := os.ReadDir(root); err == nil {
					for _, e := range entries {
						if e.IsDir() {
							for _, sub := range []string{
								filepath.Join(root, e.Name(), "Network", "Cookies"),
								filepath.Join(root, e.Name(), "Cookies"),
							} {
								if _, err := os.Stat(sub); err == nil {
									out = append(out, sub)
								}
							}
						}
					}
				}
			}
		case consts.Vivaldi:
			root := filepath.Join(homeDir, "Library", "Application Support", "Vivaldi")
			profiles := []string{"Default"}
			if entries, err := os.ReadDir(root); err == nil {
				for _, e := range entries {
					name := e.Name()
					if e.IsDir() && (name == "Default" || strings.HasPrefix(name, "Profile ")) {
						profiles = append(profiles, name)
					}
				}
			}
			for _, prof := range profiles {
				for _, sub := range []string{
					filepath.Join(root, prof, "Network", "Cookies"),
					filepath.Join(root, prof, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
			}
		case consts.Firefox:
			root := filepath.Join(homeDir, "Library", "Application Support", "Firefox", "Profiles")
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
		}
		return out
	case "linux":
		switch browserType {
		case consts.Chrome:
			root := filepath.Join(homeDir, ".config", "google-chrome")
			profiles := []string{"Default"}
			if entries, err := os.ReadDir(root); err == nil {
				for _, e := range entries {
					name := e.Name()
					if e.IsDir() && (name == "Default" || strings.HasPrefix(name, "Profile ")) {
						profiles = append(profiles, name)
					}
				}
			}
			for _, prof := range profiles {
				for _, sub := range []string{
					filepath.Join(root, prof, "Network", "Cookies"),
					filepath.Join(root, prof, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
			}
		case consts.Brave:
			root := filepath.Join(homeDir, ".config", "BraveSoftware", "Brave-Browser")
			profiles := []string{"Default"}
			if entries, err := os.ReadDir(root); err == nil {
				for _, e := range entries {
					name := e.Name()
					if e.IsDir() && (name == "Default" || strings.HasPrefix(name, "Profile ")) {
						profiles = append(profiles, name)
					}
				}
			}
			for _, prof := range profiles {
				for _, sub := range []string{
					filepath.Join(root, prof, "Network", "Cookies"),
					filepath.Join(root, prof, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
			}
		case consts.Edge:
			root := filepath.Join(homeDir, ".config", "microsoft-edge")
			profiles := []string{"Default"}
			if entries, err := os.ReadDir(root); err == nil {
				for _, e := range entries {
					name := e.Name()
					if e.IsDir() && (name == "Default" || strings.HasPrefix(name, "Profile ")) {
						profiles = append(profiles, name)
					}
				}
			}
			for _, prof := range profiles {
				for _, sub := range []string{
					filepath.Join(root, prof, "Network", "Cookies"),
					filepath.Join(root, prof, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
			}
		case consts.Chromium:
			root := filepath.Join(homeDir, ".config", "chromium")
			profiles := []string{"Default"}
			if entries, err := os.ReadDir(root); err == nil {
				for _, e := range entries {
					name := e.Name()
					if e.IsDir() && (name == "Default" || strings.HasPrefix(name, "Profile ")) {
						profiles = append(profiles, name)
					}
				}
			}
			for _, prof := range profiles {
				for _, sub := range []string{
					filepath.Join(root, prof, "Network", "Cookies"),
					filepath.Join(root, prof, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
			}
		case consts.Opera:
			root := filepath.Join(homeDir, ".config", "opera")
			if _, err := os.Stat(root); err == nil {
				for _, sub := range []string{
					filepath.Join(root, "Network", "Cookies"),
					filepath.Join(root, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
				if entries, err := os.ReadDir(root); err == nil {
					for _, e := range entries {
						if e.IsDir() {
							for _, sub := range []string{
								filepath.Join(root, e.Name(), "Network", "Cookies"),
								filepath.Join(root, e.Name(), "Cookies"),
							} {
								if _, err := os.Stat(sub); err == nil {
									out = append(out, sub)
								}
							}
						}
					}
				}
			}
		case consts.Vivaldi:
			root := filepath.Join(homeDir, ".config", "vivaldi")
			profiles := []string{"Default"}
			if entries, err := os.ReadDir(root); err == nil {
				for _, e := range entries {
					name := e.Name()
					if e.IsDir() && (name == "Default" || strings.HasPrefix(name, "Profile ")) {
						profiles = append(profiles, name)
					}
				}
			}
			for _, prof := range profiles {
				for _, sub := range []string{
					filepath.Join(root, prof, "Network", "Cookies"),
					filepath.Join(root, prof, "Cookies"),
				} {
					if _, err := os.Stat(sub); err == nil {
						out = append(out, sub)
					}
				}
			}
		case consts.Firefox:
			root := filepath.Join(homeDir, ".mozilla", "firefox")
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
		}
		return out
	default:
		return out
	}
}

// deriveChromiumProfile 从 Cookies 路径推断 Profile 名称（Unix 系）
func deriveChromiumProfile(cookiePath string) string {
	// macOS/Linux: .../<Browser>/<Profile>/(Network/)?Cookies
	dir := filepath.Dir(cookiePath) // .../<Profile>/(Network)
	base := filepath.Base(dir)
	if base == "Network" {
		return filepath.Base(filepath.Dir(dir))
	}
	return base
}

package browercookies

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/storage"
	"CanMe/backend/types"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"CanMe/backend/pkg/dependencies"

	"github.com/lrstanley/go-ytdlp"
	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"

	"CanMe/backend/utils"
)

type cookieManager struct {
	storage    *storage.BoltStorage
	depManager dependencies.Manager
}

func NewCookieManager(storage *storage.BoltStorage, depManager dependencies.Manager) CookieManager {
	return &cookieManager{
		storage:    storage,
		depManager: depManager,
	}
}

func (c *cookieManager) Sync(ctx context.Context, syncFrom string, browsers []string) error {
	if c.storage == nil {
		return errors.New("storage is not initialized")
	}

	logger.Debug("cookies.Sync start",
		zap.String("from", syncFrom),
		zap.Strings("browsers", browsers),
		zap.String("os", runtime.GOOS),
	)

	browserCookiesMap := make(map[string]*types.BrowserCookies)
	if syncFrom == "canme" {
		// deprecated
		return fmt.Errorf("canme cookies sync is not supported; please use yt-dlp")
	} else if syncFrom == "yt-dlp" {
		// do not block elevated; return direct error from yt-dlp
		browserCookiesMap = c.readAllBrowserCookiesByYTDLP(ctx, browsers)
	} else {
		return fmt.Errorf("unsupported syncFrom: %s", syncFrom)
	}

	if len(browserCookiesMap) == 0 {
		return fmt.Errorf("no browser cookies found")
	}

	// 统计成功/失败，用于整体结果判断
	anySuccess := false
	var firstErrMsg string

	for browser, cookies := range browserCookiesMap {
		if browser != "" && cookies != nil {
			err := c.storage.SaveCookies(browser, cookies)
			if err != nil {
				logger.Error("Failed to save cookies for browser", zap.String("browser", browser), zap.Error(err))
			}
			if strings.EqualFold(cookies.LastSyncStatus, "success") {
				anySuccess = true
			} else if firstErrMsg == "" {
				firstErrMsg = cookies.StatusDescription
			}
		}
	}

	if anySuccess {
		logger.Debug("cookies.Sync complete", zap.Bool("success", true))
		return nil
	}
	if firstErrMsg == "" {
		firstErrMsg = "no cookies found"
	}
	logger.Warn("cookies.Sync complete", zap.Bool("success", false), zap.String("error", firstErrMsg))
	return fmt.Errorf(firstErrMsg)
}

// ListAllCookies retrieves all cached cookies, grouped by browser.
func (c *cookieManager) ListAllCookies() (map[string]*types.BrowserCookies, error) {
	if c.storage == nil {
		return nil, errors.New("storage is not initialized")
	}

	// available type to get cookies: yt-dlp only
	syncFrom := []string{"yt-dlp"}
	osType := runtime.GOOS

	var currentBrowsers []*types.BrowserCookies
	allBrowsers := []consts.BrowserType{consts.Chrome, consts.Chromium, consts.Firefox, consts.Edge, consts.Safari}
	for _, browser := range allBrowsers {
		homeDir, _ := os.UserHomeDir()
		// 扫描多 Profile 的候选路径，取第一个存在的作为展示路径
		var firstPath string
		var paths []string
		if osType == "windows" {
			paths = listCandidateCookiePaths(browser, homeDir)
		} else if osType == "darwin" { // 暂不预支持 Linux
			paths = listCandidateCookiePaths(browser, homeDir)
		}
		if len(paths) > 0 {
			firstPath = paths[0]
		} else {
			// 回退单一路径（兼容旧逻辑）
			p := GetCookieFilePath(browser)
			if _, err := os.Stat(p); err == nil {
				firstPath = p
			}
		}

		if firstPath != "" {
			currentBrowsers = append(currentBrowsers, &types.BrowserCookies{
				Browser:           string(browser),
				Path:              firstPath,
				Status:            "never",
				StatusDescription: "never sync cookies",
				SyncFrom:          syncFrom,
				LastSyncFrom:      "never",
				LastSyncTime:      time.Time{},
				LastSyncStatus:    "never syncd",
				DomainCookies:     nil,
			})
		} else {
			logger.Info("No cookie store detected for browser", zap.String("browser", string(browser)))
		}
	}

	// Windows 高权限运行提示：添加描述，便于前端展示指引
	if osType == "windows" {
		if utils.WindowsIsElevated() {
			for _, bc := range currentBrowsers {
				if bc != nil {
					bc.StatusDescription = "Running as Administrator may block cookie decryption. Please run as a normal user."
				}
			}
		}
	}

	savedCookies, err := c.storage.ListAllCookies()
	if err != nil {
		return nil, err
	}

	// merge currentBrowsers and savedCookies
	for _, browser := range currentBrowsers {
		if savedCookies[browser.Browser] == nil {
			savedCookies[browser.Browser] = browser
		}
	}

	return savedCookies, nil
}

func (c *cookieManager) GetBrowserByDomain(domain string) ([]string, error) {
	if c.storage == nil {
		return nil, errors.New("storage is not initialized")
	}

	allCookies, err := c.ListAllCookies()
	if err != nil {
		return nil, err
	}

	var browsers []string
	for browser, _ := range allCookies {
		c, err := c.GetCookiesByDomain(browser, domain)
		if err != nil {
			continue
		}
		if len(c) > 0 {
			browsers = append(browsers, browser)
		}
	}

	if len(browsers) == 0 {
		return nil, fmt.Errorf("no browser found for domain: %s", domain)
	}

	return browsers, nil
}

// GetCookiesByDomain retrieves cookies for a specific domain from a specified browser's cache.
func (c *cookieManager) GetCookiesByDomain(browser, domain string) ([]*http.Cookie, error) {
	if c.storage == nil {
		return nil, errors.New("storage is not initialized")
	}
	if browser == "" {
		return nil, errors.New("browser must be specified")
	}
	if domain == "" {
		return nil, errors.New("domain must be specified")
	}

	// Fetch cookies only for the specified browser, which is much more efficient.
	storedCookies, err := c.storage.GetCookies(browser)
	if err != nil {
		return nil, fmt.Errorf("failed to list cookies for browser %s: %w \n", browser, err)
	}

	// 生成domain的lookup列表
	var domainCookies []*http.Cookie
	domainLookups := c.generateDomainLookups(domain)

	// 遍历lookup列表，找到第一个匹配的domain
	for _, lookup := range domainLookups {
		if cookies := storedCookies.DomainCookies[lookup]; cookies != nil && len(cookies.Cookies) > 0 {
			domainCookies = append(domainCookies, cookies.Cookies...)
		}
	}

	return domainCookies, nil
}

func (c *cookieManager) GetNetscapeCookiesByDomain(browser, domain string) (string, error) {
	if c.storage == nil {
		return "", errors.New("storage is not initialized")
	}
	if browser == "" {
		return "", errors.New("browser must be specified")
	}
	if domain == "" {
		return "", errors.New("domain must be specified")
	}

	cookies, err := c.GetCookiesByDomain(browser, domain)
	if err != nil {
		return "", err
	}

	return c.convertToNetscape(cookies), nil
}

// readAllBrowserCookies reads all cookies from all supported browsers and groups them by browser name.
// canme sync removed

// canme sync removed
func (c *cookieManager) readAllBrowserCookiesByYTDLP(ctx context.Context, browsers []string) map[string]*types.BrowserCookies {
	browserCookiesMap := make(map[string]*types.BrowserCookies)

	// available type to get cookies: yt-dlp only
	syncFrom := []string{"yt-dlp"}
	osType := runtime.GOOS

	// 遍历使用yt-dlp导出cookies为 export_[browser]_cookies.txt
	dl := ytdlp.New()
	ytdlpExec, err := c.YTDLPExecPath(ctx)
	if err != nil {
		logger.Error("Failed to get ytdlp path", zap.Error(err))
		return browserCookiesMap
	}
	dl.SetExecutable(ytdlpExec)
	logger.Debug("ytdlp export: using executable", zap.String("exec", ytdlpExec))

	type agg struct {
		paths []string
	}
	cookiesPaths := make(map[string]*agg) // 浏览器 => 导出文件路径列表
	// 聚合每个浏览器的首要错误信息（来自 yt-dlp 的 stderr）
	errMsgByBrowser := make(map[string]string)

	for _, browser := range browsers {
		// 每个浏览器独立收集错误摘要
		var lastErrMsg string
		// Representative path for UI
		btype := consts.BrowserType(browser)
		homeDir, _ := os.UserHomeDir()
		var pathHint string
		var cands []string
		if paths := listCandidateCookiePaths(btype, homeDir); len(paths) > 0 {
			cands = paths
			pathHint = paths[0]
		} else {
			pathHint = GetCookieFilePath(btype)
		}
		// Avoid printing scanned paths; only log counts/summary
		logger.Debug("ytdlp export: detected cookie locations",
			zap.String("browser", string(btype)),
			zap.Int("candidates", len(cands)),
		)

		eachBrowserCookies := types.BrowserCookies{
			Browser:           browser,
			Path:              pathHint,
			Status:            "syncing",
			StatusDescription: "syncing cookies",
			SyncFrom:          syncFrom,
			LastSyncFrom:      "yt-dlp",
			LastSyncTime:      time.Time{},
			LastSyncStatus:    "syncing",
			DomainCookies:     make(map[string]*types.DomainCookies),
		}

		ytdlpDir, err := c.YTDLPPath(ctx)
		if err != nil {
			eachBrowserCookies.Status = "error"
			eachBrowserCookies.StatusDescription = err.Error()
			eachBrowserCookies.LastSyncTime = time.Now()
			eachBrowserCookies.LastSyncStatus = "failed"

			// save and continue
			browserCookiesMap[string(browser)] = &eachBrowserCookies
			continue
		}
		// 针对 Windows/macOS，尝试多 Profile（Default, Profile *）
		// 注：若已发现明确的 Profile，则不再追加空 Profile，避免重复导出导致多次系统弹窗（macOS 钥匙串）
		// Safari（macOS）只有单一存储，yt-dlp 不接受 safari:<profile> 形式，这里强制仅用空 Profile
		var profiles []string
		homeDir, _ = os.UserHomeDir()
		var candPaths []string
		if osType == "darwin" && strings.EqualFold(browser, string(consts.Safari)) {
			profiles = []string{""}
		} else if osType == "windows" || osType == "darwin" {
			// 扫描候选 cookie 文件以推断可用的 profile 名称
			// 我们根据路径推断出 Profile 名称（Default/Profile X）
			candPaths = listCandidateCookiePaths(consts.BrowserType(browser), homeDir)
			var profSet = map[string]bool{}
			for _, p := range candPaths {
				var prof string
				if osType == "windows" {
					prof = deriveChromiumProfile(p)
				} else {
					// macOS: .../Chrome/<Profile>/(Network/)?Cookies
					// 取上两级目录作为 Profile 名称
					dir := filepath.Dir(p) // .../<Profile>/(Network)
					// 若包含 Network，再上一级
					base := filepath.Base(dir)
					if base == "Network" {
						prof = filepath.Base(filepath.Dir(dir))
					} else {
						prof = base
					}
				}
				prof = strings.TrimSpace(prof)
				if prof != "" && !profSet[prof] {
					profSet[prof] = true
				}
			}
			// 将 profSet 写回 profiles，若未发现任何 profile，则回退到空 Profile
			for k := range profSet {
				profiles = append(profiles, k)
			}
			if len(profiles) == 0 {
				profiles = []string{""}
			}
		} else {
			// 其他系统按旧逻辑：使用空 Profile 以兼容默认配置
			profiles = []string{""}
		}
		// Do not print profile names derived from disk scan; just count
		logger.Debug("ytdlp export: profiles",
			zap.String("browser", browser),
			zap.Int("profileCount", len(profiles)),
		)

		// 遍历 profile 进行导出。成功的合并到同一个 BrowserCookies
		// map CanMe browser name -> yt-dlp expected name (lowercase)
		toYtDlp := func(name string) string {
			n := strings.ToLower(strings.TrimSpace(name))
			switch n {
			case "chrome":
				return "chrome"
			case "chromium":
				return "chromium"
			case "edge":
				return "edge"
			case "firefox":
				return "firefox"
			case "safari":
				return "safari"
			default:
				return n
			}
		}

		for _, prof := range profiles {
			dlEach := ytdlp.New().Verbose()
			dlEach.SetExecutable(ytdlpExec)
			base := toYtDlp(browser)
			browserSpec := base
			if prof != "" {
				browserSpec = fmt.Sprintf("%s:%s", base, prof)
			}
			cookiePath := filepath.Join(ytdlpDir, fmt.Sprintf("export_%s_%s_cookies.txt", base, strings.ReplaceAll(prof, " ", "_")))
			dlEach.CookiesFromBrowser(browserSpec)
			dlEach.Cookies(cookiePath)
			// Windows: 显式传递关键环境变量，确保 yt-dlp 在相同用户上下文读取浏览器配置
			if runtime.GOOS == "windows" {
				for _, key := range []string{"LOCALAPPDATA", "APPDATA", "USERPROFILE", "HOMEDRIVE", "HOMEPATH"} {
					if val := os.Getenv(key); val != "" {
						dlEach.SetEnvVar(key, val)
					}
				}
			}
			logger.Debug("ytdlp export: running",
				zap.String("browserSpec", browserSpec),
				zap.String("out", cookiePath),
			)
			result, rerr := dlEach.Run(ctx)
			if rerr != nil {
				logger.Warn("ytdlp export: run error",
					zap.String("browserSpec", browserSpec),
					zap.String("out", cookiePath),
					zap.Error(rerr),
				)
				if result != nil && result.Stderr != "" {
					msg := result.Stderr
					if len(msg) > 800 {
						msg = msg[:800] + "..."
					}
					logger.Debug("ytdlp export: stderr", zap.String("stderr", msg))
					// 提取形如 "ERROR: ..." 的首行作为用户可见错误
					line := strings.SplitN(result.Stderr, "\n", 2)[0]
					if idx := strings.Index(line, "ERROR:"); idx >= 0 {
						short := strings.TrimSpace(line[idx:]) // 从 ERROR: 开始
						// 去掉路径部分（例如 ": '/Users/...'")
						cut := strings.Index(short, ": '")
						if cut == -1 {
							cut = strings.Index(short, ": /")
						}
						if cut > 0 {
							short = strings.TrimSpace(strings.TrimSuffix(short[:cut], ":"))
						}
						if short != "" {
							lastErrMsg = short
						}
					}
				}
				if lastErrMsg == "" {
					// 退化：从 error 串提要
					em := rerr.Error()
					if i := strings.Index(em, "ERROR:"); i >= 0 {
						short := strings.TrimSpace(em[i:])
						if j := strings.Index(short, "\n"); j > 0 {
							short = short[:j]
						}
						if k := strings.Index(short, ": '"); k > 0 {
							short = strings.TrimSpace(strings.TrimSuffix(short[:k], ":"))
						}
						lastErrMsg = short
					} else {
						lastErrMsg = em
					}
				}
				// 针对 macOS Safari 权限受限的友好提示
				lowerErr := strings.ToLower(rerr.Error())
				var lowerStderr string
				if result != nil {
					lowerStderr = strings.ToLower(result.Stderr)
				}
				if runtime.GOOS == "darwin" && strings.EqualFold(base, "safari") && (strings.Contains(lowerErr, "operation not permitted") || strings.Contains(lowerStderr, "operation not permitted")) {
					// 不中断提取的错误文案，但保留提示供日志排查
					if lastErrMsg == "" {
						lastErrMsg = "ERROR: Operation not permitted"
					}
				}
			} else if result != nil {
				logger.Debug("ytdlp export: done",
					zap.Int("exit", result.ExitCode),
				)
			}
			if _, osErr := os.Stat(cookiePath); osErr != nil {
				// 该 profile 失败，尝试下一个
				logger.Warn("ytdlp export: output not found",
					zap.String("browserSpec", browserSpec),
					zap.String("out", cookiePath),
					zap.Error(osErr),
				)
				continue
			}
			if cookiesPaths[browser] == nil {
				cookiesPaths[browser] = &agg{}
			}
			cookiesPaths[browser].paths = append(cookiesPaths[browser].paths, cookiePath)
		}

		// 记录错误摘要（若有）以便在后续合并阶段展示
		if strings.TrimSpace(lastErrMsg) != "" {
			errMsgByBrowser[browser] = lastErrMsg
		}

		browserCookiesMap[string(browser)] = &eachBrowserCookies
	}

	// 使用c.convertFromNetscape()转换为http.Cookie（合并多 profile 的导出）
	for browser, item := range cookiesPaths {
		var merged = make(map[string]*types.DomainCookies)
		var ok bool
		for _, cookiePath := range item.paths {
			// Skip logging concrete cookie path to avoid printing scanned filenames
			domainCookies, err := c.convertFromNetscape(cookiePath)
			if err != nil {
				logger.Warn("ytdlp export: convert failed", zap.String("path", cookiePath), zap.Error(err))
				continue
			}
			if domainCookies != nil {
				ok = true
				for d, dc := range domainCookies {
					if merged[d] == nil {
						merged[d] = &types.DomainCookies{Domain: d, Cookies: []*http.Cookie{}}
					}
					merged[d].Cookies = append(merged[d].Cookies, dc.Cookies...)
				}
			}
		}
		if ok {
			logger.Debug("ytdlp export: merged domains",
				zap.String("browser", browser),
				zap.Int("domains", len(merged)),
			)
			browserCookiesMap[browser].Status = "synced"
			browserCookiesMap[browser].StatusDescription = "cookies synced"
			browserCookiesMap[browser].LastSyncTime = time.Now()
			browserCookiesMap[browser].LastSyncStatus = "success"
			browserCookiesMap[browser].DomainCookies = merged
		} else {
			logger.Warn("ytdlp export: no cookies found after export",
				zap.String("browser", browser),
				zap.Int("attempts", len(item.paths)),
			)
			browserCookiesMap[browser].Status = "error"
			// 优先显示 yt-dlp 的 ERROR 首行，作为用户可见的失败原因
			if msg := strings.TrimSpace(errMsgByBrowser[browser]); msg != "" {
				browserCookiesMap[browser].StatusDescription = msg
			} else {
				browserCookiesMap[browser].StatusDescription = "no cookies found"
			}
			browserCookiesMap[browser].LastSyncTime = time.Now()
			browserCookiesMap[browser].LastSyncStatus = "failed"
		}
	}

	// 对于未出现在 cookiesPaths 的浏览器，标记为失败，避免状态停留在 "syncing"
	for b, bc := range browserCookiesMap {
		if _, ok := cookiesPaths[b]; !ok {
			if bc != nil && strings.EqualFold(bc.LastSyncStatus, "syncing") {
				bc.Status = "error"
				if msg := strings.TrimSpace(errMsgByBrowser[b]); msg != "" {
					bc.StatusDescription = msg
				} else {
					bc.StatusDescription = "no cookies found"
				}
				bc.LastSyncStatus = "failed"
				bc.LastSyncTime = time.Now()
			}
		}
	}

	// 删除对应的export_[browser]_cookies.txt文件
	for _, item := range cookiesPaths {
		for _, cookiePath := range item.paths {
			if err := os.Remove(cookiePath); err != nil {
				logger.Error("Failed to remove export cookies file", zap.String("path", cookiePath), zap.Error(err))
			}
		}
	}

	return browserCookiesMap
}

func (c *cookieManager) convertToNetscape(cookies []*http.Cookie) string {
	var b strings.Builder
	b.WriteString("# Netscape HTTP Cookie File\n\n")

	for _, cookie := range cookies {
		flag := "FALSE"
		if strings.HasPrefix(cookie.Domain, ".") {
			flag = "TRUE"
		}

		secure := "FALSE"
		if cookie.Secure {
			secure = "TRUE"
		}

		var expiration int64
		if !cookie.Expires.IsZero() {
			expiration = cookie.Expires.Unix()
		}

		line := fmt.Sprintf(
			"%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			cookie.Domain,
			flag,
			cookie.Path,
			secure,
			expiration,
			cookie.Name,
			cookie.Value,
		)
		b.WriteString(line)
	}

	return b.String()
}

// convertFromNetscape parses a Netscape cookie format string and returns []*http.Cookie
func (c *cookieManager) convertFromNetscape(netscapeData string) (map[string]*types.DomainCookies, error) {
	byte, err := os.ReadFile(netscapeData)
	if err != nil {
		return nil, err
	}

	domainCookies := make(map[string]*types.DomainCookies)
	lines := strings.Split(string(byte), "\n")

	for _, line := range lines {
		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split by tab characters
		fields := strings.Split(line, "\t")
		if len(fields) != 7 {
			continue // Skip malformed lines
		}

		domain := fields[0]
		// flag := fields[1]
		path := fields[2]
		secure := fields[3]
		expirationStr := fields[4]
		name := fields[5]
		value := fields[6]

		// Create http.Cookie
		cookie := &http.Cookie{
			Name:   name,
			Value:  value,
			Domain: domain,
			Path:   path,
		}

		// Set secure flag
		if secure == "TRUE" {
			cookie.Secure = true
		}

		// Parse expiration time
		if expirationStr != "0" {
			if expiration, err := strconv.ParseInt(expirationStr, 10, 64); err == nil {
				// 验证时间戳范围，避免超出 Go 时间范围
				// Unix 时间戳范围：1970-01-01 到 2038-01-19 (32位) 或更大范围 (64位)
				// 但 Go 的 time.Time JSON 序列化要求年份在 [0,9999] 范围内
				minTimestamp := int64(0)            // 1970-01-01
				maxTimestamp := int64(253402300799) // 9999-12-31 23:59:59 UTC

				if expiration >= minTimestamp && expiration <= maxTimestamp {
					cookie.Expires = time.Unix(expiration, 0)
				} else {
					// 可设置为零值（永不过期）
					cookie.Expires = time.Time{} // 零值表示会话 cookie
				}
			}
		}

		if domainCookies[domain] == nil {
			domainCookies[domain] = &types.DomainCookies{
				Domain:  domain,
				Cookies: []*http.Cookie{},
			}
		} else {
			domainCookies[domain].Cookies = append(domainCookies[domain].Cookies, cookie)
		}

	}

	return domainCookies, nil
}

func (c *cookieManager) generateDomainLookups(hostname string) []string {
	// 1. 完全匹配是第一优先级
	lookups := []string{hostname}

	registrableDomain, err := publicsuffix.EffectiveTLDPlusOne(hostname)
	if err == nil && registrableDomain != "" {
		lookups = append(lookups, "."+registrableDomain)
	} else {
		logger.Debug("Failed to get registrable domain", zap.String("hostname", hostname), zap.Error(err))
	}

	return lookups
}

// kooky removed

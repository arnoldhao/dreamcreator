package browercookies

import (
	"CanMe/backend/pkg/logger"
	"CanMe/backend/storage"
	"CanMe/backend/types"
	"CanMe/backend/consts"
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
	"github.com/reveever/gocookie"
	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"
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

	browserCookiesMap := make(map[string]*types.BrowserCookies)
	if syncFrom == "canme" {
		browserCookiesMap = c.readAllBrowserCookiesByGoCookie(ctx, browsers)

	} else if syncFrom == "yt-dlp" {
		browserCookiesMap = c.readAllBrowserCookiesByYTDLP(ctx, browsers)
	} else {
		return fmt.Errorf("unsupported syncFrom: %s", syncFrom)
	}

	if len(browserCookiesMap) == 0 {
		return fmt.Errorf("no browser cookies found")
	}

	for browser, cookies := range browserCookiesMap {
		if browser != "" && cookies != nil {
			err := c.storage.SaveCookies(browser, cookies)
			if err != nil {
				logger.GetLogger().Error("Failed to save cookies for browser %s: %v", zap.String("browser", browser), zap.Error(err))
			}
		}
	}

	return nil
}

// ListAllCookies retrieves all cached cookies, grouped by browser.
func (c *cookieManager) ListAllCookies() (map[string]*types.BrowserCookies, error) {
	if c.storage == nil {
		return nil, errors.New("storage is not initialized")
	}

	// available type to get cookies
	syncFrom := []string{"yt-dlp"}
	osType := runtime.GOOS
	if osType == "darwin" {
		syncFrom = append(syncFrom, "canme")
	}

	var currentBrowsers []*types.BrowserCookies
	allBrowsers := []consts.BrowserType{consts.Chrome, consts.Chromium, consts.Firefox, consts.Edge, consts.Safari}
	for _, browser := range allBrowsers {
		cookiePath := GetCookieFilePath(browser)
		if _, err := os.Stat(cookiePath); err == nil {
			// supportedBrowsers = append(supportedBrowsers, string(browser))
			currentBrowsers = append(currentBrowsers, &types.BrowserCookies{
				Browser:           string(browser),
				Path:              cookiePath,
				Status:            "never",
				StatusDescription: "never sync cookies",
				SyncFrom:          syncFrom,
				LastSyncFrom:      "never",
				LastSyncTime:      time.Time{},
				LastSyncStatus:    "never syncd",
				DomainCookies:     nil,
			})
		} else {
			logger.GetLogger().Info("Failed to stat cookie file for browser", zap.String("browser", string(browser)), zap.Any("path", cookiePath), zap.Error(err))
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
func (c *cookieManager) readAllBrowserCookiesByGoCookie(ctx context.Context, browsers []string) map[string]*types.BrowserCookies {
	browserCookiesMap := make(map[string]*types.BrowserCookies)

	// available type to get cookies
	syncFrom := []string{"yt-dlp"}
	osType := runtime.GOOS
	if osType == "darwin" {
		syncFrom = append(syncFrom, "canme")
	}

	for _, browser := range browsers {
		eachBrowserCookies := types.BrowserCookies{
			Browser:           browser,
			Path:              gocookie.GetCookieFilePath(gocookie.BrowserType(browser)),
			Status:            "syncing",
			StatusDescription: "syncing cookies",
			SyncFrom:          syncFrom,
			LastSyncFrom:      "canme",
			LastSyncTime:      time.Time{},
			LastSyncStatus:    "syncing",
			DomainCookies:     make(map[string]*types.DomainCookies),
		}

		gocookies, err := gocookie.GetCookies(gocookie.BrowserType(browser))
		if err != nil {
			logger.GetLogger().Error("Failed to extract cookies using gocookie: ", zap.Any("browser", string(browser)), zap.Error(err))
			continue
		} else if len(gocookies) > 0 {
			// 重新针对domain进行分组
			for _, cookie := range gocookies {
				if eachBrowserCookies.DomainCookies[cookie.Domain] == nil {
					eachBrowserCookies.DomainCookies[cookie.Domain] = &types.DomainCookies{
						Domain:  cookie.Domain,
						Cookies: []*http.Cookie{},
					}
				}

				// fill info
				eachBrowserCookies.Status = "synced"
				eachBrowserCookies.StatusDescription = "cookies synced"
				eachBrowserCookies.LastSyncTime = time.Now()
				eachBrowserCookies.LastSyncStatus = "success"

				eachBrowserCookies.DomainCookies[cookie.Domain].Cookies = append(eachBrowserCookies.DomainCookies[cookie.Domain].Cookies, cookie)
			}
		} else {
			// 没有发现cookies,填充信息
			eachBrowserCookies.Status = "error"
			eachBrowserCookies.StatusDescription = "no cookies found"
			eachBrowserCookies.LastSyncTime = time.Now()
			eachBrowserCookies.LastSyncStatus = "failed"
		}

		browserCookiesMap[string(browser)] = &eachBrowserCookies
	}

	return browserCookiesMap
}

func (c *cookieManager) readAllBrowserCookiesByYTDLP(ctx context.Context, browsers []string) map[string]*types.BrowserCookies {
	browserCookiesMap := make(map[string]*types.BrowserCookies)

	// available type to get cookies
	syncFrom := []string{"yt-dlp"}
	osType := runtime.GOOS
	if osType == "darwin" {
		syncFrom = append(syncFrom, "canme")
	}

	// 遍历使用yt-dlp导出cookies为 export_[browser]_cookies.txt
	dl := ytdlp.New()
	ytdlpPath, err := c.YTDLPExecPath(ctx)
	if err != nil {
		logger.GetLogger().Error("Failed to get ytdlp path: ", zap.Error(err))
		return browserCookiesMap
	}
	dl.SetExecutable(ytdlpPath)

	cookiesPaths := make(map[string]string)
	for _, browser := range browsers {
		eachBrowserCookies := types.BrowserCookies{
			Browser:           browser,
			Path:              gocookie.GetCookieFilePath(gocookie.BrowserType(browser)),
			Status:            "syncing",
			StatusDescription: "syncing cookies",
			SyncFrom:          syncFrom,
			LastSyncFrom:      "yt-dlp",
			LastSyncTime:      time.Time{},
			LastSyncStatus:    "syncing",
			DomainCookies:     make(map[string]*types.DomainCookies),
		}

		ytdlpPath, err := c.YTDLPPath(ctx)
		if err != nil {
			eachBrowserCookies.Status = "error"
			eachBrowserCookies.StatusDescription = err.Error()
			eachBrowserCookies.LastSyncTime = time.Now()
			eachBrowserCookies.LastSyncStatus = "failed"

			// save and continue
			browserCookiesMap[string(browser)] = &eachBrowserCookies
			continue
		}

		cookiePath := filepath.Join(ytdlpPath, fmt.Sprintf("export_%s_cookies.txt", browser))
		dl.CookiesFromBrowser(browser)
		dl.Cookies(cookiePath)

		_, err = dl.Run(ctx)
		// 因为yt-dlp的命令一定会报错，这里只检查有没有生成文件，如果有生成则通过，没有生成则填充yt-dlp的错误信息
		if _, osErr := os.Stat(cookiePath); osErr != nil {
			eachBrowserCookies.Status = "error"
			eachBrowserCookies.StatusDescription = err.Error()
			eachBrowserCookies.LastSyncTime = time.Now()
			eachBrowserCookies.LastSyncStatus = "failed"

			// save and continue
			browserCookiesMap[string(browser)] = &eachBrowserCookies
			continue
		}

		browserCookiesMap[string(browser)] = &eachBrowserCookies
		cookiesPaths[browser] = cookiePath
	}

	if len(browserCookiesMap) == 0 {
		// 删除对应的export_[browser]_cookies.txt文件
		for _, cookiePath := range cookiesPaths {
			err := os.Remove(cookiePath)
			if err != nil {
				logger.GetLogger().Error("Failed to remove export cookies file: ", zap.Error(err))
			}
		}
		return browserCookiesMap
	}

	// 使用c.convertFromNetscape()转换为http.Cookie
	for browser, cookiePath := range cookiesPaths {
		domainCookies, err := c.convertFromNetscape(cookiePath)
		if err != nil {
			browserCookiesMap[browser].Status = "error"
			browserCookiesMap[browser].StatusDescription = err.Error()
			browserCookiesMap[browser].LastSyncTime = time.Now()
			browserCookiesMap[browser].LastSyncStatus = "failed"
			continue
		}

		if domainCookies != nil {
			browserCookiesMap[browser].Status = "synced"
			browserCookiesMap[browser].StatusDescription = "cookies synced"
			browserCookiesMap[browser].LastSyncTime = time.Now()
			browserCookiesMap[browser].LastSyncStatus = "success"
			browserCookiesMap[browser].DomainCookies = domainCookies
		} else {
			browserCookiesMap[browser].Status = "error"
			browserCookiesMap[browser].StatusDescription = "no cookies found"
			browserCookiesMap[browser].LastSyncTime = time.Now()
			browserCookiesMap[browser].LastSyncStatus = "failed"
		}
	}

	// 删除对应的export_[browser]_cookies.txt文件
	for _, cookiePath := range cookiesPaths {
		err := os.Remove(cookiePath)
		if err != nil {
			logger.GetLogger().Error("Failed to remove export cookies file: ", zap.Error(err))
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
		logger.GetLogger().Info("Failed to get registrable domain: ", zap.String("hostname", hostname), zap.Error(err))
	}

	return lookups
}

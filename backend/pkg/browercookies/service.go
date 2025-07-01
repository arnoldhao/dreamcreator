package browercookies

import (
	"CanMe/backend/pkg/logger"
	"CanMe/backend/storage"
	"CanMe/backend/types"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/reveever/gocookie"
	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"
)

type cookieManager struct {
	storage *storage.BoltStorage
}

func NewCookieManager(storage *storage.BoltStorage) CookieManager {
	return &cookieManager{
		storage: storage,
	}
}

func (c *cookieManager) Sync() (map[string]*types.BrowserCookies, error) {
	if c.storage == nil {
		return nil, errors.New("storage is not initialized")
	}

	browserCookiesMap := c.readAllBrowserCookies()
	if len(browserCookiesMap) == 0 {
		return nil, fmt.Errorf("no browser cookies found")
	}

	for browser, cookies := range browserCookiesMap {
		if browser != "" && cookies != nil {
			err := c.storage.SaveCookies(browser, cookies)
			if err != nil {
				logger.GetLogger().Error("Failed to save cookies for browser %s: %v", zap.String("browser", browser), zap.Error(err))
			}
		}
	}

	return browserCookiesMap, nil
}

// ListAllCookies retrieves all cached cookies, grouped by browser.
func (c *cookieManager) ListAllCookies() (map[string]*types.BrowserCookies, error) {
	if c.storage == nil {
		return nil, errors.New("storage is not initialized")
	}

	return c.storage.ListAllCookies()
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
func (c *cookieManager) readAllBrowserCookies() map[string]*types.BrowserCookies {
	browserCookiesMap := make(map[string]*types.BrowserCookies)

	// gocookie.ExtractCookies() finds all cookies from all supported browsers
	supportedBrowsers := []gocookie.BrowserType{gocookie.Chrome, gocookie.Firefox, gocookie.Edge, gocookie.Safari}
	for _, browser := range supportedBrowsers {
		eachBrowserCookies := types.BrowserCookies{
			Browser:       string(browser),
			LastSyncTime:  time.Now(),
			DomainCookies: make(map[string]*types.DomainCookies),
		}

		gocookies, err := gocookie.GetCookies(browser)
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
				eachBrowserCookies.DomainCookies[cookie.Domain].Cookies = append(eachBrowserCookies.DomainCookies[cookie.Domain].Cookies, cookie)
			}

			browserCookiesMap[string(browser)] = &eachBrowserCookies
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

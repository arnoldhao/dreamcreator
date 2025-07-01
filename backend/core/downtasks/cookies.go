package downtasks

import (
	"CanMe/backend/types"
	"fmt"
	"net/http"
)

// SyncCookies refreshes the browser cookies.
func (s *Service) SyncCookies() (map[string]*types.BrowserCookies, error) {
	return s.cookieManager.Sync()
}

// ListAllCookies lists all cached cookies, grouped by browser.
func (s *Service) ListAllCookies() (map[string]*types.BrowserCookies, error) {
	return s.cookieManager.ListAllCookies()
}

func (s *Service) GetBrowserByDomain(domain string) ([]string, error) {
	// 如果为短链接，获取真正地址
	if s.proxyManager != nil {
		// head
		resp, err := s.proxyManager.GetHTTPClient().Head(domain)
		if err != nil {
			return nil, err
		}
		// 获取重定向后的url
		pureDomain := resp.Request.URL.Hostname()
		if pureDomain == "" {
			return nil, fmt.Errorf("domain must be specified: %s", domain)
		}

		return s.cookieManager.GetBrowserByDomain(pureDomain)
	} else {
		return nil, fmt.Errorf("proxy not initialized, domain: %s", domain)
	}
}

// GetCookiesByDomain retrieves cookies for a specific URL from a specific browser.
func (s *Service) GetCookiesByDomain(browser, domain string) ([]*http.Cookie, error) {
	// 如果为短链接，获取真正地址
	if s.proxyManager != nil {
		// head
		resp, err := s.proxyManager.GetHTTPClient().Head(domain)
		if err != nil {
			return nil, err
		}
		// 获取重定向后的url
		pureDomain := resp.Request.URL.Hostname()
		if pureDomain == "" {
			return nil, fmt.Errorf("domain must be specified: %s", domain)
		}

		return s.cookieManager.GetCookiesByDomain(browser, pureDomain)
	} else {
		return nil, fmt.Errorf("proxy not initialized, domain: %s", domain)
	}
}

// GetNetscapeCookiesByDomain retrieves cookies for a specific URL from a specific browser.
func (s *Service) GetNetscapeCookiesByDomain(browser, domain string) (string, error) {
	// 如果为短链接，获取真正地址
	if s.proxyManager != nil {
		// head
		resp, err := s.proxyManager.GetHTTPClient().Head(domain)
		if err != nil {
			return "", err
		}
		// 获取重定向后的url
		pureDomain := resp.Request.URL.Hostname()
		if pureDomain == "" {
			return "nil", fmt.Errorf("domain must be specified: %s", domain)
		}

		return s.cookieManager.GetNetscapeCookiesByDomain(browser, pureDomain)
	} else {
		return "", fmt.Errorf("proxy not initialized, domain: %s", domain)
	}
}

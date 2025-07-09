package downtasks

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/events"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/types"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// SyncCookies refreshes the browser cookies.
func (s *Service) SyncCookies(syncFrom string, browsers []string) {
	logger.GetLogger().Debug("Starting async browser cookies sync...")

	go func() {
		// 发送开始同步事件
		startEvent := &events.BaseEvent{
			ID:        uuid.New().String(),
			Type:      consts.TopicDowntasksCookieSync,
			Source:    "downtasks",
			Timestamp: time.Now(),
			Data: &types.DTCookieSync{
				SyncFrom:  syncFrom,
				Browsers:  browsers,
				Status:    types.CookieSyncStatusStarted,
				Done:      false,
				Error:     "",
				Timestamp: time.Now().Unix(),
			},
		}

		if s.eventBus != nil {
			s.eventBus.Publish(s.ctx, startEvent)
		}

		// 执行同步操作
		err := s.cookieManager.Sync(s.ctx, syncFrom, browsers)

		// 准备结果事件
		var status types.DTCookieSyncStatus
		var errMessage string
		if err == nil {
			status = types.CookieSyncStatusSuccess
			errMessage = ""
		} else {
			status = types.CookieSyncStatusFailed
			errMessage = err.Error()
		}

		resultEvent := &events.BaseEvent{
			ID:        uuid.New().String(),
			Type:      consts.TopicDowntasksCookieSync,
			Source:    "downtasks",
			Timestamp: time.Now(),
			Data: &types.DTCookieSync{
				SyncFrom:  syncFrom,
				Browsers:  browsers,
				Status:    status,
				Done:      true,
				Error:     errMessage,
				Timestamp: time.Now().Unix(),
			},
		}

		// 通过事件总线发送结果
		if s.eventBus != nil {
			s.eventBus.Publish(s.ctx, resultEvent)
		}
	}()
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

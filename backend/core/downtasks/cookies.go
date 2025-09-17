package downtasks

import (
    "CanMe/backend/consts"
    "CanMe/backend/pkg/events"
    "CanMe/backend/pkg/logger"
    "CanMe/backend/types"
    "context"
    "fmt"
    "net/http"
    neturl "net/url"
    "strings"
    "time"

    "github.com/google/uuid"
)

// SyncCookies refreshes the browser cookies.
func (s *Service) SyncCookies(syncFrom string, browsers []string) {
    logger.Debug("Starting async browser cookies sync...")

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

func (s *Service) GetBrowserByDomain(targetURL string) ([]string, error) {
    if s.proxyManager == nil {
        return nil, fmt.Errorf("proxy not initialized, domain: %s", targetURL)
    }
    pureDomain := s.resolveDomainWithFallback(targetURL)
    if pureDomain == "" {
        return nil, fmt.Errorf("domain must be specified: %s", targetURL)
    }
    return s.cookieManager.GetBrowserByDomain(pureDomain)
}

// GetCookiesByDomain retrieves cookies for a specific URL from a specific browser.
func (s *Service) GetCookiesByDomain(browser, targetURL string) ([]*http.Cookie, error) {
    if s.proxyManager == nil {
        return nil, fmt.Errorf("proxy not initialized, domain: %s", targetURL)
    }
    pureDomain := s.resolveDomainWithFallback(targetURL)
    if pureDomain == "" {
        return nil, fmt.Errorf("domain must be specified: %s", targetURL)
    }
    return s.cookieManager.GetCookiesByDomain(browser, pureDomain)
}

// GetNetscapeCookiesByDomain retrieves cookies for a specific URL from a specific browser.
func (s *Service) GetNetscapeCookiesByDomain(browser, targetURL string) (string, error) {
    if s.proxyManager == nil {
        return "", fmt.Errorf("proxy not initialized, domain: %s", targetURL)
    }
    pureDomain := s.resolveDomainWithFallback(targetURL)
    if pureDomain == "" {
        return "", fmt.Errorf("domain must be specified: %s", targetURL)
    }
    return s.cookieManager.GetNetscapeCookiesByDomain(browser, pureDomain)
}

// resolveDomainWithFallback 尽量避免网络阻塞：
// 1) 优先本地解析 URL 获取 hostname
// 2) 若命中已知短链域名（如 youtu.be/b23.tv/t.co/bit.ly），再做一个 5s 超时的 HEAD 解析跳转
// 3) HEAD 失败则回退：对常见短链做静态映射（youtu.be->youtube.com，b23.tv->bilibili.com），否则返回原 hostname
func (s *Service) resolveDomainWithFallback(rawURL string) string {
    u, err := neturl.Parse(strings.TrimSpace(rawURL))
    if err == nil && u.Hostname() != "" {
        host := u.Hostname()
        // 已知短链域名列表
        shorteners := map[string]string{
            "youtu.be":  "youtube.com",
            "b23.tv":     "bilibili.com",
            "t.co":       "twitter.com",
            "bit.ly":     "",
            "tinyurl.com":"",
        }
        if _, ok := shorteners[strings.ToLower(host)]; !ok {
            return host
        }

        // 对短链尝试一次短超时 HEAD 解析
        ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
        defer cancel()
        req, _ := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
        resp, err := s.proxyManager.GetHTTPClient().Do(req)
        if err == nil && resp != nil && resp.Request != nil && resp.Request.URL != nil && resp.Request.URL.Hostname() != "" {
            return resp.Request.URL.Hostname()
        }

        // 回退到静态映射
        if mapped := shorteners[strings.ToLower(host)]; mapped != "" {
            return mapped
        }
        return host
    }
    return ""
}

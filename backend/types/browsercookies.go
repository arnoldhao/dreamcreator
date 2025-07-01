package types

import (
	"net/http"
	"time"
)

type BrowserCookies struct {
	Browser       string                    `json:"browser"`
	LastSyncTime  time.Time                 `json:"last_sync_time"`
	DomainCookies map[string]*DomainCookies `json:"domain_cookies"`
}

type DomainCookies struct {
	Domain  string         `json:"domain"`
	Cookies []*http.Cookie `json:"cookies"`
}

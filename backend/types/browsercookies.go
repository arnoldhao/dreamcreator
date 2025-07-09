package types

import (
	"net/http"
	"time"
)

// type BrowserCookies struct {
// 	Browser       string                    `json:"browser"`
// 	LastSyncFrom  string                    `json:"last_sync_from"`
// 	LastSyncTime  time.Time                 `json:"last_sync_time"`
// 	DomainCookies map[string]*DomainCookies `json:"domain_cookies"`
// }

type BrowserCookies struct {
	Browser           string                    `json:"browser"`
	Path              string                    `json:"path"`
	Status            string                    `json:"status"` // e.g. "synced", "never", "syncing", "error"
	StatusDescription string                    `json:"status_description"`
	SyncFrom          []string                  `json:"sync_from"` // e.g. ["canme", "yt-dlp"]
	LastSyncFrom      string                    `json:"last_sync_from"`
	LastSyncTime      time.Time                 `json:"last_sync_time"`
	LastSyncStatus    string                    `json:"last_sync_status"` // e.g. "success", "failed"
	DomainCookies     map[string]*DomainCookies `json:"domain_cookies"`
}

type DomainCookies struct {
	Domain  string         `json:"domain"`
	Cookies []*http.Cookie `json:"cookies"`
}

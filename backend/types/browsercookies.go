package types

import (
	"net/http"
	"time"
)

type CookieSource string

const (
	CookieSourceYTDLP  CookieSource = "yt-dlp"
	CookieSourceManual CookieSource = "manual"
)

type CookieCollection struct {
	ID                string                    `json:"id"`
	Name              string                    `json:"name"`
	Browser           string                    `json:"browser"`
	Profile           string                    `json:"profile"`
	Path              string                    `json:"path"`
	Source            CookieSource              `json:"source"`
	Status            string                    `json:"status"`
	StatusDescription string                    `json:"status_description"`
	SyncFrom          []string                  `json:"sync_from"`
	LastSyncFrom      string                    `json:"last_sync_from"`
	LastSyncTime      time.Time                 `json:"last_sync_time"`
	LastSyncStatus    string                    `json:"last_sync_status"`
	DomainCookies     map[string]*DomainCookies `json:"domain_cookies"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
}

type CookieCollections struct {
	BrowserCollections []*CookieCollection `json:"browser_collections"`
	ManualCollections  []*CookieCollection `json:"manual_collections"`
}

type CookieProvider struct {
	ID     string       `json:"id"`
	Label  string       `json:"label"`
	Source CookieSource `json:"source"`
	Kind   string       `json:"kind,omitempty"`
}

type ManualCookieInput struct {
	Domain   string `json:"domain"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path"`
	Secure   bool   `json:"secure"`
	HTTPOnly bool   `json:"http_only"`
	Expires  int64  `json:"expires"`
}

type ManualCollectionPayload struct {
	Name     string              `json:"name"`
	Netscape string              `json:"netscape"`
	Cookies  []ManualCookieInput `json:"cookies"`
	Replace  bool                `json:"replace"`
}

type DomainCookies struct {
	Domain  string         `json:"domain"`
	Cookies []*http.Cookie `json:"cookies"`
}

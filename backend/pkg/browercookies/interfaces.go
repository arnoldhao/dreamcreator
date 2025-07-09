package browercookies

import (
	"CanMe/backend/types"
	"context"
	"net/http"
)

// CookieManager defines the interface for managing browser cookies.
type CookieManager interface {
	// Sync scans all supported browsers and updates the cookie cache.
	Sync(ctx context.Context, syncFrom string, browser []string) error
	// ListAllCookies returns all cookies from the cache, grouped by browser.
	ListAllCookies() (map[string]*types.BrowserCookies, error)
	// GetBrowserByDomain returns the browser name for a given domain.
	GetBrowserByDomain(domain string) ([]string, error)
	// GetCookiesByDomain returns a slice of cookies for a given domain from a specific browser.
	GetCookiesByDomain(browser, domain string) ([]*http.Cookie, error)
	// GetNetscapeCookiesByDomain returns a Netscape-formatted string of cookies for a given domain from a specific browser.
	GetNetscapeCookiesByDomain(browser, domain string) (string, error)
}

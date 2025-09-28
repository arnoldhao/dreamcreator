package browercookies

import (
	"context"
	"dreamcreator/backend/types"
	"net/http"
)

// CookieManager defines the interface for managing browser cookies.
type CookieManager interface {
	// Sync scans all supported browsers and updates the cookie cache.
	Sync(ctx context.Context, syncFrom string, browser []string) error
	// ListAllCookies returns all cookie collections from the cache.
	ListAllCookies() (*types.CookieCollections, error)
	// GetBrowserByDomain returns cookie providers (browser or manual collections) for a given domain.
	GetBrowserByDomain(domain string) ([]*types.CookieProvider, error)
	// GetCookiesByDomain returns cookies for a given domain from a specific provider (browser or manual collection).
	GetCookiesByDomain(providerID, domain string) ([]*http.Cookie, error)
	// GetNetscapeCookiesByDomain returns a Netscape-formatted string of cookies for a given domain from a specific provider.
	GetNetscapeCookiesByDomain(providerID, domain string) (string, error)
	// CreateManualCollection stores a new manual cookie collection.
	CreateManualCollection(payload *types.ManualCollectionPayload) (*types.CookieCollection, error)
	// UpdateManualCollection updates an existing manual cookie collection.
	UpdateManualCollection(id string, payload *types.ManualCollectionPayload) (*types.CookieCollection, error)
	// DeleteCollection removes a cookie collection (manual or browser).
	DeleteCollection(id string) error
	// ExportCollectionNetscape exports a collection as Netscape cookies text.
	ExportCollectionNetscape(id string) (string, error)
}

package extractors

import (
	"net/url"
	"strings"
	"sync"

	"CanMe/backend/types"
	"CanMe/backend/utils/domainUtil"

	"github.com/pkg/errors"
)

// define error
var (
	ErrUnsupportedDomain = errors.New("unsupported domain")
	ErrInvalidURL        = errors.New("invalid URL")
)

// define bilibili
var bilibiliURLMap = map[string]string{
	"av": "https://www.bilibili.com/video/",
	"BV": "https://www.bilibili.com/video/",
	"ep": "https://www.bilibili.com/bangumi/play/",
}

// Registry abstract extractor registry
type Registry struct {
	sync.RWMutex
	extractors map[string]Extractor
}

// NewRegistry creates a new registry
func NewRegistry() *Registry {
	return &Registry{
		extractors: make(map[string]Extractor),
	}
}

// Register registers an extractor
func (r *Registry) Register(domain string, e Extractor) {
	r.Lock()
	defer r.Unlock()
	r.extractors[domain] = e
}

// Get retrieves an extractor
func (r *Registry) Get(domain string) (Extractor, bool) {
	r.RLock()
	defer r.RUnlock()
	e, ok := r.extractors[domain]
	return e, ok
}

// global registry instance
var defaultRegistry = NewRegistry()

// Register convenience method for registering extractors
func Register(domain string, e Extractor) {
	defaultRegistry.Register(domain, e)
}

// parseBilibiliShortLink handles bilibili short links
func parseBilibiliShortLink(rawURL string) (string, bool) {
	matches := domainUtil.MatchOneOf(rawURL, `^(av|BV|ep)\w+`)
	if len(matches) > 1 {
		if baseURL, ok := bilibiliURLMap[matches[1]]; ok {
			return baseURL + rawURL, true
		}
	}
	return rawURL, false
}

// getDomain gets the domain
// for the future version
func getDomain(u *url.URL) string {
	switch u.Host {
	case "haokan.baidu.com":
		return "haokan"
	case "xhslink.com":
		return "xiaohongshu"
	default:
		return domainUtil.Domain(u.Host)
	}
}

// Extract main extraction function
func Extract(rawURL string, option types.ExtractorOptions) ([]*types.ExtractorData, error) {
	// Preprocess URL
	processedURL := strings.TrimSpace(rawURL)
	if processedURL == "" {
		return nil, ErrInvalidURL
	}

	// Handle bilibili short links
	if finalURL, ok := parseBilibiliShortLink(processedURL); ok {
		processedURL = finalURL
	}

	// Parse URL
	parsedURL, err := url.ParseRequestURI(processedURL)
	if err != nil {
		return nil, errors.Wrap(ErrInvalidURL, err.Error())
	}

	// Get domain
	domain := getDomain(parsedURL)

	// Get extractor
	extractor, ok := defaultRegistry.Get(domain)
	if !ok {
		return nil, errors.Wrapf(ErrUnsupportedDomain, "domain: %s", domain)
	}

	// Extract video data
	videos, err := extractor.Extract(processedURL, option)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract from %s", processedURL)
	}

	// check data error
	if len(videos) > 0 {
		for _, video := range videos {
			if video.Err != nil {
				return nil, errors.Wrapf(video.Err, "failed to extract from %s", processedURL)
			} else {
				video.FillUpStreamsData()
			}
		}
	} else {
		return nil, errors.Wrapf(ErrInvalidURL, "current url: %s do not have any video data", processedURL)
	}

	return videos, nil
}

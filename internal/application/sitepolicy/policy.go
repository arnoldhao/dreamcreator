package sitepolicy

import (
	"net/url"
	"strings"
)

type Policy struct {
	Key                string
	ConnectorType      string
	Domains            []string
	ReadySelectors     []string
	ExtractorSelectors []string
	RemoveSelectors    []string
	Capabilities       []string
}

var builtinPolicyOrder = []string{
	"youtube",
	"google",
	"github",
	"reddit",
	"zhihu",
	"x",
	"xiaohongshu",
	"bilibili",
}

var builtinPolicies = map[string]Policy{
	"youtube": {
		Key:           "youtube",
		ConnectorType: "google",
		Domains: []string{
			"youtube.com",
			"youtu.be",
		},
		ReadySelectors: []string{
			"ytd-watch-flexy",
			"#content",
			"main",
			"body",
		},
		ExtractorSelectors: []string{
			"#description",
			"#description-inline-expander",
			"ytd-watch-metadata",
			"main",
		},
		RemoveSelectors: []string{
			"#related",
			"ytd-comments",
			"ytd-merch-shelf-renderer",
			"ytd-rich-grid-renderer",
		},
		Capabilities: []string{"cookies", "web_fetch", "browser", "download"},
	},
	"google": {
		Key:           "google",
		ConnectorType: "google",
		Domains: []string{
			"google.com",
			"youtube.com",
			"youtu.be",
		},
		ReadySelectors: []string{
			"#search",
			"main",
			"body",
		},
		ExtractorSelectors: []string{
			"main",
			"article",
			"#content",
		},
		RemoveSelectors: []string{
			"#related",
			"#secondary",
			"ytd-comments",
		},
		Capabilities: []string{"cookies", "web_fetch", "browser", "download"},
	},
	"github": {
		Key:           "github",
		ConnectorType: "github",
		Domains: []string{
			"github.com",
			"raw.githubusercontent.com",
		},
		ReadySelectors: []string{
			"main",
			"#repo-content-pjax-container",
			"body",
		},
		ExtractorSelectors: []string{
			"main",
			"article",
			".markdown-body",
			"[data-testid=\"issue-body\"]",
			"[data-testid=\"pull-request-comment\"]",
		},
		RemoveSelectors: []string{
			"header",
			"footer",
			".Layout-sidebar",
			".js-header-wrapper",
			"#repos-sticky-header",
		},
		Capabilities: []string{"cookies", "web_fetch", "browser"},
	},
	"reddit": {
		Key:           "reddit",
		ConnectorType: "reddit",
		Domains: []string{
			"reddit.com",
			"redd.it",
		},
		ReadySelectors: []string{
			"main",
			"[data-testid=\"post-container\"]",
			"body",
		},
		ExtractorSelectors: []string{
			"main",
			"article",
			"[data-testid=\"post-container\"]",
			".md",
		},
		RemoveSelectors: []string{
			"nav",
			"[data-testid=\"frontpage-sidebar\"]",
			"shreddit-comments-page-ad",
			"shreddit-experience-tree",
		},
		Capabilities: []string{"cookies", "web_fetch", "browser"},
	},
	"zhihu": {
		Key:           "zhihu",
		ConnectorType: "zhihu",
		Domains: []string{
			"zhihu.com",
		},
		ReadySelectors: []string{
			"main",
			".Question-main",
			".Post-content",
			"body",
		},
		ExtractorSelectors: []string{
			"main",
			".Question-mainColumn",
			".Post-RichTextContainer",
			"article",
		},
		RemoveSelectors: []string{
			".Question-sideColumn",
			".CornerButtons",
			".Recommendations-Main",
			".Comment-container",
		},
		Capabilities: []string{"cookies", "web_fetch", "browser"},
	},
	"x": {
		Key:           "x",
		ConnectorType: "x",
		Domains: []string{
			"x.com",
			"twitter.com",
		},
		ReadySelectors: []string{
			"main",
			"[data-testid=\"primaryColumn\"]",
			"body",
		},
		ExtractorSelectors: []string{
			"main",
			"article",
			"[data-testid=\"tweet\"]",
		},
		RemoveSelectors: []string{
			"nav",
			"[data-testid=\"sidebarColumn\"]",
			"[aria-label=\"Timeline: Trending now\"]",
			"[aria-label=\"Who to follow\"]",
		},
		Capabilities: []string{"cookies", "web_fetch", "browser"},
	},
	"xiaohongshu": {
		Key:           "xiaohongshu",
		ConnectorType: "xiaohongshu",
		Domains: []string{
			"xiaohongshu.com",
			"xhslink.com",
			"redbook.com",
		},
		ReadySelectors: []string{
			"#app",
			"main",
			"body",
		},
		ExtractorSelectors: []string{
			"main",
			"article",
			"#noteContainer",
		},
		RemoveSelectors: []string{
			".note-side-bar",
			".recommend-container",
			".comment-container",
		},
		Capabilities: []string{"cookies", "web_fetch", "browser"},
	},
	"bilibili": {
		Key:           "bilibili",
		ConnectorType: "bilibili",
		Domains: []string{
			"bilibili.com",
			"b23.tv",
		},
		ReadySelectors: []string{
			"#app",
			"#arc_toolbar_report",
			"main",
			"body",
		},
		ExtractorSelectors: []string{
			"main",
			"article",
			"#app",
		},
		RemoveSelectors: []string{
			".video-toolbar-v1",
			".right-container",
			".comment-container",
		},
		Capabilities: []string{"cookies", "web_fetch", "browser", "download"},
	},
}

func List() []Policy {
	result := make([]Policy, 0, len(builtinPolicyOrder))
	for _, key := range builtinPolicyOrder {
		policy, ok := builtinPolicies[key]
		if !ok {
			continue
		}
		result = append(result, policy)
	}
	return result
}

func ForConnectorType(connectorType string) (Policy, bool) {
	policy, ok := builtinPolicies[strings.ToLower(strings.TrimSpace(connectorType))]
	return policy, ok
}

func ForURL(rawURL string) (Policy, bool) {
	host := hostname(rawURL)
	if host == "" {
		return Policy{}, false
	}
	for _, key := range builtinPolicyOrder {
		policy, ok := builtinPolicies[key]
		if !ok {
			continue
		}
		for _, domain := range policy.Domains {
			if HostMatchesDomain(host, domain) {
				return policy, true
			}
		}
	}
	return Policy{}, false
}

func DomainsForConnector(connectorType string) []string {
	policy, ok := ForConnectorType(connectorType)
	if !ok {
		return nil
	}
	return cloneStrings(policy.Domains)
}

func ReadySelectorForURL(rawURL string) string {
	policy, ok := ForURL(rawURL)
	if !ok {
		return ""
	}
	for _, selector := range policy.ReadySelectors {
		if strings.TrimSpace(selector) != "" {
			return strings.TrimSpace(selector)
		}
	}
	return ""
}

func HostMatchesDomain(host string, domain string) bool {
	normalizedHost := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(host)), ".")
	normalizedDomain := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), ".")
	if normalizedHost == "" || normalizedDomain == "" {
		return false
	}
	return normalizedHost == normalizedDomain || strings.HasSuffix(normalizedHost, "."+normalizedDomain)
}

func MatchDomains(rawURL string, domains []string) bool {
	host := hostname(rawURL)
	if host == "" {
		return false
	}
	for _, domain := range domains {
		if HostMatchesDomain(host, domain) {
			return true
		}
	}
	return false
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func hostname(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parsed.Hostname()))
}

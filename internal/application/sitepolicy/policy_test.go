package sitepolicy

import "testing"

func TestForURLPrefersYouTubePolicyBeforeGoogle(t *testing.T) {
	t.Parallel()

	policy, ok := ForURL("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if !ok {
		t.Fatalf("expected youtube policy match")
	}
	if policy.Key != "youtube" {
		t.Fatalf("expected youtube policy key, got %q", policy.Key)
	}
	if policy.ConnectorType != "google" {
		t.Fatalf("expected youtube URLs to reuse google connector cookies, got %q", policy.ConnectorType)
	}
}

func TestForConnectorTypeGoogleDomainsIncludeYouTube(t *testing.T) {
	t.Parallel()

	policy, ok := ForConnectorType("google")
	if !ok {
		t.Fatalf("expected google connector policy")
	}
	if !MatchDomains("https://www.youtube.com/watch?v=test", policy.Domains) {
		t.Fatalf("expected google connector domains to cover youtube URLs")
	}
}

func TestForURLMatchesNewBuiltinSites(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"https://github.com/owner/repo":                 "github",
		"https://www.reddit.com/r/golang/comments/test": "reddit",
		"https://www.zhihu.com/question/123456":         "zhihu",
		"https://x.com/example/status/1":                "x",
		"https://www.xiaohongshu.com/explore/abc":       "xiaohongshu",
		"https://www.bilibili.com/video/BV1xx411c7mD/":  "bilibili",
	}

	for rawURL, expected := range cases {
		policy, ok := ForURL(rawURL)
		if !ok {
			t.Fatalf("expected policy match for %s", rawURL)
		}
		if policy.Key != expected {
			t.Fatalf("expected policy %q for %s, got %q", expected, rawURL, policy.Key)
		}
	}
}

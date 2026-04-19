package settings

const DefaultBrowserColor = "#FF4500"

func DefaultCallsToolsConfig() map[string]any {
	defaultWebFetch := map[string]any{
		"headless":         true,
		"preferredBrowser": "chrome",
	}
	defaultBrowser := map[string]any{
		"enabled":          true,
		"headless":         true,
		"preferredBrowser": "chrome",
		"ssrfPolicy": map[string]any{
			"dangerouslyAllowPrivateNetwork": false,
		},
	}
	return map[string]any{
		"web_fetch": cloneAnyMap(defaultWebFetch),
		"browser":   cloneAnyMap(defaultBrowser),
		"web": map[string]any{
			"search": map[string]any{
				"type":           "api",
				"external_tools": map[string]any{},
				"providers": map[string]any{
					"brave": map[string]any{
						"label":      "Brave",
						"apiBaseUrl": "https://api.search.brave.com/res/v1/web/search",
					},
					"perplexity": map[string]any{
						"label":             "Perplexity",
						"apiBaseUrl":        "https://api.perplexity.ai",
						"openRouterBaseUrl": "https://openrouter.ai/api/v1",
					},
					"grok": map[string]any{
						"label":      "Grok",
						"apiBaseUrl": "https://api.x.ai/v1/responses",
					},
					"tavily": map[string]any{
						"label":      "Tavily",
						"apiBaseUrl": "https://api.tavily.com/search",
					},
				},
			},
		},
	}
}

func normalizeToolsConfig(source map[string]any) map[string]any {
	if len(source) == 0 {
		return cloneAnyMap(DefaultCallsToolsConfig())
	}
	result := cloneAnyMap(source)
	ensureDefaultWebSearchProviders(result)
	return result
}

func cloneAnyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func ensureDefaultWebSearchProviders(target map[string]any) {
	if target == nil {
		return
	}
	root := target
	if nestedRaw, ok := target["tools"].(map[string]any); ok && nestedRaw != nil {
		nestedCopy := cloneAnyMap(nestedRaw)
		target["tools"] = nestedCopy
		root = nestedCopy
	}
	webRaw, ok := root["web"].(map[string]any)
	if ok && webRaw != nil {
		webRaw = cloneAnyMap(webRaw)
	}
	web := webRaw
	if web == nil {
		web = map[string]any{}
	}
	root["web"] = web
	ensureDefaultWebFetchConfig(root)
	ensureDefaultBrowserConfig(root)
	searchRaw, ok := web["search"].(map[string]any)
	if ok && searchRaw != nil {
		searchRaw = cloneAnyMap(searchRaw)
	}
	search := searchRaw
	if search == nil {
		search = map[string]any{}
	}
	web["search"] = search
	if _, exists := search["type"]; !exists {
		search["type"] = "api"
	}
	if _, exists := search["external_tools"]; !exists {
		search["external_tools"] = map[string]any{}
	}
	providersRaw, ok := search["providers"].(map[string]any)
	if ok && providersRaw != nil {
		providersRaw = cloneAnyMap(providersRaw)
	}
	providers := providersRaw
	if providers == nil {
		providers = map[string]any{}
	}
	defaults := DefaultCallsToolsConfig()
	defaultWeb, ok := defaults["web"].(map[string]any)
	if !ok {
		return
	}
	defaultSearch, ok := defaultWeb["search"].(map[string]any)
	if !ok {
		return
	}
	defaultProviders, ok := defaultSearch["providers"].(map[string]any)
	if !ok {
		return
	}
	for key, value := range defaultProviders {
		if _, exists := providers[key]; !exists {
			providers[key] = value
		}
	}
	search["providers"] = providers
}

func ensureDefaultWebFetchConfig(root map[string]any) {
	if root == nil {
		return
	}
	fetchRaw, ok := root["web_fetch"].(map[string]any)
	if ok && fetchRaw != nil {
		fetchRaw = cloneAnyMap(fetchRaw)
	}
	fetch := fetchRaw
	if fetch == nil {
		fetch = map[string]any{}
	}
	defaults := DefaultCallsToolsConfig()
	defaultFetch, ok := defaults["web_fetch"].(map[string]any)
	if !ok {
		root["web_fetch"] = fetch
		return
	}
	for key, value := range defaultFetch {
		if _, exists := fetch[key]; !exists {
			fetch[key] = value
		}
	}
	root["web_fetch"] = fetch
}

func ensureDefaultBrowserConfig(root map[string]any) {
	if root == nil {
		return
	}
	defaults := DefaultCallsToolsConfig()
	defaultBrowser, ok := defaults["browser"].(map[string]any)
	if !ok || defaultBrowser == nil {
		return
	}
	browserRaw, ok := root["browser"].(map[string]any)
	if ok && browserRaw != nil {
		browserRaw = cloneAnyMap(browserRaw)
	}
	browser := browserRaw
	if browser == nil {
		browser = map[string]any{}
	}
	for key, value := range defaultBrowser {
		if _, exists := browser[key]; !exists {
			browser[key] = value
		}
	}

	defaultSSRFRaw, ok := defaultBrowser["ssrfPolicy"].(map[string]any)
	if ok && defaultSSRFRaw != nil {
		ssrfRaw, ok := browser["ssrfPolicy"].(map[string]any)
		if ok && ssrfRaw != nil {
			ssrfRaw = cloneAnyMap(ssrfRaw)
		}
		ssrf := ssrfRaw
		if ssrf == nil {
			ssrf = map[string]any{}
		}
		for key, value := range defaultSSRFRaw {
			if _, exists := ssrf[key]; !exists {
				ssrf[key] = value
			}
		}
		browser["ssrfPolicy"] = ssrf
	}

	root["browser"] = browser
}

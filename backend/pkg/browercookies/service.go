package browercookies

import (
	"context"
	"dreamcreator/backend/consts"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/storage"
	"dreamcreator/backend/types"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"dreamcreator/backend/pkg/dependencies"

	"github.com/google/uuid"
	"github.com/lrstanley/go-ytdlp"
	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"

	"dreamcreator/backend/utils"
)

type cookieManager struct {
	storage    *storage.BoltStorage
	depManager dependencies.Manager
}

const defaultManualCollectionBaseName = "Custom Set"

func browserCollectionID(browser string) string {
	return "browser:" + strings.ToLower(strings.TrimSpace(browser))
}

func NewCookieManager(storage *storage.BoltStorage, depManager dependencies.Manager) CookieManager {
	return &cookieManager{
		storage:    storage,
		depManager: depManager,
	}
}

func (c *cookieManager) Sync(ctx context.Context, syncFrom string, browsers []string) error {
	if c.storage == nil {
		return errors.New("storage is not initialized")
	}

	logger.Debug("cookies.Sync start",
		zap.String("from", syncFrom),
		zap.Strings("browsers", browsers),
		zap.String("os", runtime.GOOS),
	)

	collections := make(map[string]*types.CookieCollection)
	if syncFrom == "dreamcreator" {
		return fmt.Errorf("dreamcreator cookies sync is not supported; please use yt-dlp")
	} else if syncFrom == "yt-dlp" {
		collections = c.readAllBrowserCollectionsByYTDLP(ctx, browsers)
	} else {
		return fmt.Errorf("unsupported syncFrom: %s", syncFrom)
	}

	if len(collections) == 0 {
		return fmt.Errorf("no browser cookies found")
	}

	// 统计成功/失败，用于整体结果判断
	anySuccess := false
	var firstErrMsg string

	for browser, collection := range collections {
		if browser != "" && collection != nil {
			if collection.ID == "" {
				collection.ID = browserCollectionID(browser)
			}
			collection.Source = types.CookieSourceYTDLP
			collection.Browser = browser
			if collection.Name == "" {
				collection.Name = browser
			}

			err := c.storage.SaveCookieCollection(collection)
			if err != nil {
				logger.Error("Failed to save cookies for browser", zap.String("browser", browser), zap.Error(err))
			}
			if strings.EqualFold(collection.LastSyncStatus, "success") {
				anySuccess = true
			} else if firstErrMsg == "" {
				firstErrMsg = collection.StatusDescription
			}
		}
	}

	if anySuccess {
		logger.Debug("cookies.Sync complete", zap.Bool("success", true))
		return nil
	}
	if firstErrMsg == "" {
		firstErrMsg = "no cookies found"
	}
	logger.Warn("cookies.Sync complete", zap.Bool("success", false), zap.String("error", firstErrMsg))
	return errors.New(firstErrMsg)
}

// ListAllCookies 返回所有 Cookie 集合，包含浏览器与手动来源。
func (c *cookieManager) ListAllCookies() (*types.CookieCollections, error) {
	if c.storage == nil {
		return nil, errors.New("storage is not initialized")
	}

	storedCollections, err := c.storage.ListCookieCollections()
	if err != nil {
		return nil, err
	}

	manualCollections := make([]*types.CookieCollection, 0)
	storedByBrowser := make(map[string]*types.CookieCollection)

	for _, collection := range storedCollections {
		if collection == nil {
			continue
		}
		switch collection.Source {
		case types.CookieSourceManual:
			manualCollections = append(manualCollections, collection)
		case types.CookieSourceYTDLP, "":
			key := strings.ToLower(collection.Browser)
			storedByBrowser[key] = collection
		}
	}

	osType := runtime.GOOS
	allBrowsers := []consts.BrowserType{
		consts.Chrome,
		consts.Chromium,
		consts.Edge,
		consts.Firefox,
		consts.Safari,
		consts.Brave,
		consts.Opera,
		consts.Vivaldi,
	}

	browserCollections := make([]*types.CookieCollection, 0, len(allBrowsers))

	for _, browser := range allBrowsers {
		browserName := string(browser)
		key := strings.ToLower(browserName)

		homeDir, _ := os.UserHomeDir()
		paths := listCandidateCookiePaths(browser, homeDir)
		var firstPath string
		if len(paths) > 0 {
			firstPath = paths[0]
		} else {
			p := GetCookieFilePath(browser)
			if _, err := os.Stat(p); err == nil {
				firstPath = p
			}
		}

		existing := storedByBrowser[key]
		if existing != nil {
			if existing.ID == "" {
				existing.ID = browserCollectionID(browserName)
			}
			if existing.Name == "" {
				existing.Name = browserName
			}
			if existing.Source == "" {
				existing.Source = types.CookieSourceYTDLP
			}
			if existing.Browser == "" {
				existing.Browser = browserName
			}
			if firstPath != "" && existing.Path == "" {
				existing.Path = firstPath
			}
			if len(existing.SyncFrom) == 0 {
				existing.SyncFrom = []string{string(types.CookieSourceYTDLP)}
			}
			browserCollections = append(browserCollections, existing)
			continue
		}

		if firstPath == "" {
			logger.Debug("No cookie store detected", zap.String("browser", browserName))
			continue
		}

		browserCollections = append(browserCollections, &types.CookieCollection{
			ID:                browserCollectionID(browserName),
			Name:              browserName,
			Browser:           browserName,
			Path:              firstPath,
			Source:            types.CookieSourceYTDLP,
			Status:            "never",
			StatusDescription: "never sync cookies",
			SyncFrom:          []string{string(types.CookieSourceYTDLP)},
			LastSyncFrom:      "never",
			LastSyncTime:      time.Time{},
			LastSyncStatus:    "never synced",
			DomainCookies:     nil,
		})
	}

	if osType == "windows" && utils.WindowsIsElevated() {
		for _, bc := range browserCollections {
			if bc != nil {
				if bc.Status == "never" || bc.StatusDescription == "" {
					bc.StatusDescription = "Running as Administrator may block cookie decryption. Please run as a normal user."
				}
			}
		}
	}

	return &types.CookieCollections{
		BrowserCollections: browserCollections,
		ManualCollections:  manualCollections,
	}, nil
}

func (c *cookieManager) GetBrowserByDomain(domain string) ([]*types.CookieProvider, error) {
	if c.storage == nil {
		return nil, errors.New("storage is not initialized")
	}
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return nil, errors.New("domain must be specified")
	}

	storedCollections, err := c.storage.ListCookieCollections()
	if err != nil {
		return nil, err
	}

	lookups := c.generateDomainLookups(domain)
	providers := make([]*types.CookieProvider, 0)
	seen := make(map[string]struct{})

	for _, collection := range storedCollections {
		if collection == nil {
			continue
		}
		if !c.collectionMatchesDomain(collection, lookups) {
			continue
		}

		switch collection.Source {
		case types.CookieSourceManual:
			label := strings.TrimSpace(collection.Name)
			if label == "" {
				label = "Manual Cookies"
			}
			providers = append(providers, &types.CookieProvider{
				ID:     collection.ID,
				Label:  label,
				Source: types.CookieSourceManual,
				Kind:   "manual",
			})
		case types.CookieSourceYTDLP, "":
			browserName := strings.TrimSpace(collection.Browser)
			if browserName == "" {
				continue
			}
			key := strings.ToLower(browserName)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			label := browserName
			if strings.TrimSpace(collection.Name) != "" {
				label = collection.Name
			}
			providers = append(providers, &types.CookieProvider{
				ID:     browserName,
				Label:  label,
				Source: types.CookieSourceYTDLP,
				Kind:   "browser",
			})
		}
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no cookie provider found for domain: %s", domain)
	}

	return providers, nil
}

// GetCookiesByDomain retrieves cookies for a specific domain from a specified provider's cache.
func (c *cookieManager) GetCookiesByDomain(providerID, domain string) ([]*http.Cookie, error) {
	if c.storage == nil {
		return nil, errors.New("storage is not initialized")
	}
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, errors.New("provider must be specified")
	}
	if strings.TrimSpace(domain) == "" {
		return nil, errors.New("domain must be specified")
	}

	domainLookups := c.generateDomainLookups(domain)

	if strings.HasPrefix(providerID, string(types.CookieSourceManual)+":") {
		collection, err := c.storage.GetCookieCollection(providerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get manual collection %s: %w", providerID, err)
		}
		if collection == nil || collection.Source != types.CookieSourceManual {
			return nil, fmt.Errorf("collection %s is not manual", providerID)
		}
		return c.extractCookiesForLookups(collection, domainLookups), nil
	}

	storedCollection, err := c.storage.FindCookieCollectionByBrowser(providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cookies for browser %s: %w", providerID, err)
	}

	return c.extractCookiesForLookups(storedCollection, domainLookups), nil
}

func (c *cookieManager) GetNetscapeCookiesByDomain(browser, domain string) (string, error) {
	if c.storage == nil {
		return "", errors.New("storage is not initialized")
	}
	if browser == "" {
		return "", errors.New("provider must be specified")
	}
	if domain == "" {
		return "", errors.New("domain must be specified")
	}

	cookies, err := c.GetCookiesByDomain(browser, domain)
	if err != nil {
		return "", err
	}

	return c.convertToNetscape(cookies), nil
}

func (c *cookieManager) collectionMatchesDomain(collection *types.CookieCollection, lookups []string) bool {
	if collection == nil {
		return false
	}
	return len(c.extractCookiesForLookups(collection, lookups)) > 0
}

func (c *cookieManager) extractCookiesForLookups(collection *types.CookieCollection, lookups []string) []*http.Cookie {
	type candidate struct {
		cookie   *http.Cookie
		priority int
		order    int
	}

	if collection == nil || collection.DomainCookies == nil || len(collection.DomainCookies) == 0 {
		return nil
	}

	lookupPriorities := make(map[string]int, len(lookups))
	preferredDomains := make(map[int]string, len(lookups))
	for idx, lookup := range lookups {
		norm := c.normalizeDomainKey(lookup)
		if norm == "" {
			continue
		}
		if _, exists := lookupPriorities[norm]; !exists {
			lookupPriorities[norm] = idx
			preferredDomains[idx] = norm
		}
	}

	if len(lookupPriorities) == 0 {
		return nil
	}

	candidates := make(map[string]candidate)
	orderSeed := 0

	for key, bucket := range collection.DomainCookies {
		if bucket == nil || len(bucket.Cookies) == 0 {
			continue
		}

		normKey := c.normalizeDomainKey(key)
		priority, ok := lookupPriorities[normKey]
		if !ok {
			continue
		}

		preferredDomain := preferredDomains[priority]

		for _, cookie := range bucket.Cookies {
			if cookie == nil {
				continue
			}

			mapKey := strings.ToLower(cookie.Name) + "|" + cookie.Path
			existing, exists := candidates[mapKey]
			replace := false

			if !exists {
				replace = true
				orderSeed++
			} else if priority < existing.priority {
				replace = true
			} else if priority == existing.priority {
				if preferCookieForDomain(cookie, existing.cookie, preferredDomain) {
					replace = true
				}
			}

			if replace {
				order := existing.order
				if !exists {
					order = orderSeed
				}
				candidates[mapKey] = candidate{cookie: cookie, priority: priority, order: order}
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	items := make([]candidate, 0, len(candidates))
	for _, cand := range candidates {
		items = append(items, cand)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].priority != items[j].priority {
			return items[i].priority < items[j].priority
		}
		return items[i].order < items[j].order
	})

	result := make([]*http.Cookie, 0, len(items))
	for _, cand := range items {
		result = append(result, cand.cookie)
	}

	return result
}

func (c *cookieManager) normalizeDomainKey(domain string) string {
	trimmed := strings.TrimSpace(strings.ToLower(domain))
	trimmed = strings.TrimPrefix(trimmed, ".")
	return trimmed
}

func preferCookieForDomain(newCookie, oldCookie *http.Cookie, preferredDomain string) bool {
	if newCookie == nil {
		return false
	}
	if oldCookie == nil {
		return true
	}

	newDomain := sanitizeCookieDomain(newCookie.Domain)
	oldDomain := sanitizeCookieDomain(oldCookie.Domain)

	if preferredDomain != "" {
		if newDomain == preferredDomain && oldDomain != preferredDomain {
			return true
		}
		if oldDomain == preferredDomain && newDomain != preferredDomain {
			return false
		}
	}

	newHostOnly := cookieIsHostOnly(newCookie.Domain)
	oldHostOnly := cookieIsHostOnly(oldCookie.Domain)
	if newHostOnly && !oldHostOnly {
		return true
	}
	if oldHostOnly && !newHostOnly {
		return false
	}

	if !newCookie.Expires.IsZero() && oldCookie.Expires.IsZero() {
		return true
	}
	if newCookie.Expires.IsZero() && !oldCookie.Expires.IsZero() {
		return false
	}
	if newCookie.Expires.After(oldCookie.Expires) {
		return true
	}
	if oldCookie.Expires.After(newCookie.Expires) {
		return false
	}

	// 默认使用最新值覆盖，避免返回旧的 profile 数据
	return true
}

func sanitizeCookieDomain(domain string) string {
	d := strings.ToLower(strings.TrimSpace(domain))
	d = strings.TrimPrefix(d, "#httponly_")
	return strings.TrimPrefix(d, ".")
}

func cookieIsHostOnly(domain string) bool {
	d := strings.TrimSpace(domain)
	if d == "" {
		return false
	}
	return !strings.HasPrefix(d, ".")
}

func (c *cookieManager) CreateManualCollection(payload *types.ManualCollectionPayload) (*types.CookieCollection, error) {
	if c.storage == nil {
		return nil, errors.New("storage is not initialized")
	}
	if payload == nil {
		return nil, errors.New("payload is required")
	}

	domainCookies, err := manualPayloadToDomainCookies(payload)
	if err != nil {
		return nil, err
	}

	if len(domainCookies) == 0 {
		return nil, errors.New("no cookies provided")
	}

	id := "manual:" + uuid.NewString()
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		var err error
		name, err = c.generateDefaultManualCollectionName("")
		if err != nil {
			return nil, err
		}
	} else {
		if err := c.ensureUniqueManualCollectionName(name, ""); err != nil {
			return nil, err
		}
	}

	collection := &types.CookieCollection{
		ID:                id,
		Name:              name,
		Browser:           "",
		Path:              "",
		Source:            types.CookieSourceManual,
		Status:            "manual",
		StatusDescription: "manual collection",
		SyncFrom:          []string{string(types.CookieSourceManual)},
		LastSyncFrom:      string(types.CookieSourceManual),
		LastSyncTime:      time.Now(),
		LastSyncStatus:    "manual",
		DomainCookies:     domainCookies,
	}

	if err := c.storage.SaveCookieCollection(collection); err != nil {
		return nil, err
	}

	return collection, nil
}

func (c *cookieManager) UpdateManualCollection(id string, payload *types.ManualCollectionPayload) (*types.CookieCollection, error) {
	if c.storage == nil {
		return nil, errors.New("storage is not initialized")
	}
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("collection id is required")
	}

	collection, err := c.storage.GetCookieCollection(id)
	if err != nil {
		return nil, err
	}
	if collection == nil || collection.Source != types.CookieSourceManual {
		return nil, fmt.Errorf("collection %s is not manual", id)
	}

	if payload != nil {
		if nameRaw := payload.Name; nameRaw != "" {
			trimmed := strings.TrimSpace(nameRaw)
			if trimmed == "" {
				generated, err := c.generateDefaultManualCollectionName(collection.ID)
				if err != nil {
					return nil, err
				}
				collection.Name = generated
			} else {
				if err := c.ensureUniqueManualCollectionName(trimmed, collection.ID); err != nil {
					return nil, err
				}
				collection.Name = trimmed
			}
		}
		domainCookies, err := manualPayloadToDomainCookies(payload)
		if err != nil {
			return nil, err
		}
		if len(domainCookies) > 0 {
			if payload.Replace {
				collection.DomainCookies = domainCookies
			} else {
				collection.DomainCookies = mergeDomainCookies(collection.DomainCookies, domainCookies)
			}
		}
	}

	if strings.TrimSpace(collection.Name) == "" {
		generated, err := c.generateDefaultManualCollectionName(collection.ID)
		if err != nil {
			return nil, err
		}
		collection.Name = generated
	}

	if err := c.ensureUniqueManualCollectionName(collection.Name, collection.ID); err != nil {
		return nil, err
	}

	collection.Status = "manual"
	collection.StatusDescription = "manual collection"
	collection.LastSyncFrom = string(types.CookieSourceManual)
	collection.LastSyncStatus = "manual"
	collection.LastSyncTime = time.Now()

	if err := c.storage.SaveCookieCollection(collection); err != nil {
		return nil, err
	}

	return collection, nil
}

func (c *cookieManager) DeleteCollection(id string) error {
	if c.storage == nil {
		return errors.New("storage is not initialized")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("collection id is required")
	}
	return c.storage.DeleteCookieCollection(id)
}

func (c *cookieManager) ensureUniqueManualCollectionName(name string, excludeID string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("collection name cannot be empty")
	}
	collections, err := c.storage.ListCookieCollections()
	if err != nil {
		return err
	}
	lowerName := strings.ToLower(trimmed)
	for _, col := range collections {
		if col == nil {
			continue
		}
		if excludeID != "" && strings.EqualFold(col.ID, excludeID) {
			continue
		}
		existingName := strings.TrimSpace(col.Name)
		if existingName == "" {
			continue
		}
		if strings.ToLower(existingName) == lowerName {
			return fmt.Errorf("collection name already exists")
		}
	}
	return nil
}

func (c *cookieManager) generateDefaultManualCollectionName(excludeID string) (string, error) {
	collections, err := c.storage.ListCookieCollections()
	if err != nil {
		return "", err
	}
	base := defaultManualCollectionBaseName
	existing := make(map[string]struct{})
	for _, col := range collections {
		if col == nil {
			continue
		}
		if excludeID != "" && strings.EqualFold(col.ID, excludeID) {
			continue
		}
		name := strings.TrimSpace(col.Name)
		if name == "" {
			continue
		}
		existing[strings.ToLower(name)] = struct{}{}
	}

	candidate := base
	index := 2
	for {
		if _, ok := existing[strings.ToLower(candidate)]; !ok {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s %d", base, index)
		index++
	}
}

func (c *cookieManager) ExportCollectionNetscape(id string) (string, error) {
	if c.storage == nil {
		return "", errors.New("storage is not initialized")
	}
	if strings.TrimSpace(id) == "" {
		return "", errors.New("collection id is required")
	}

	collection, err := c.storage.GetCookieCollection(id)
	if err != nil {
		return "", err
	}
	if collection == nil {
		return "", fmt.Errorf("collection not found: %s", id)
	}

	var all []*http.Cookie
	for _, dc := range collection.DomainCookies {
		if dc == nil {
			continue
		}
		all = append(all, dc.Cookies...)
	}

	return c.convertToNetscape(all), nil
}

// readAllBrowserCollectionsByYTDLP 使用 yt-dlp 导出浏览器 cookies。
func (c *cookieManager) readAllBrowserCollectionsByYTDLP(ctx context.Context, browsers []string) map[string]*types.CookieCollection {
	browserCookiesMap := make(map[string]*types.CookieCollection)

	// available type to get cookies: yt-dlp only
	syncFrom := []string{"yt-dlp"}
	osType := runtime.GOOS

	// 遍历使用yt-dlp导出cookies为 export_[browser]_cookies.txt
	dl := ytdlp.New()
	ytdlpExec, err := c.YTDLPExecPath(ctx)
	if err != nil {
		logger.Error("Failed to get ytdlp path", zap.Error(err))
		return browserCookiesMap
	}
	dl.SetExecutable(ytdlpExec)
	logger.Debug("ytdlp export: using executable", zap.String("exec", ytdlpExec))

	type agg struct {
		paths []string
	}
	cookiesPaths := make(map[string]*agg) // 浏览器 => 导出文件路径列表
	// 聚合每个浏览器的首要错误信息（来自 yt-dlp 的 stderr）
	errMsgByBrowser := make(map[string]string)

	for _, browser := range browsers {
		// 每个浏览器独立收集错误摘要
		var lastErrMsg string
		// Representative path for UI
		btype := consts.BrowserType(browser)
		homeDir, _ := os.UserHomeDir()
		var pathHint string
		var cands []string
		if paths := listCandidateCookiePaths(btype, homeDir); len(paths) > 0 {
			cands = paths
			pathHint = paths[0]
		} else {
			pathHint = GetCookieFilePath(btype)
		}
		// Avoid printing scanned paths; only log counts/summary
		logger.Debug("ytdlp export: detected cookie locations",
			zap.String("browser", string(btype)),
			zap.Int("candidates", len(cands)),
		)

		eachBrowserCookies := &types.CookieCollection{
			ID:                browserCollectionID(browser),
			Name:              browser,
			Browser:           browser,
			Path:              pathHint,
			Source:            types.CookieSourceYTDLP,
			Status:            "syncing",
			StatusDescription: "syncing cookies",
			SyncFrom:          syncFrom,
			LastSyncFrom:      "yt-dlp",
			LastSyncTime:      time.Time{},
			LastSyncStatus:    "syncing",
			DomainCookies:     make(map[string]*types.DomainCookies),
		}

		ytdlpDir, err := c.YTDLPPath(ctx)
		if err != nil {
			eachBrowserCookies.Status = "error"
			eachBrowserCookies.StatusDescription = err.Error()
			eachBrowserCookies.LastSyncTime = time.Now()
			eachBrowserCookies.LastSyncStatus = "failed"

			// save and continue
			browserCookiesMap[string(browser)] = eachBrowserCookies
			continue
		}
		// 针对 Windows/macOS，尝试多 Profile（Default, Profile *）
		// 注：若已发现明确的 Profile，则不再追加空 Profile，避免重复导出导致多次系统弹窗（macOS 钥匙串）
		// Safari（macOS）只有单一存储，yt-dlp 不接受 safari:<profile> 形式，这里强制仅用空 Profile
		var profiles []string
		homeDir, _ = os.UserHomeDir()
		var candPaths []string
		if osType == "darwin" && strings.EqualFold(browser, string(consts.Safari)) {
			profiles = []string{""}
		} else if osType == "windows" || osType == "darwin" {
			// 扫描候选 cookie 文件以推断可用的 profile 名称
			// 我们根据路径推断出 Profile 名称（Default/Profile X）
			candPaths = listCandidateCookiePaths(consts.BrowserType(browser), homeDir)
			var profSet = map[string]bool{}
			for _, p := range candPaths {
				var prof string
				if osType == "windows" {
					prof = deriveChromiumProfile(p)
				} else {
					// macOS: .../Chrome/<Profile>/(Network/)?Cookies
					// 取上两级目录作为 Profile 名称
					dir := filepath.Dir(p) // .../<Profile>/(Network)
					// 若包含 Network，再上一级
					base := filepath.Base(dir)
					if base == "Network" {
						prof = filepath.Base(filepath.Dir(dir))
					} else {
						prof = base
					}
				}
				prof = strings.TrimSpace(prof)
				if prof != "" && !profSet[prof] {
					profSet[prof] = true
				}
			}
			// 将 profSet 写回 profiles，若未发现任何 profile，则回退到空 Profile
			for k := range profSet {
				profiles = append(profiles, k)
			}
			if len(profiles) == 0 {
				profiles = []string{""}
			}
		} else {
			// 其他系统按旧逻辑：使用空 Profile 以兼容默认配置
			profiles = []string{""}
		}
		// Do not print profile names derived from disk scan; just count
		logger.Debug("ytdlp export: profiles",
			zap.String("browser", browser),
			zap.Int("profileCount", len(profiles)),
		)

		// 遍历 profile 进行导出。成功的结果合并到同一个 CookieCollection
		// map dreamcreator browser name -> yt-dlp expected name (lowercase)
		toYtDlp := func(name string) string {
			n := strings.ToLower(strings.TrimSpace(name))
			switch n {
			case "chrome":
				return "chrome"
			case "chromium":
				return "chromium"
			case "edge":
				return "edge"
			case "firefox":
				return "firefox"
			case "safari":
				return "safari"
			default:
				return n
			}
		}

		for _, prof := range profiles {
			dlEach := ytdlp.New().Verbose()
			dlEach.SetExecutable(ytdlpExec)
			base := toYtDlp(browser)
			browserSpec := base
			if prof != "" {
				browserSpec = fmt.Sprintf("%s:%s", base, prof)
			}
			cookiePath := filepath.Join(ytdlpDir, fmt.Sprintf("export_%s_%s_cookies.txt", base, strings.ReplaceAll(prof, " ", "_")))
			dlEach.CookiesFromBrowser(browserSpec)
			dlEach.Cookies(cookiePath)
			// Windows: 显式传递关键环境变量，确保 yt-dlp 在相同用户上下文读取浏览器配置
			if runtime.GOOS == "windows" {
				for _, key := range []string{"LOCALAPPDATA", "APPDATA", "USERPROFILE", "HOMEDRIVE", "HOMEPATH"} {
					if val := os.Getenv(key); val != "" {
						dlEach.SetEnvVar(key, val)
					}
				}
			}
			logger.Debug("ytdlp export: running",
				zap.String("browserSpec", browserSpec),
				zap.String("out", cookiePath),
			)
			result, rerr := dlEach.Run(ctx)
			if rerr != nil {
				logger.Warn("ytdlp export: run error",
					zap.String("browserSpec", browserSpec),
					zap.String("out", cookiePath),
					zap.Error(rerr),
				)
				if result != nil && result.Stderr != "" {
					msg := result.Stderr
					if len(msg) > 800 {
						msg = msg[:800] + "..."
					}
					logger.Debug("ytdlp export: stderr", zap.String("stderr", msg))
					// 提取形如 "ERROR: ..." 的首行作为用户可见错误
					line := strings.SplitN(result.Stderr, "\n", 2)[0]
					if idx := strings.Index(line, "ERROR:"); idx >= 0 {
						short := strings.TrimSpace(line[idx:]) // 从 ERROR: 开始
						// 去掉路径部分（例如 ": '/Users/...'")
						cut := strings.Index(short, ": '")
						if cut == -1 {
							cut = strings.Index(short, ": /")
						}
						if cut > 0 {
							short = strings.TrimSpace(strings.TrimSuffix(short[:cut], ":"))
						}
						if short != "" {
							lastErrMsg = short
						}
					}
				}
				if lastErrMsg == "" {
					// 退化：从 error 串提要
					em := rerr.Error()
					if i := strings.Index(em, "ERROR:"); i >= 0 {
						short := strings.TrimSpace(em[i:])
						if j := strings.Index(short, "\n"); j > 0 {
							short = short[:j]
						}
						if k := strings.Index(short, ": '"); k > 0 {
							short = strings.TrimSpace(strings.TrimSuffix(short[:k], ":"))
						}
						lastErrMsg = short
					} else {
						lastErrMsg = em
					}
				}
				// 针对 macOS Safari 权限受限的友好提示
				lowerErr := strings.ToLower(rerr.Error())
				var lowerStderr string
				if result != nil {
					lowerStderr = strings.ToLower(result.Stderr)
				}
				if runtime.GOOS == "darwin" && strings.EqualFold(base, "safari") && (strings.Contains(lowerErr, "operation not permitted") || strings.Contains(lowerStderr, "operation not permitted")) {
					// 不中断提取的错误文案，但保留提示供日志排查
					if lastErrMsg == "" {
						lastErrMsg = "ERROR: Operation not permitted"
					}
				}
			} else if result != nil {
				logger.Debug("ytdlp export: done",
					zap.Int("exit", result.ExitCode),
				)
			}
			if _, osErr := os.Stat(cookiePath); osErr != nil {
				// 该 profile 失败，尝试下一个
				logger.Warn("ytdlp export: output not found",
					zap.String("browserSpec", browserSpec),
					zap.String("out", cookiePath),
					zap.Error(osErr),
				)
				continue
			}
			if cookiesPaths[browser] == nil {
				cookiesPaths[browser] = &agg{}
			}
			cookiesPaths[browser].paths = append(cookiesPaths[browser].paths, cookiePath)
		}

		// 记录错误摘要（若有）以便在后续合并阶段展示
		if strings.TrimSpace(lastErrMsg) != "" {
			errMsgByBrowser[browser] = lastErrMsg
		}

		browserCookiesMap[string(browser)] = eachBrowserCookies
	}

	// 使用c.convertFromNetscape()转换为http.Cookie（合并多 profile 的导出）
	for browser, item := range cookiesPaths {
		var merged = make(map[string]*types.DomainCookies)
		var ok bool
		for _, cookiePath := range item.paths {
			// Skip logging concrete cookie path to avoid printing scanned filenames
			domainCookies, err := c.convertFromNetscapeFile(cookiePath)
			if err != nil {
				logger.Warn("ytdlp export: convert failed", zap.String("path", cookiePath), zap.Error(err))
				continue
			}
			if domainCookies != nil {
				ok = true
				for d, dc := range domainCookies {
					if merged[d] == nil {
						merged[d] = &types.DomainCookies{Domain: d, Cookies: []*http.Cookie{}}
					}
					merged[d].Cookies = append(merged[d].Cookies, dc.Cookies...)
				}
			}
		}
		if ok {
			logger.Debug("ytdlp export: merged domains",
				zap.String("browser", browser),
				zap.Int("domains", len(merged)),
			)
			browserCookiesMap[browser].Status = "synced"
			browserCookiesMap[browser].StatusDescription = "cookies synced"
			browserCookiesMap[browser].LastSyncTime = time.Now()
			browserCookiesMap[browser].LastSyncStatus = "success"
			browserCookiesMap[browser].DomainCookies = merged
		} else {
			logger.Warn("ytdlp export: no cookies found after export",
				zap.String("browser", browser),
				zap.Int("attempts", len(item.paths)),
			)
			browserCookiesMap[browser].Status = "error"
			// 优先显示 yt-dlp 的 ERROR 首行，作为用户可见的失败原因
			if msg := strings.TrimSpace(errMsgByBrowser[browser]); msg != "" {
				browserCookiesMap[browser].StatusDescription = msg
			} else {
				browserCookiesMap[browser].StatusDescription = "no cookies found"
			}
			browserCookiesMap[browser].LastSyncTime = time.Now()
			browserCookiesMap[browser].LastSyncStatus = "failed"
		}
	}

	// 对于未出现在 cookiesPaths 的浏览器，标记为失败，避免状态停留在 "syncing"
	for b, bc := range browserCookiesMap {
		if _, ok := cookiesPaths[b]; !ok {
			if bc != nil && strings.EqualFold(bc.LastSyncStatus, "syncing") {
				bc.Status = "error"
				if msg := strings.TrimSpace(errMsgByBrowser[b]); msg != "" {
					bc.StatusDescription = msg
				} else {
					bc.StatusDescription = "no cookies found"
				}
				bc.LastSyncStatus = "failed"
				bc.LastSyncTime = time.Now()
			}
		}
	}

	// 删除对应的export_[browser]_cookies.txt文件
	for _, item := range cookiesPaths {
		for _, cookiePath := range item.paths {
			if err := os.Remove(cookiePath); err != nil {
				logger.Error("Failed to remove export cookies file", zap.String("path", cookiePath), zap.Error(err))
			}
		}
	}

	return browserCookiesMap
}

func (c *cookieManager) convertToNetscape(cookies []*http.Cookie) string {
	var b strings.Builder
	b.WriteString("# Netscape HTTP Cookie File\n\n")

	for _, cookie := range cookies {
		domain := cookie.Domain
		if strings.HasPrefix(domain, "#HttpOnly_") {
			domain = strings.TrimPrefix(domain, "#HttpOnly_")
		}

		flag := "FALSE"
		if strings.HasPrefix(domain, ".") {
			flag = "TRUE"
		}

		secure := "FALSE"
		if cookie.Secure {
			secure = "TRUE"
		}

		domainField := domain
		if cookie.HttpOnly {
			domainField = "#HttpOnly_" + strings.TrimPrefix(domain, "#HttpOnly_")
		}

		var expiration int64
		if !cookie.Expires.IsZero() {
			expiration = cookie.Expires.Unix()
		}

		line := fmt.Sprintf(
			"%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			domainField,
			flag,
			cookie.Path,
			secure,
			expiration,
			cookie.Name,
			cookie.Value,
		)
		b.WriteString(line)
	}

	return b.String()
}

func (c *cookieManager) convertFromNetscapeFile(path string) (map[string]*types.DomainCookies, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseNetscapeBytes(data)
}

func parseNetscapeBytes(data []byte) (map[string]*types.DomainCookies, error) {
	domainCookies := make(map[string]*types.DomainCookies)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		httpOnly := false
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "#httponly_") {
			httpOnly = true
			line = line[len("#HttpOnly_"):]
		} else if strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) != 7 {
			continue
		}

		domain := fields[0]
		path := fields[2]
		secure := fields[3]
		expirationStr := fields[4]
		name := fields[5]
		value := fields[6]

		cookie := &http.Cookie{
			Name:   name,
			Value:  value,
			Domain: domain,
			Path:   path,
		}

		if strings.EqualFold(secure, "TRUE") {
			cookie.Secure = true
		}
		cookie.HttpOnly = httpOnly

		if expirationStr != "0" && expirationStr != "" {
			if expiration, err := strconv.ParseInt(expirationStr, 10, 64); err == nil {
				minTimestamp := int64(0)
				maxTimestamp := int64(253402300799)
				if expiration >= minTimestamp && expiration <= maxTimestamp {
					cookie.Expires = time.Unix(expiration, 0)
				} else {
					cookie.Expires = time.Time{}
				}
			}
		}

		if domainCookies[domain] == nil {
			domainCookies[domain] = &types.DomainCookies{Domain: domain, Cookies: []*http.Cookie{}}
		}

		domainCookies[domain].Cookies = append(domainCookies[domain].Cookies, cookie)
	}

	return domainCookies, nil
}

func manualPayloadToDomainCookies(payload *types.ManualCollectionPayload) (map[string]*types.DomainCookies, error) {
	result := make(map[string]*types.DomainCookies)
	if payload == nil {
		return result, nil
	}

	if strings.TrimSpace(payload.Netscape) != "" {
		parsed, err := parseNetscapeBytes([]byte(payload.Netscape))
		if err != nil {
			return nil, err
		}
		return parsed, nil
	}

	for _, item := range payload.Cookies {
		domain := strings.TrimSpace(item.Domain)
		name := strings.TrimSpace(item.Name)
		if domain == "" || name == "" {
			continue
		}

		path := item.Path
		if strings.TrimSpace(path) == "" {
			path = "/"
		}

		cookie := &http.Cookie{
			Name:     name,
			Value:    item.Value,
			Domain:   domain,
			Path:     path,
			Secure:   item.Secure,
			HttpOnly: item.HTTPOnly,
		}
		if item.Expires > 0 {
			cookie.Expires = time.Unix(item.Expires, 0)
		}

		if result[domain] == nil {
			result[domain] = &types.DomainCookies{Domain: domain, Cookies: []*http.Cookie{}}
		}
		result[domain].Cookies = append(result[domain].Cookies, cookie)
	}

	return result, nil
}

func mergeDomainCookies(dest map[string]*types.DomainCookies, src map[string]*types.DomainCookies) map[string]*types.DomainCookies {
	if dest == nil && len(src) == 0 {
		return dest
	}
	if dest == nil {
		dest = make(map[string]*types.DomainCookies)
	}

	for domain, dc := range src {
		if dc == nil {
			continue
		}
		if dest[domain] == nil {
			dest[domain] = &types.DomainCookies{Domain: domain, Cookies: []*http.Cookie{}}
		}

		existing := dest[domain]
		keySet := make(map[string]struct{})
		for _, cookie := range existing.Cookies {
			if cookie == nil {
				continue
			}
			key := fmt.Sprintf("%s|%s", cookie.Name, cookie.Path)
			keySet[key] = struct{}{}
		}

		for _, cookie := range dc.Cookies {
			if cookie == nil {
				continue
			}
			key := fmt.Sprintf("%s|%s", cookie.Name, cookie.Path)
			if _, exists := keySet[key]; exists {
				// replace existing cookie with the new value
				for idx, existingCookie := range existing.Cookies {
					if existingCookie != nil && existingCookie.Name == cookie.Name && existingCookie.Path == cookie.Path {
						existing.Cookies[idx] = cookie
						break
					}
				}
				continue
			}
			existing.Cookies = append(existing.Cookies, cookie)
		}
	}

	return dest
}

func (c *cookieManager) generateDomainLookups(hostname string) []string {
	// 1. 完全匹配是第一优先级
	lookups := []string{hostname}

	registrableDomain, err := publicsuffix.EffectiveTLDPlusOne(hostname)
	if err == nil && registrableDomain != "" {
		lookups = append(lookups, "."+registrableDomain)
	} else {
		logger.Debug("Failed to get registrable domain", zap.String("hostname", hostname), zap.Error(err))
	}

	return lookups
}

// kooky removed

package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	cdptarget "github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"

	"dreamcreator/internal/application/browsercdp"
	"dreamcreator/internal/application/connectors/dto"
	appcookies "dreamcreator/internal/application/cookies"
	"dreamcreator/internal/application/sitepolicy"
	"dreamcreator/internal/domain/connectors"
)

func (service *ConnectorsService) StartConnectorConnect(ctx context.Context, request dto.StartConnectorConnectRequest) (dto.StartConnectorConnectResult, error) {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return dto.StartConnectorConnectResult{}, connectors.ErrInvalidConnector
	}
	connector, err := service.repo.Get(ctx, id)
	if err != nil {
		return dto.StartConnectorConnectResult{}, err
	}
	targetURL, err := connectorHomeURL(connector.Type)
	if err != nil {
		return dto.StartConnectorConnectResult{}, err
	}

	sessionID := service.newSessionID()
	userDataDir := connectorSessionDir(connector.Type, sessionID)
	runtime, tabCtx, cancel, err := service.startBrowser(service.preferredBrowser(ctx), false, userDataDir)
	if err != nil {
		return dto.StartConnectorConnectResult{}, err
	}
	if err := chromedp.Run(tabCtx, chromedp.Navigate(targetURL)); err != nil {
		cancel()
		runtime.Stop()
		if service.removeAll != nil {
			_ = service.removeAll(userDataDir)
		}
		return dto.StartConnectorConnectResult{}, err
	}

	session := &connectorSession{
		ID:                sessionID,
		ConnectorID:       connector.ID,
		ConnectorType:     connector.Type,
		Runtime:           runtime,
		TabCtx:            tabCtx,
		Cancel:            cancel,
		UserDataDir:       userDataDir,
		State:             connectorSessionStateRunning,
		ConnectorSnapshot: mapConnectorDTO(connector),
		finalizeDone:      make(chan struct{}),
	}
	if current := chromedp.FromContext(tabCtx); current != nil && current.Target != nil {
		session.TargetID = current.Target.TargetID
	}

	replaced := service.putSession(session)
	service.cleanupSession(replaced)
	service.startConnectSessionMonitor(sessionID)
	log.Printf("connectors: started connect session id=%s connector=%s target=%s userDataDir=%s", sessionID, connector.Type, session.TargetID, userDataDir)

	return dto.StartConnectorConnectResult{
		SessionID: sessionID,
		Connector: mapConnectorDTO(connector),
	}, nil
}

func (service *ConnectorsService) FinishConnectorConnect(ctx context.Context, request dto.FinishConnectorConnectRequest) (dto.FinishConnectorConnectResult, error) {
	sessionID := strings.TrimSpace(request.SessionID)
	if sessionID == "" {
		return dto.FinishConnectorConnectResult{}, connectors.ErrConnectorSessionGone
	}
	result, _, err := service.finalizeConnectSession(ctx, sessionID, "manual_finish")
	if err != nil {
		return dto.FinishConnectorConnectResult{}, err
	}
	return result, nil
}

func (service *ConnectorsService) CancelConnectorConnect(ctx context.Context, request dto.CancelConnectorConnectRequest) error {
	sessionID := strings.TrimSpace(request.SessionID)
	if sessionID == "" {
		return connectors.ErrConnectorSessionGone
	}
	log.Printf("connectors: canceled connect session id=%s", sessionID)
	service.cleanupSession(service.popSession(sessionID))
	return nil
}

func (service *ConnectorsService) finalizeConnectSession(ctx context.Context, sessionID string, reason string) (dto.FinishConnectorConnectResult, bool, error) {
	session, ok := service.getSession(sessionID)
	if !ok || session == nil {
		return dto.FinishConnectorConnectResult{}, false, connectors.ErrConnectorSessionGone
	}
	triggered := false
	session.finalizeOnce.Do(func() {
		triggered = true
		result, err := service.performFinalize(ctx, session, reason)
		service.mu.Lock()
		defer service.mu.Unlock()
		if err != nil {
			session.State = connectorSessionStateFailed
			session.FinalError = err.Error()
		} else {
			session.State = connectorSessionStateCompleted
			session.FinalError = ""
			session.FinalResult = &result
			session.ConnectorSnapshot = result.Connector
		}
		close(session.finalizeDone)
	})
	<-session.finalizeDone

	session, ok = service.getSession(sessionID)
	if !ok || session == nil {
		return dto.FinishConnectorConnectResult{}, triggered, connectors.ErrConnectorSessionGone
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if session.FinalError != "" {
		return dto.FinishConnectorConnectResult{}, triggered, errors.New(session.FinalError)
	}
	if session.FinalResult == nil {
		return dto.FinishConnectorConnectResult{}, triggered, connectors.ErrConnectorSessionDead
	}
	return *session.FinalResult, triggered, nil
}

func (service *ConnectorsService) performFinalize(ctx context.Context, session *connectorSession, reason string) (dto.FinishConnectorConnectResult, error) {
	if session == nil {
		return dto.FinishConnectorConnectResult{}, connectors.ErrConnectorSessionGone
	}

	log.Printf("connectors: finalize requested session=%s connector=%s reason=%s", session.ID, session.ConnectorType, reason)
	records, err := readConnectorCookiesFromRuntime(session.Runtime)
	if err != nil {
		log.Printf("connectors: live cookie read failed session=%s connector=%s reason=%s err=%v", session.ID, session.ConnectorType, reason, err)
		service.mu.Lock()
		records = append([]appcookies.Record(nil), session.LastCookies...)
		service.mu.Unlock()
	} else {
		service.updateSession(session.ID, func(current *connectorSession) {
			current.LastCookies = append([]appcookies.Record(nil), records...)
			current.LastCookiesAt = service.now()
		})
	}

	policy, _ := sitepolicy.ForConnectorType(string(session.ConnectorType))
	filtered := appcookies.FilterByDomains(records, policy.Domains)
	log.Printf("connectors: finalize cookies session=%s connector=%s reason=%s raw=%d filtered=%d domains=%s", session.ID, session.ConnectorType, reason, len(records), len(filtered), strings.Join(cookieDomains(filtered), ","))

	current, err := service.repo.Get(ctx, session.ConnectorID)
	if err != nil {
		service.cleanupSession(session)
		return dto.FinishConnectorConnectResult{}, err
	}

	result := dto.FinishConnectorConnectResult{
		SessionID:            session.ID,
		Saved:                len(filtered) > 0,
		RawCookiesCount:      len(records),
		FilteredCookiesCount: len(filtered),
		Domains:              cookieDomains(filtered),
		Reason:               reason,
		Connector:            mapConnectorDTO(current),
	}
	if len(filtered) == 0 {
		service.cleanupSession(session)
		log.Printf("connectors: finalize completed without matching cookies session=%s connector=%s reason=%s", session.ID, session.ConnectorType, reason)
		return result, nil
	}

	cookiesJSON, err := encodeCookies(filtered)
	if err != nil {
		service.cleanupSession(session)
		return dto.FinishConnectorConnectResult{}, err
	}
	now := service.now()
	updated, err := connectors.NewConnector(connectors.ConnectorParams{
		ID:             current.ID,
		Type:           string(current.Type),
		Status:         string(connectors.StatusConnected),
		CookiesJSON:    cookiesJSON,
		LastVerifiedAt: &now,
		CreatedAt:      &current.CreatedAt,
		UpdatedAt:      &now,
	})
	if err != nil {
		service.cleanupSession(session)
		return dto.FinishConnectorConnectResult{}, err
	}
	if err := service.repo.Save(ctx, updated); err != nil {
		service.cleanupSession(session)
		return dto.FinishConnectorConnectResult{}, err
	}
	result.Connector = mapConnectorDTO(updated)
	service.cleanupSession(session)
	log.Printf("connectors: finalize saved cookies session=%s connector=%s reason=%s filtered=%d", session.ID, session.ConnectorType, reason, len(filtered))
	return result, nil
}

func (service *ConnectorsService) startConnectSessionMonitor(sessionID string) {
	session, ok := service.getSession(sessionID)
	if !ok || session == nil {
		return
	}
	service.watchConnectSessionTarget(sessionID, session)
	service.watchConnectSessionBrowser(sessionID, session)

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			session, ok := service.getSession(sessionID)
			if !ok || session == nil {
				return
			}
			service.mu.Lock()
			state := session.State
			runtime := session.Runtime
			targetID := session.TargetID
			tabCtx := session.TabCtx
			service.mu.Unlock()
			if state != connectorSessionStateRunning {
				return
			}
			if runtime == nil || !runtime.Status().Ready {
				service.triggerSessionFinalize(sessionID, "browser_closed")
				return
			}
			if cookies, err := readConnectorCookiesFromRuntime(runtime); err == nil {
				service.updateSession(sessionID, func(current *connectorSession) {
					current.LastCookies = append([]appcookies.Record(nil), cookies...)
					current.LastCookiesAt = service.now()
				})
			}
			if targetID != "" {
				exists, err := connectorTargetExists(runtime, targetID)
				if err == nil && !exists {
					service.triggerSessionFinalize(sessionID, "tab_closed")
					return
				}
			}
			var tabDone <-chan struct{}
			if tabCtx != nil {
				tabDone = tabCtx.Done()
			}
			var browserDone <-chan struct{}
			if browserCtx := runtime.BrowserContext(); browserCtx != nil {
				browserDone = browserCtx.Done()
			}

			select {
			case <-ticker.C:
			case <-tabDone:
				service.triggerSessionFinalize(sessionID, "tab_closed")
				return
			case <-browserDone:
				service.triggerSessionFinalize(sessionID, "browser_closed")
				return
			}
		}
	}()
}

func (service *ConnectorsService) watchConnectSessionTarget(sessionID string, session *connectorSession) {
	if session == nil || session.TabCtx == nil {
		return
	}
	targetID := session.TargetID
	chromedp.ListenTarget(session.TabCtx, func(ev any) {
		switch current := ev.(type) {
		case *cdptarget.EventTargetDestroyed:
			if targetID != "" && current.TargetID != targetID {
				return
			}
			service.triggerSessionFinalize(sessionID, "tab_closed")
		case *cdptarget.EventTargetCrashed:
			if targetID != "" && current.TargetID != targetID {
				return
			}
			service.triggerSessionFinalize(sessionID, "tab_closed")
		case *cdptarget.EventDetachedFromTarget:
			service.triggerSessionFinalize(sessionID, "tab_closed")
		}
	})
}

func (service *ConnectorsService) watchConnectSessionBrowser(sessionID string, session *connectorSession) {
	if session == nil || session.Runtime == nil || session.Runtime.BrowserContext() == nil {
		return
	}
	go func(browserCtx context.Context) {
		<-browserCtx.Done()
		service.triggerSessionFinalize(sessionID, "browser_closed")
	}(session.Runtime.BrowserContext())
}

func (service *ConnectorsService) triggerSessionFinalize(sessionID string, reason string) {
	go func() {
		_, _, err := service.finalizeConnectSession(context.Background(), sessionID, reason)
		if err != nil && !errors.Is(err, connectors.ErrConnectorSessionGone) {
			log.Printf("connectors: auto-finalize failed session=%s reason=%s err=%v", sessionID, reason, err)
		}
	}()
}

func connectorTargetExists(runtime *browsercdp.Runtime, targetID cdptarget.ID) (bool, error) {
	if runtime == nil || targetID == "" {
		return true, nil
	}
	timeoutCtx, cancel := context.WithTimeout(runtime.BrowserContext(), 3*time.Second)
	defer cancel()

	var exists bool
	if err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(actionCtx context.Context) error {
		targets, err := cdptarget.GetTargets().Do(actionCtx)
		if err != nil {
			return err
		}
		for _, info := range targets {
			if info != nil && info.TargetID == targetID {
				exists = true
				break
			}
		}
		return nil
	})); err != nil {
		return false, err
	}
	return exists, nil
}

func connectorHomeURL(connectorType connectors.ConnectorType) (string, error) {
	switch connectorType {
	case connectors.ConnectorGoogle:
		return "https://www.google.com/", nil
	case connectors.ConnectorGitHub:
		return "https://github.com/", nil
	case connectors.ConnectorReddit:
		return "https://www.reddit.com/", nil
	case connectors.ConnectorZhihu:
		return "https://www.zhihu.com/", nil
	case connectors.ConnectorX:
		return "https://x.com/", nil
	case connectors.ConnectorXiaohongshu:
		return "https://www.xiaohongshu.com/", nil
	case connectors.ConnectorBilibili:
		return "https://www.bilibili.com/", nil
	default:
		return "", connectors.ErrInvalidConnector
	}
}

func startConnectorBrowser(preferredBrowser string, headless bool, userDataDir string) (*browsercdp.Runtime, context.Context, context.CancelFunc, error) {
	runtime, err := browsercdp.Start(context.Background(), browsercdp.LaunchOptions{
		PreferredBrowser: preferredBrowser,
		Headless:         headless,
		UserDataDir:      userDataDir,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	tabCtx, cancel, err := attachConnectorTab(runtime)
	if err != nil {
		runtime.Stop()
		return nil, nil, nil, err
	}
	return runtime, tabCtx, cancel, nil
}

func attachConnectorTab(runtime *browsercdp.Runtime) (context.Context, context.CancelFunc, error) {
	if runtime == nil {
		return nil, nil, connectors.ErrConnectorSessionDead
	}
	targets, err := chromedp.Targets(runtime.BrowserContext())
	if err != nil {
		return nil, nil, err
	}

	targetID := selectConnectorStartupTarget(targets)
	var tabCtx context.Context
	var cancel context.CancelFunc
	if targetID != "" {
		tabCtx, cancel = chromedp.NewContext(runtime.BrowserContext(), chromedp.WithTargetID(targetID))
	} else {
		tabCtx, cancel = chromedp.NewContext(runtime.BrowserContext())
	}
	if err := chromedp.Run(tabCtx); err != nil {
		cancel()
		return nil, nil, err
	}
	return tabCtx, cancel, nil
}

func selectConnectorStartupTarget(targets []*cdptarget.Info) cdptarget.ID {
	var fallback cdptarget.ID
	for _, item := range targets {
		if item == nil || item.Type != "page" || item.Attached {
			continue
		}
		if isConnectorStartupBlank(item.URL) {
			return item.TargetID
		}
		if fallback == "" {
			fallback = item.TargetID
		}
	}
	return fallback
}

func isConnectorStartupBlank(targetURL string) bool {
	trimmed := strings.TrimSpace(targetURL)
	if trimmed == "" || trimmed == "about:blank" {
		return true
	}
	return strings.HasPrefix(trimmed, "chrome://newtab") ||
		strings.HasPrefix(trimmed, "edge://newtab") ||
		strings.HasPrefix(trimmed, "brave://newtab")
}

func readConnectorCookies(ctx context.Context) ([]appcookies.Record, error) {
	var records []appcookies.Record
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(actionCtx context.Context) error {
		items, err := browsercdp.GetAllCookies(actionCtx)
		if err != nil {
			return err
		}
		records = items
		return nil
	})); err != nil {
		return nil, err
	}
	return records, nil
}

func readConnectorCookiesFromRuntime(runtime *browsercdp.Runtime) ([]appcookies.Record, error) {
	if runtime == nil {
		return nil, connectors.ErrConnectorSessionDead
	}
	timeoutCtx, cancel := context.WithTimeout(runtime.BrowserContext(), 8*time.Second)
	defer cancel()

	var records []appcookies.Record
	if err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(actionCtx context.Context) error {
		items, err := browsercdp.GetStorageCookies(actionCtx)
		if err != nil {
			return err
		}
		records = items
		return nil
	})); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("connector cookie read timed out: %w", err)
		}
		return nil, err
	}
	return records, nil
}

func waitForConnectorTabClose(ctx context.Context, runtime *browsercdp.Runtime, tabCtx context.Context, captureCookies bool, readCookies func(context.Context) ([]appcookies.Record, error)) ([]appcookies.Record, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var latest []appcookies.Record
	for {
		select {
		case <-ctx.Done():
			return latest, ctx.Err()
		case <-ticker.C:
			if captureCookies && readCookies != nil {
				if cookies, err := readCookies(tabCtx); err == nil {
					latest = cookies
				}
			}
			if runtime == nil || !runtime.Status().Ready {
				return latest, nil
			}
			var currentURL string
			if err := chromedp.Run(tabCtx, chromedp.Location(&currentURL)); err != nil {
				return latest, nil
			}
		}
	}
}

func connectorSessionDir(connectorType connectors.ConnectorType, sessionID string) string {
	return filepath.Join(connectorSessionRootDir(), string(connectorType), sessionID)
}

func connectorOpenDir(connectorType connectors.ConnectorType, sessionID string) string {
	return filepath.Join(connectorSessionRootDir(), "open", string(connectorType), sessionID)
}

func connectorSessionRootDir() string {
	return filepath.Join(os.TempDir(), "dreamcreator", "connectors")
}

func cookieDomains(records []appcookies.Record) []string {
	if len(records) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(records))
	result := make([]string, 0, len(records))
	for _, record := range records {
		domain := strings.TrimSpace(record.Domain)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		result = append(result, domain)
	}
	sort.Strings(result)
	return result
}

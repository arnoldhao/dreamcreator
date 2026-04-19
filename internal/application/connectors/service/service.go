package service

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/google/uuid"

	"dreamcreator/internal/application/browsercdp"
	"dreamcreator/internal/application/connectors/dto"
	appcookies "dreamcreator/internal/application/cookies"
	settingsdto "dreamcreator/internal/application/settings/dto"
	"dreamcreator/internal/domain/connectors"
)

type SettingsReader interface {
	GetSettings(ctx context.Context) (settingsdto.Settings, error)
}

type ConnectorsService struct {
	repo     connectors.Repository
	settings SettingsReader
	now      func() time.Time

	mu                  sync.Mutex
	sessions            map[string]*connectorSession
	sessionsByConnector map[string]string
	startBrowser        func(preferredBrowser string, headless bool, userDataDir string) (*browsercdp.Runtime, context.Context, context.CancelFunc, error)
	readCookies         func(ctx context.Context) ([]appcookies.Record, error)
	removeAll           func(path string) error
	newSessionID        func() string
}

const (
	connectorSessionStateRunning   = "running"
	connectorSessionStateCompleted = "completed"
	connectorSessionStateFailed    = "failed"
)

type connectorSession struct {
	ID                string
	ConnectorID       string
	ConnectorType     connectors.ConnectorType
	Runtime           *browsercdp.Runtime
	TabCtx            context.Context
	Cancel            context.CancelFunc
	UserDataDir       string
	TargetID          target.ID
	State             string
	LastCookies       []appcookies.Record
	LastCookiesAt     time.Time
	FinalResult       *dto.FinishConnectorConnectResult
	FinalError        string
	ConnectorSnapshot dto.Connector
	finalizeOnce      sync.Once
	finalizeDone      chan struct{}
}

func NewConnectorsService(repo connectors.Repository, settings SettingsReader) *ConnectorsService {
	return &ConnectorsService{
		repo:                repo,
		settings:            settings,
		now:                 time.Now,
		sessions:            make(map[string]*connectorSession),
		sessionsByConnector: make(map[string]string),
		startBrowser:        startConnectorBrowser,
		readCookies:         readConnectorCookies,
		removeAll:           os.RemoveAll,
		newSessionID:        uuid.NewString,
	}
}

func (service *ConnectorsService) preferredBrowser(ctx context.Context) string {
	if service == nil || service.settings == nil {
		return ""
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return ""
	}
	if tools := current.Tools; tools != nil {
		if browserRaw, ok := tools["browser"].(map[string]any); ok && browserRaw != nil {
			if value, ok := browserRaw["preferredBrowser"].(string); ok && strings.TrimSpace(value) != "" {
				return strings.ToLower(strings.TrimSpace(value))
			}
		}
		if fetchRaw, ok := tools["web_fetch"].(map[string]any); ok && fetchRaw != nil {
			if value, ok := fetchRaw["preferredBrowser"].(string); ok && strings.TrimSpace(value) != "" {
				return strings.ToLower(strings.TrimSpace(value))
			}
		}
	}
	return ""
}

func (service *ConnectorsService) EnsureDefaults(ctx context.Context) error {
	defaults := []struct {
		ID   string
		Type connectors.ConnectorType
	}{
		{ID: "connector-google", Type: connectors.ConnectorGoogle},
		{ID: "connector-github", Type: connectors.ConnectorGitHub},
		{ID: "connector-reddit", Type: connectors.ConnectorReddit},
		{ID: "connector-zhihu", Type: connectors.ConnectorZhihu},
		{ID: "connector-x", Type: connectors.ConnectorX},
		{ID: "connector-xiaohongshu", Type: connectors.ConnectorXiaohongshu},
		{ID: "connector-bilibili", Type: connectors.ConnectorBilibili},
	}
	existing, err := service.repo.List(ctx)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		if !isSupportedConnectorType(item.Type) {
			if err := service.repo.Delete(ctx, item.ID); err != nil {
				return err
			}
			continue
		}
		seen[item.ID] = struct{}{}
	}
	for _, item := range defaults {
		if _, ok := seen[item.ID]; ok {
			continue
		}
		now := service.now()
		connector, err := connectors.NewConnector(connectors.ConnectorParams{
			ID:        item.ID,
			Type:      string(item.Type),
			Status:    string(connectors.StatusDisconnected),
			CreatedAt: &now,
			UpdatedAt: &now,
		})
		if err != nil {
			return err
		}
		if err := service.repo.Save(ctx, connector); err != nil {
			return err
		}
	}
	return nil
}

func (service *ConnectorsService) ListConnectors(ctx context.Context) ([]dto.Connector, error) {
	items, err := service.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.Connector, 0, len(items))
	for _, item := range items {
		if !isSupportedConnectorType(item.Type) {
			continue
		}
		result = append(result, mapConnectorDTO(item))
	}
	return result, nil
}

func (service *ConnectorsService) UpsertConnector(ctx context.Context, request dto.UpsertConnectorRequest) (dto.Connector, error) {
	id := strings.TrimSpace(request.ID)
	connectorType := strings.TrimSpace(request.Type)
	status := strings.TrimSpace(request.Status)
	cookiesPath := strings.TrimSpace(request.CookiesPath)
	if id == "" {
		id = uuid.NewString()
	}
	if connectorType != "" && !isSupportedConnectorType(connectors.ConnectorType(connectorType)) {
		return dto.Connector{}, connectors.ErrInvalidConnector
	}
	now := service.now()
	createdAt := (*time.Time)(nil)
	var lastVerifiedAt *time.Time
	cookiesJSON := ""
	if existing, err := service.repo.Get(ctx, id); err == nil {
		if connectorType == "" {
			connectorType = string(existing.Type)
		}
		if status == "" {
			status = string(existing.Status)
		}
		if cookiesPath == "" {
			cookiesPath = existing.CookiesPath
		}
		createdAt = &existing.CreatedAt
		lastVerifiedAt = existing.LastVerifiedAt
		cookiesJSON = existing.CookiesJSON
	} else if err != connectors.ErrConnectorNotFound {
		return dto.Connector{}, err
	}
	if status == string(connectors.StatusConnected) {
		lastVerifiedAt = &now
	}

	connector, err := connectors.NewConnector(connectors.ConnectorParams{
		ID:             id,
		Type:           connectorType,
		Status:         status,
		CookiesPath:    cookiesPath,
		CookiesJSON:    cookiesJSON,
		LastVerifiedAt: lastVerifiedAt,
		CreatedAt:      createdAt,
		UpdatedAt:      &now,
	})
	if err != nil {
		return dto.Connector{}, err
	}

	if err := service.repo.Save(ctx, connector); err != nil {
		return dto.Connector{}, err
	}

	return mapConnectorDTO(connector), nil
}

func (service *ConnectorsService) ClearConnector(ctx context.Context, request dto.ClearConnectorRequest) error {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return connectors.ErrInvalidConnector
	}
	connector, err := service.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	now := service.now()
	updated, err := connectors.NewConnector(connectors.ConnectorParams{
		ID:          connector.ID,
		Type:        string(connector.Type),
		Status:      string(connectors.StatusDisconnected),
		CookiesJSON: "",
		CreatedAt:   &connector.CreatedAt,
		UpdatedAt:   &now,
	})
	if err != nil {
		return err
	}
	return service.repo.Save(ctx, updated)
}

func (service *ConnectorsService) putSession(session *connectorSession) *connectorSession {
	if service == nil || session == nil {
		return nil
	}
	service.mu.Lock()
	defer service.mu.Unlock()

	var replaced *connectorSession
	if currentID, ok := service.sessionsByConnector[session.ConnectorID]; ok && currentID != "" {
		replaced = service.sessions[currentID]
		delete(service.sessions, currentID)
	}
	service.sessions[session.ID] = session
	service.sessionsByConnector[session.ConnectorID] = session.ID
	return replaced
}

func (service *ConnectorsService) getSession(sessionID string) (*connectorSession, bool) {
	if service == nil {
		return nil, false
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	session, ok := service.sessions[sessionID]
	return session, ok
}

func (service *ConnectorsService) updateSession(sessionID string, update func(session *connectorSession)) (*connectorSession, bool) {
	if service == nil {
		return nil, false
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	session, ok := service.sessions[sessionID]
	if !ok || session == nil {
		return nil, false
	}
	update(session)
	return session, true
}

func (service *ConnectorsService) popSession(sessionID string) *connectorSession {
	if service == nil {
		return nil
	}
	service.mu.Lock()
	defer service.mu.Unlock()

	session := service.sessions[sessionID]
	if session == nil {
		return nil
	}
	delete(service.sessions, sessionID)
	if currentID, ok := service.sessionsByConnector[session.ConnectorID]; ok && currentID == sessionID {
		delete(service.sessionsByConnector, session.ConnectorID)
	}
	return session
}

func (service *ConnectorsService) cleanupSession(session *connectorSession) {
	if session == nil {
		return
	}
	if session.Cancel != nil {
		session.Cancel()
	}
	if session.Runtime != nil {
		session.Runtime.Stop()
	}
	if service.removeAll != nil && strings.TrimSpace(session.UserDataDir) != "" {
		_ = service.removeAll(session.UserDataDir)
	}
}

func (service *ConnectorsService) GetConnectorConnectSession(ctx context.Context, request dto.GetConnectorConnectSessionRequest) (dto.ConnectorConnectSession, error) {
	sessionID := strings.TrimSpace(request.SessionID)
	if sessionID == "" {
		return dto.ConnectorConnectSession{}, connectors.ErrConnectorSessionGone
	}
	session, ok := service.getSession(sessionID)
	if !ok {
		return dto.ConnectorConnectSession{}, connectors.ErrConnectorSessionGone
	}
	return service.snapshotSession(ctx, session), nil
}

func (service *ConnectorsService) snapshotSession(ctx context.Context, session *connectorSession) dto.ConnectorConnectSession {
	if session == nil {
		return dto.ConnectorConnectSession{}
	}
	service.mu.Lock()
	snapshotID := session.ID
	snapshotConnectorID := session.ConnectorID
	snapshotState := session.State
	snapshotLastCookiesAt := session.LastCookiesAt
	snapshotFinalError := session.FinalError
	snapshotConnector := session.ConnectorSnapshot
	var snapshotFinalResult *dto.FinishConnectorConnectResult
	if session.FinalResult != nil {
		copyResult := *session.FinalResult
		copyResult.Domains = append([]string(nil), session.FinalResult.Domains...)
		snapshotFinalResult = &copyResult
	}
	service.mu.Unlock()

	connector := snapshotConnector
	if snapshotFinalResult != nil {
		connector = snapshotFinalResult.Connector
	} else if current, err := service.repo.Get(ctx, snapshotConnectorID); err == nil {
		connector = mapConnectorDTO(current)
	}
	lastCookiesAt := ""
	if !snapshotLastCookiesAt.IsZero() {
		lastCookiesAt = snapshotLastCookiesAt.Format(time.RFC3339)
	}
	result := dto.ConnectorConnectSession{
		SessionID:     snapshotID,
		ConnectorID:   snapshotConnectorID,
		State:         snapshotState,
		Error:         snapshotFinalError,
		LastCookiesAt: lastCookiesAt,
		Connector:     connector,
	}
	if snapshotFinalResult != nil {
		result.Saved = snapshotFinalResult.Saved
		result.RawCookiesCount = snapshotFinalResult.RawCookiesCount
		result.FilteredCookiesCount = snapshotFinalResult.FilteredCookiesCount
		result.Domains = append([]string(nil), snapshotFinalResult.Domains...)
		result.Reason = snapshotFinalResult.Reason
	}
	return result
}

func isSupportedConnectorType(connectorType connectors.ConnectorType) bool {
	switch connectorType {
	case connectors.ConnectorGoogle,
		connectors.ConnectorGitHub,
		connectors.ConnectorReddit,
		connectors.ConnectorZhihu,
		connectors.ConnectorX,
		connectors.ConnectorXiaohongshu,
		connectors.ConnectorBilibili:
		return true
	default:
		return false
	}
}

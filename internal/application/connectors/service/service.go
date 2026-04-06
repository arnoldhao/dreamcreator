package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"dreamcreator/internal/application/connectors/dto"
	"dreamcreator/internal/domain/connectors"
)

type ConnectorsService struct {
	repo connectors.Repository
	now  func() time.Time
}

func NewConnectorsService(repo connectors.Repository) *ConnectorsService {
	return &ConnectorsService{
		repo: repo,
		now:  time.Now,
	}
}

func (service *ConnectorsService) EnsureDefaults(ctx context.Context) error {
	defaults := []struct {
		ID   string
		Type connectors.ConnectorType
	}{
		{ID: "connector-google", Type: connectors.ConnectorGoogle},
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

func isSupportedConnectorType(connectorType connectors.ConnectorType) bool {
	switch connectorType {
	case connectors.ConnectorGoogle, connectors.ConnectorXiaohongshu, connectors.ConnectorBilibili:
		return true
	default:
		return false
	}
}

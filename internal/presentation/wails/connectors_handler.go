package wails

import (
	"context"

	"dreamcreator/internal/application/connectors/dto"
	"dreamcreator/internal/application/connectors/service"
)

type ConnectorsHandler struct {
	service   *service.ConnectorsService
	telemetry connectorsTelemetry
}

type connectorsTelemetry interface {
	TrackConnectorConnected(ctx context.Context, connectorType string)
}

func NewConnectorsHandler(service *service.ConnectorsService, telemetry connectorsTelemetry) *ConnectorsHandler {
	return &ConnectorsHandler{service: service, telemetry: telemetry}
}

func (handler *ConnectorsHandler) ServiceName() string {
	return "ConnectorsHandler"
}

func (handler *ConnectorsHandler) ListConnectors(ctx context.Context) ([]dto.Connector, error) {
	return handler.service.ListConnectors(ctx)
}

func (handler *ConnectorsHandler) UpsertConnector(ctx context.Context, request dto.UpsertConnectorRequest) (dto.Connector, error) {
	return handler.service.UpsertConnector(ctx, request)
}

func (handler *ConnectorsHandler) ClearConnector(ctx context.Context, request dto.ClearConnectorRequest) error {
	return handler.service.ClearConnector(ctx, request)
}

func (handler *ConnectorsHandler) ConnectConnector(ctx context.Context, request dto.ConnectConnectorRequest) (dto.Connector, error) {
	result, err := handler.service.ConnectConnector(ctx, request)
	if err != nil {
		return dto.Connector{}, err
	}
	if handler.telemetry != nil && result.Status == "connected" {
		handler.telemetry.TrackConnectorConnected(ctx, result.Type)
	}
	return result, nil
}

func (handler *ConnectorsHandler) OpenConnectorSite(ctx context.Context, request dto.OpenConnectorSiteRequest) error {
	return handler.service.OpenConnectorSite(ctx, request)
}

func (handler *ConnectorsHandler) InstallPlaywright(ctx context.Context) error {
	return handler.service.InstallPlaywright(ctx)
}

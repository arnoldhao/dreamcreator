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

func (handler *ConnectorsHandler) StartConnectorConnect(ctx context.Context, request dto.StartConnectorConnectRequest) (dto.StartConnectorConnectResult, error) {
	return handler.service.StartConnectorConnect(ctx, request)
}

func (handler *ConnectorsHandler) FinishConnectorConnect(ctx context.Context, request dto.FinishConnectorConnectRequest) (dto.FinishConnectorConnectResult, error) {
	result, err := handler.service.FinishConnectorConnect(ctx, request)
	if err != nil {
		return dto.FinishConnectorConnectResult{}, err
	}
	if handler.telemetry != nil && result.Saved && result.Connector.Status == "connected" {
		handler.telemetry.TrackConnectorConnected(ctx, result.Connector.Type)
	}
	return result, nil
}

func (handler *ConnectorsHandler) CancelConnectorConnect(ctx context.Context, request dto.CancelConnectorConnectRequest) error {
	return handler.service.CancelConnectorConnect(ctx, request)
}

func (handler *ConnectorsHandler) GetConnectorConnectSession(ctx context.Context, request dto.GetConnectorConnectSessionRequest) (dto.ConnectorConnectSession, error) {
	return handler.service.GetConnectorConnectSession(ctx, request)
}

func (handler *ConnectorsHandler) OpenConnectorSite(ctx context.Context, request dto.OpenConnectorSiteRequest) error {
	return handler.service.OpenConnectorSite(ctx, request)
}

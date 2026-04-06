package wails

import (
	"context"

	apptelemetry "dreamcreator/internal/application/telemetry"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const telemetrySignalEvent = "telemetry:signal"

type telemetrySignalEmitter struct {
	app *application.App
}

func NewTelemetrySignalEmitter(app *application.App) apptelemetry.Emitter {
	return &telemetrySignalEmitter{app: app}
}

func (emitter *telemetrySignalEmitter) Emit(signal apptelemetry.Signal) {
	if emitter == nil || emitter.app == nil {
		return
	}
	emitter.app.Event.Emit(telemetrySignalEvent, signal)
}

type TelemetryHandler struct {
	service       *apptelemetry.Service
	launchContext apptelemetry.AppLaunchContext
}

func NewTelemetryHandler(service *apptelemetry.Service, launchContext apptelemetry.AppLaunchContext) *TelemetryHandler {
	return &TelemetryHandler{
		service:       service,
		launchContext: launchContext,
	}
}

func (handler *TelemetryHandler) ServiceName() string {
	return "TelemetryHandler"
}

func (handler *TelemetryHandler) Bootstrap(ctx context.Context) (apptelemetry.Bootstrap, error) {
	if handler == nil || handler.service == nil {
		return apptelemetry.Bootstrap{}, nil
	}
	return handler.service.Bootstrap(ctx)
}

func (handler *TelemetryHandler) TrackAppLaunch(ctx context.Context) (int, error) {
	if handler == nil || handler.service == nil {
		return 0, nil
	}
	return handler.service.TrackAppLaunch(ctx, handler.launchContext)
}

func (handler *TelemetryHandler) FlushSessionSummary(ctx context.Context) error {
	if handler == nil || handler.service == nil {
		return nil
	}
	return handler.service.FlushSessionSummary(ctx)
}

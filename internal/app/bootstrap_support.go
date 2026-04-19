package app

import (
	"context"
	"errors"
	"sync"

	settingsdto "dreamcreator/internal/application/settings/dto"
	"dreamcreator/internal/application/settings/service"
	"dreamcreator/internal/presentation/wails"
)

type providersUpdatedWindowNotifier struct {
	manager *wails.WindowManager
}

func (notifier providersUpdatedWindowNotifier) ProvidersUpdated() {
	if notifier.manager == nil {
		return
	}
	notifier.manager.EmitProvidersUpdated()
}

type settingsBroadcastAdapter struct {
	service *service.SettingsService

	mu      sync.RWMutex
	applier func(settingsdto.Settings)
}

func newSettingsBroadcastAdapter(settingsService *service.SettingsService) *settingsBroadcastAdapter {
	return &settingsBroadcastAdapter{service: settingsService}
}

func (adapter *settingsBroadcastAdapter) SetApplier(applier func(settingsdto.Settings)) {
	if adapter == nil {
		return
	}
	adapter.mu.Lock()
	adapter.applier = applier
	adapter.mu.Unlock()
}

func (adapter *settingsBroadcastAdapter) GetSettings(ctx context.Context) (settingsdto.Settings, error) {
	if adapter == nil || adapter.service == nil {
		return settingsdto.Settings{}, errors.New("settings service unavailable")
	}
	return adapter.service.GetSettings(ctx)
}

func (adapter *settingsBroadcastAdapter) UpdateSettings(ctx context.Context, request settingsdto.UpdateSettingsRequest) (settingsdto.Settings, error) {
	if adapter == nil || adapter.service == nil {
		return settingsdto.Settings{}, errors.New("settings service unavailable")
	}
	return adapter.service.UpdateSettings(ctx, request)
}

func (adapter *settingsBroadcastAdapter) ApplySettings(updated settingsdto.Settings) {
	if adapter == nil {
		return
	}
	adapter.mu.RLock()
	applier := adapter.applier
	adapter.mu.RUnlock()
	if applier != nil {
		applier(updated)
	}
}

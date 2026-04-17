package app

import (
	"context"
	"encoding/json"
	"io/fs"
	"runtime"
	"time"

	gatewayruntimedto "dreamcreator/internal/application/gateway/runtime/dto"
	llmrecord "dreamcreator/internal/application/llmrecord"
	threadservice "dreamcreator/internal/application/thread/service"
	"dreamcreator/internal/infrastructure/providersync"

	"go.uber.org/zap"
)

func marshalCronRuntimeUsage(usage gatewayruntimedto.RuntimeUsage) (string, error) {
	if usage == (gatewayruntimedto.RuntimeUsage{}) {
		return "", nil
	}
	encoded, err := json.Marshal(usage)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func startThreadPurgeWorker(ctx context.Context, service *threadservice.ThreadService) {
	const interval = time.Hour
	const batchSize = 100

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := service.PurgeExpired(ctx, batchSize); err != nil {
					zap.L().Warn("thread purge worker failed", zap.Error(err))
				}
			}
		}
	}()
}

func startLLMCallRecordPruneWorker(ctx context.Context, service *llmrecord.Service) {
	if service == nil {
		return
	}

	const interval = time.Hour

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := service.RunScheduledCleanup(ctx); err != nil {
					zap.L().Warn("llm call record prune worker failed", zap.Error(err))
				}
			}
		}
	}()
}

func startModelsDevCatalogSyncWorker(ctx context.Context, service *providersync.ModelsDevCatalogService) {
	if service == nil {
		return
	}

	const interval = time.Hour

	go func() {
		hasEntries, err := service.HasEntries(ctx)
		if err != nil {
			zap.L().Warn("models.dev catalog check failed", zap.Error(err))
		} else if !hasEntries {
			if _, err := service.Refresh(ctx); err != nil {
				zap.L().Warn("models.dev catalog initial sync failed", zap.Error(err))
			}
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := service.Refresh(ctx); err != nil {
					zap.L().Warn("models.dev catalog sync failed", zap.Error(err))
				}
			}
		}
	}()
}

func loadAppIcon(assets fs.FS) []byte {
	data, err := fs.ReadFile(assets, appIconAssetPath(runtime.GOOS))
	if err != nil {
		zap.L().Debug("app icon not found, fallback to default icon", zap.Error(err))
		return nil
	}
	return data
}

func appIconAssetPath(goos string) string {
	if goos == "windows" {
		return "frontend/dist/appicon_windows.png"
	}
	return "frontend/dist/appicon.png"
}

func loadTrayIcon(assets fs.FS) []byte {
	data, err := fs.ReadFile(assets, trayIconAssetPath(runtime.GOOS))
	if err != nil {
		zap.L().Debug("tray icon not found, fallback to default icon", zap.Error(err))
		return nil
	}
	return data
}

func trayIconAssetPath(goos string) string {
	if goos == "windows" {
		return "frontend/dist/tray_windows.ico"
	}
	return "frontend/dist/tray.png"
}

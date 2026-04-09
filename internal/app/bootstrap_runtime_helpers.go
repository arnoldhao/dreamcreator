package app

import (
	"context"
	"encoding/json"
	"io/fs"
	"time"

	gatewayruntimedto "dreamcreator/internal/application/gateway/runtime/dto"
	threadservice "dreamcreator/internal/application/thread/service"

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

func loadAppIcon(assets fs.FS) []byte {
	data, err := fs.ReadFile(assets, "frontend/dist/appicon.png")
	if err != nil {
		zap.L().Debug("app icon not found, fallback to default icon", zap.Error(err))
		return nil
	}
	return data
}

func loadTrayIcon(assets fs.FS) []byte {
	data, err := fs.ReadFile(assets, "frontend/dist/tray.png")
	if err != nil {
		zap.L().Debug("tray icon not found, fallback to default icon", zap.Error(err))
		return nil
	}
	return data
}

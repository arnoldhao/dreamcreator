package api

import (
	"context"
	"dreamcreator/backend/core/downtasks"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/types"
	"encoding/json"

	"go.uber.org/zap"
)

// CookiesAPI handles browser cookie related operations.
type CookiesAPI struct {
	ctx              context.Context
	downtasksService *downtasks.Service
}

// NewCookiesAPI creates a new CookiesAPI instance.
func NewCookiesAPI(downtasksService *downtasks.Service) *CookiesAPI {
	return &CookiesAPI{
		downtasksService: downtasksService,
	}
}

// WailsInit is called at application startup.
func (a *CookiesAPI) WailsInit(ctx context.Context) error {
	a.ctx = ctx
	return nil
}

// RefreshCookies triggers a refresh of the cookie cache.
func (a *CookiesAPI) SyncCookies(syncFrom string, browsers []string) types.JSResp {
	logger.Debug("Starting browser cookies sync...")

	// 立即启动异步同步，不等待结果
	a.downtasksService.SyncCookies(syncFrom, browsers)

	// 立即返回成功响应
	logger.Debug("Browser cookies sync started successfully.")
	return types.JSResp{
		Success: true,
		Msg:     "Cookie sync started, you will be notified when completed",
		Data:    nil,
	}
}

func (a *CookiesAPI) GetBrowserByDomain(targetURL string) types.JSResp {
	logger.Debug("Getting browser for URL", zap.String("url", targetURL))
	providers, err := a.downtasksService.GetBrowserByDomain(targetURL)
	if err != nil {
		logger.Error("Failed to get browser for URL", zap.String("url", targetURL), zap.Error(err))
		return types.JSResp{
			Success: false,
			Msg:     "Failed to get browser: " + err.Error(),
		}
	}

	data, _ := json.Marshal(providers)
	return types.JSResp{
		Success: true,
		Data:    string(data),
	}
}

// GetCookiesByDomain retrieves cookies for a specific URL from a given cookie provider.
func (a *CookiesAPI) GetCookiesByDomain(browser string, targetURL string) types.JSResp {
	logger.Debug("Getting cookies for URL", zap.String("url", targetURL), zap.String("browser", browser))
	cookies, err := a.downtasksService.GetCookiesByDomain(browser, targetURL)
	if err != nil {
		logger.Error("Failed to get cookies for URL", zap.String("url", targetURL), zap.String("browser", browser), zap.Error(err))
		return types.JSResp{
			Success: false,
			Msg:     "Failed to get cookies: " + err.Error(),
		}
	}

	data, _ := json.Marshal(cookies)
	return types.JSResp{
		Success: true,
		Data:    string(data),
	}
}

// ListAllCookies retrieves all cached cookies, grouped by browser.
func (a *CookiesAPI) ListAllCookies() types.JSResp {
	logger.Debug("Listing all cached cookies by browser.")
	cookies, err := a.downtasksService.ListAllCookies()
	if err != nil {
		logger.Error("Failed to list all cookies", zap.Error(err))
		return types.JSResp{
			Success: false,
			Msg:     "Failed to list all cookies: " + err.Error(),
		}
	}

	data, _ := json.Marshal(cookies)
	return types.JSResp{
		Success: true,
		Data:    string(data),
	}
}

func (a *CookiesAPI) CreateManualCollection(payload types.ManualCollectionPayload) types.JSResp {
	logger.Debug("Creating manual cookie collection")
	col, err := a.downtasksService.CreateManualCookieCollection(&payload)
	if err != nil {
		logger.Error("Failed to create manual collection", zap.Error(err))
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	data, _ := json.Marshal(col)
	return types.JSResp{Success: true, Data: string(data)}
}

func (a *CookiesAPI) UpdateManualCollection(id string, payload types.ManualCollectionPayload) types.JSResp {
	logger.Debug("Updating manual cookie collection", zap.String("id", id))
	col, err := a.downtasksService.UpdateManualCookieCollection(id, &payload)
	if err != nil {
		logger.Error("Failed to update manual collection", zap.String("id", id), zap.Error(err))
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	data, _ := json.Marshal(col)
	return types.JSResp{Success: true, Data: string(data)}
}

func (a *CookiesAPI) DeleteCookieCollection(id string) types.JSResp {
	logger.Debug("Deleting cookie collection", zap.String("id", id))
	if err := a.downtasksService.DeleteCookieCollection(id); err != nil {
		logger.Error("Failed to delete cookie collection", zap.String("id", id), zap.Error(err))
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	return types.JSResp{Success: true}
}

func (a *CookiesAPI) ExportCookieCollection(id string) types.JSResp {
	logger.Debug("Exporting cookie collection", zap.String("id", id))
	data, err := a.downtasksService.ExportCookieCollection(id)
	if err != nil {
		logger.Error("Failed to export cookie collection", zap.String("id", id), zap.Error(err))
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	return types.JSResp{Success: true, Data: data}
}

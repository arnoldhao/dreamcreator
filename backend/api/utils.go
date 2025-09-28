package api

import (
	"context"
	"dreamcreator/backend/core/imageproxies"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/types"

	"encoding/json"
	"go.uber.org/zap"
	"net/url"
)

type UtilsAPI struct {
	ctx context.Context
	ips *imageproxies.Service
}

func NewUtilsAPI(ips *imageproxies.Service) *UtilsAPI {
	return &UtilsAPI{
		ips: ips,
	}
}

func (api *UtilsAPI) Subscribe(ctx context.Context) {
	api.ctx = ctx
}

func (api *UtilsAPI) GetImage(imageUrl string) (resp *types.JSResp) {
	// url check
	if url, err := url.ParseRequestURI(imageUrl); err != nil || url.Scheme == "" || url.Host == "" {
		logger.Warn("UtilsAPI.GetImage: invalid url", zap.String("imageUrl", imageUrl), zap.Error(err))
		return &types.JSResp{Msg: "Invalid image url"}
	}

	// get image (remove verbose proxying logs)
	image, err := api.ips.ProxyImage(imageUrl)
	if err != nil {
		logger.Error("UtilsAPI.GetImage: proxy failed", zap.String("imageUrl", imageUrl), zap.Error(err))
		return &types.JSResp{Msg: err.Error()}
	}

	contentString, err := json.Marshal(image)
	if err != nil {
		logger.Error("UtilsAPI.GetImage: marshal failed", zap.String("imageUrl", imageUrl), zap.Error(err))
		return &types.JSResp{Msg: err.Error()}
	}

	content := string(contentString)
	return &types.JSResp{Success: true, Data: content}
}

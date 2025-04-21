package api

import (
	"CanMe/backend/core/imageproxies"
	"CanMe/backend/types"
	"context"

	"encoding/json"
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
		return &types.JSResp{Msg: "Invalid image url"}
	}

	// get image
	image, err := api.ips.ProxyImage(imageUrl)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	contentString, err := json.Marshal(image)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	content := string(contentString)

	return &types.JSResp{Success: true, Data: content}
}

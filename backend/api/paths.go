package api

import (
	"context"
	"dreamcreator/backend/core/downtasks"
	"dreamcreator/backend/services/preferences"
	"dreamcreator/backend/types"
)

type PathsAPI struct {
	ctx  context.Context
	pref *preferences.Service
	dts  *downtasks.Service
}

func NewPathsAPI(pref *preferences.Service, dts *downtasks.Service) *PathsAPI {
	return &PathsAPI{
		pref: pref,
		dts:  dts,
	}
}

func (api *PathsAPI) Subscribe(ctx context.Context) {
	api.ctx = ctx
}

func (api *PathsAPI) GetPreferencesPath() (resp *types.JSResp) {
	return &types.JSResp{Success: true, Data: api.pref.GetPrefrenceConfigPath()}
}

func (api *PathsAPI) GetTaskDbPath() (resp *types.JSResp) {
	return &types.JSResp{Success: true, Data: api.dts.Path()}
}

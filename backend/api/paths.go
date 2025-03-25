package api

import (
	"CanMe/backend/core/downtasks"
	"CanMe/backend/services/preferences"
	"CanMe/backend/types"
	"context"
	"encoding/json"
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

func (api *PathsAPI) GetFFMPEGPath() (resp *types.JSResp) {
	content, err := api.dts.GetFFMPEGPath()
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	contentString, err := json.Marshal(content)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(contentString)}
}

func (api *PathsAPI) GetYTDLPPath() (resp *types.JSResp) {
	content, err := api.dts.GetYTDLPPath()
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	contentString, err := json.Marshal(content)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(contentString)}
}

func (api *PathsAPI) DependenciesReady() (resp *types.JSResp) {
	ffmpeg, err := api.dts.GetFFMPEGPath()
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	if !ffmpeg.Available {
		return &types.JSResp{Msg: "FFMpeg is not avaliable"}
	}

	ytdlp, err := api.dts.GetYTDLPPath()
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	if !ytdlp.Available {
		return &types.JSResp{Msg: "YTDLP is not avaliable"}
	}

	return &types.JSResp{Success: true}
}

func (api *PathsAPI) SetFFMpegExecPath(execPath string) (resp *types.JSResp) {
	if execPath == "" {
		return &types.JSResp{Msg: "execPath is empty"}
	}

	content, err := api.dts.SetFFMpegPath(execPath)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	contentString, err := json.Marshal(content)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(contentString)}
}

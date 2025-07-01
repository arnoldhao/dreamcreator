package api

import (
	"CanMe/backend/consts"
	"CanMe/backend/core/downtasks"
	"CanMe/backend/pkg/events"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/pkg/websockets"
	"CanMe/backend/types"
	"context"
	"encoding/json"
)

type DowntasksAPI struct {
	ctx      context.Context
	service  *downtasks.Service
	eventBus events.EventBus
	ws       *websockets.Service
}

func NewDowntasksAPI(service *downtasks.Service, eventBus events.EventBus, ws *websockets.Service) *DowntasksAPI {
	return &DowntasksAPI{
		service:  service,
		eventBus: eventBus,
		ws:       ws,
	}
}

func (api *DowntasksAPI) Subscribe(ctx context.Context) {
	api.ctx = ctx

	progressHandler := events.HandlerFunc(func(ctx context.Context, event events.Event) error {
		// WebSocket Logic: report current progress to client
		if data, ok := event.GetData().(*types.DtProgress); ok {
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_DOWNTASKS,
				Event:     consts.EVENT_DOWNTASKS_PROGRESS,
				Data:      data,
			})
		} else {
			logger.Warn("Failed to convert event data to DtProgress")
		}
		return nil
	})

	signalHandler := events.HandlerFunc(func(ctx context.Context, event events.Event) error {
		// WebSocket Logic: report current progress to client
		if data, ok := event.GetData().(*types.DTSignal); ok {
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_DOWNTASKS,
				Event:     consts.EVENT_DOWNTASKS_SIGNAL,
				Data:      data,
			})
		} else {
			logger.Warn("Failed to convert event data to DTSignal")
		}
		return nil
	})

	installHandler := events.HandlerFunc(func(ctx context.Context, event events.Event) error {
		// WebSocket Logic: report current progress to client
		if data, ok := event.GetData().(*types.DtProgress); ok {
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_DOWNTASKS,
				Event:     consts.EVENT_DOWNTASKS_INSTALLING,
				Data:      data,
			})
		} else {
			logger.Warn("Failed to convert event data to DtProgress")
		}
		return nil
	})

	api.eventBus.Subscribe(consts.TopicDowntasksProgress, progressHandler)
	api.eventBus.Subscribe(consts.TopicDowntasksSignal, signalHandler)
	api.eventBus.Subscribe(consts.TopicDowntasksInstalling, installHandler)

}

func (api *DowntasksAPI) GetContent(url string, browser string) (resp *types.JSResp) {
	content, err := api.service.ParseURL(url, browser)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	contentString, err := json.Marshal(content)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(contentString)}
}

func (api *DowntasksAPI) Download(request *types.DtDownloadRequest) (resp *types.JSResp) {
	// params check
	if request.URL == "" {
		return &types.JSResp{Msg: "URL is required"}
	}

	if request.FormatID == "" {
		return &types.JSResp{Msg: "Format ID is required"}
	}

	// download
	content, err := api.service.Download(request)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	contentString, err := json.Marshal(content)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(contentString)}
}

func (api *DowntasksAPI) QuickDownload(request *types.DtQuickDownloadRequest) (resp *types.JSResp) {
	// params check
	if request.URL == "" {
		return &types.JSResp{Msg: "URL is required"}
	}

	if request.Video == "" {
		return &types.JSResp{Msg: "Video is required"}
	}

	// define type
	request.Type = consts.TASK_TYPE_QUICK

	// download
	content, err := api.service.QuickDownload(request)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	contentString, err := json.Marshal(content)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(contentString)}
}

func (api *DowntasksAPI) ListTasks() (resp *types.JSResp) {
	tasks := api.service.ListTasks()

	tasksString, err := json.Marshal(tasks)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(tasksString)}
}

func (api *DowntasksAPI) DeleteTask(id string) (resp *types.JSResp) {
	// params check
	if id == "" {
		return &types.JSResp{Msg: "ID is required"}
	}

	err := api.service.DeleteTask(id)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true}
}

func (api *DowntasksAPI) GetFormats() (resp *types.JSResp) {
	// check
	formats := api.service.GetFormats()
	if formats == nil {
		return &types.JSResp{Msg: "Formats is empty"}
	}

	formatsString, err := json.Marshal(formats)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(formatsString)}
}

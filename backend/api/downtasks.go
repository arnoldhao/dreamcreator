package api

import (
	"CanMe/backend/consts"
	"CanMe/backend/core/downtasks"
	"CanMe/backend/core/events"
	"CanMe/backend/pkg/websockets"
	"CanMe/backend/types"
	"context"
	"encoding/json"

	"github.com/wailsapp/wails/v2/pkg/runtime"
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

	// Subscribe
	api.eventBus.Subscribe(consts.TopicDowntasksProgress, func(event events.Event) {
		// WebSocket Logic:report current progress to client
		if data, ok := event.Data.(*types.DtProgress); ok {
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_DOWNTASKS,
				Event:     consts.EVENT_DOWNTASKS_PROGRESS,
				Data:      data,
			})
		}
	})

	api.eventBus.Subscribe(consts.TopicDowntasksSingle, func(event events.Event) {
		if data, ok := event.Data.(*types.DtProgress); ok {
			runtime.EventsEmit(api.ctx, string(consts.EVENT_DOWNTASKS_SINGLE), map[string]interface{}{
				"taskId": data.ID,
				"status": data.Stage,
			})
		}
	})

	api.eventBus.Subscribe(consts.TopicDowntasksInstalling, func(event events.Event) {
		// WebSocket Logic:report current installing status to client
		if data, ok := event.Data.(*types.DtProgress); ok {
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_DOWNTASKS,
				Event:     consts.EVENT_DOWNTASKS_INSTALLING,
				Data:      data,
			})
		}
	})
}

func (api *DowntasksAPI) GetContent(url string) (resp *types.JSResp) {
	content, err := api.service.ParseURL(url)
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

func (api *DowntasksAPI) InstallYTDLP() (resp *types.JSResp) {
	// install
	path, err := api.service.InstallYTDLP()
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: path}
}

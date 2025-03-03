package api

import (
	"CanMe/backend/core/download"
	"CanMe/backend/core/events"
	"CanMe/backend/core/websockets"
	"CanMe/backend/types"
	"context"
	"encoding/json"

	"CanMe/backend/consts"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DownloadAPI
type DownloadAPI struct {
	ctx      context.Context
	service  *download.Service
	eventBus events.EventBus
	ws       *websockets.Service
}

func NewDownloadAPI(service *download.Service, eventBus events.EventBus, ws *websockets.Service) *DownloadAPI {
	return &DownloadAPI{
		service:  service,
		eventBus: eventBus,
		ws:       ws,
	}
}

// Subscribe
func (api *DownloadAPI) Subscribe(ctx context.Context) {
	api.ctx = ctx

	// Subscribe
	api.eventBus.Subscribe(consts.TopicDownloadProgress, func(event events.Event) {
		// WebSocket Logic:report current progress to client
		if data, ok := event.Data.(types.DownloadResponse); ok {
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_DOWNLOAD,
				Event:     consts.EVENT_DOWNLOAD_PROGRESS,
				Data:      data,
			})
		}
	})

	api.eventBus.Subscribe(consts.TopicDownloadSingle, func(event events.Event) {
		if data, ok := event.Data.(types.DownloadResponse); ok {
			runtime.EventsEmit(api.ctx, consts.TopicDownloadSingle, map[string]interface{}{
				"taskId": data.ID,
				"status": data.TaskStatus.String(),
			})
		}
	})
}

func (api *DownloadAPI) GetContent(url string) (resp *types.JSResp) {
	content, err := api.service.GetContent(url)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	videoString, err := json.Marshal(content)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(videoString)}
}

// StartDownload
func (api *DownloadAPI) StartDownload(req *types.TaskRequest) (resp *types.JSResp) {
	// check params
	err := req.Validate()
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	// in queue check
	if api.service.IsDownloading(req) {
		return &types.JSResp{Msg: "Current Stream is already in queue"}
	}

	// create task
	taskResp, err := api.service.CreateTask(api.ctx, req)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	// return
	return &types.JSResp{Success: true, Data: taskResp}
}

func (api *DownloadAPI) CancelDownload(taskID string) (resp *types.JSResp) {
	err := api.service.CancelTask(taskID)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}
	return &types.JSResp{Success: true}
}

func (api *DownloadAPI) GetAllTasks() (resp *types.JSResp) {
	tasks, err := api.service.GetAllTasks(api.ctx)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	tasksString, err := json.Marshal(tasks)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(tasksString)}
}

func (api *DownloadAPI) ListDownloaded() (resp *types.JSResp) {
	tasks, err := api.service.ListDownloaded(api.ctx)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	tasksString, err := json.Marshal(tasks)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(tasksString)}
}

func (api *DownloadAPI) CheckFFMPEG() (resp *types.JSResp) {
	version, err := api.service.GetFFMPEGVersion(api.ctx)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(version)}
}

func (api *DownloadAPI) DeleteRecord(taskID string) (resp *types.JSResp) {
	// check if is queue
	if api.service.IsDownloadingByTaskID(taskID) {
		return &types.JSResp{Msg: "Current Stream is already in queue"}
	}

	err := api.service.DeleteRecord(api.ctx, taskID)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true}
}

package api

import (
	"context"
	"dreamcreator/backend/consts"
	"dreamcreator/backend/core/downtasks"
	"dreamcreator/backend/core/subtitles"
	"dreamcreator/backend/pkg/events"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/pkg/websockets"
	"dreamcreator/backend/types"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"path/filepath"
	"strings"
)

type DowntasksAPI struct {
	ctx      context.Context
	service  *downtasks.Service
	subs     *subtitles.Service
	eventBus events.EventBus
	ws       *websockets.Service
}

// normalizeLangCode maps human-readable language names to standard codes and
// validates already-coded inputs to a conservative BCP-47-like form.
func normalizeLangCode(s string) string {
	v := strings.TrimSpace(s)
	if v == "" {
		return ""
	}
	// Direct map of common names -> codes
	m := map[string]string{
		"english":               "en",
		"japanese":              "ja",
		"korean":                "ko",
		"russian":               "ru",
		"arabic":                "ar",
		"thai":                  "th",
		"hindi":                 "hi",
		"hebrew":                "he",
		"greek":                 "el",
		"french":                "fr",
		"german":                "de",
		"spanish":               "es",
		"italian":               "it",
		"portuguese":            "pt",
		"dutch":                 "nl",
		"polish":                "pl",
		"czech":                 "cs",
		"vietnamese":            "vi",
		"chinese":               "zh",
		"chinese (simplified)":  "zh-Hans",
		"chinese (traditional)": "zh-Hant",
	}
	if code, ok := m[strings.ToLower(v)]; ok {
		return code
	}
	// Accept already-normalized tokens like en, en-US, zh-Hans
	valid := true
	for i := 0; i < len(v); i++ {
		c := v[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			continue
		}
		valid = false
		break
	}
	if !valid {
		return ""
	}
	// unify separator to dash
	v = strings.ReplaceAll(v, "_", "-")
	return v
}

func NewDowntasksAPI(service *downtasks.Service, subs *subtitles.Service, eventBus events.EventBus, ws *websockets.Service) *DowntasksAPI {
	return &DowntasksAPI{
		service:  service,
		subs:     subs,
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
		// Installing topic can carry two payload types from different sources:
		// 1) dependencies.* -> *types.DtProgress (install/update progress)
		// 2) downtasks.*    -> *types.DTSignal ("refresh" signal with updated task metadata/files)
		if data, ok := event.GetData().(*types.DtProgress); ok {
			// forward as installing event for Settings/Dependency page
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_DOWNTASKS,
				Event:     consts.EVENT_DOWNTASKS_INSTALLING,
				Data:      data,
			})
			return nil
		}
		if sig, ok := event.GetData().(*types.DTSignal); ok {
			// forward as signal event so content views can refresh
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_DOWNTASKS,
				Event:     consts.EVENT_DOWNTASKS_SIGNAL,
				Data:      sig,
			})
			return nil
		}
		logger.Warn("Unknown install event payload type")
		return nil
	})

	cookieSyncHandler := events.HandlerFunc(func(ctx context.Context, event events.Event) error {
		// WebSocket Logic: report cookie sync result to client
		if data, ok := event.GetData().(*types.DTCookieSync); ok {
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_DOWNTASKS,
				Event:     consts.EVENT_DOWNTASKS_COOKIE_SYNC,
				Data:      data,
			})
		} else {
			logger.Warn("Failed to convert event data to DTCookieSync")
		}
		return nil
	})

	stageHandler := events.HandlerFunc(func(ctx context.Context, event events.Event) error {
		if data, ok := event.GetData().(*types.DTStageEvent); ok {
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_DOWNTASKS,
				Event:     consts.EVENT_DOWNTASKS_STAGE,
				Data:      data,
			})
		} else {
			logger.Warn("Failed to convert event data to DTStageEvent")
		}
		return nil
	})

	api.eventBus.Subscribe(consts.TopicDowntasksProgress, progressHandler)
	api.eventBus.Subscribe(consts.TopicDowntasksSignal, signalHandler)
	api.eventBus.Subscribe(consts.TopicDowntasksInstalling, installHandler)
	api.eventBus.Subscribe(consts.TopicDowntasksCookieSync, cookieSyncHandler)
	api.eventBus.Subscribe(consts.TopicDowntasksStage, stageHandler)
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

// ImportTaskSubtitle imports the first available subtitle file of a task into the Subtitle module
// and persists the created project ID back to the task's SubtitleProcess.
func (api *DowntasksAPI) ImportTaskSubtitle(id string) (resp *types.JSResp) {
	if id == "" {
		return &types.JSResp{Msg: "ID is required"}
	}
	if api.subs == nil {
		return &types.JSResp{Msg: "Subtitle service not available"}
	}

	// load task
	task, err := api.service.GetTaskStatus(id)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	// If already imported, verify project exists and return directly
	if pid := task.SubtitleProcess.ProjectID; pid != "" {
		if prj, err := api.subs.GetSubtitle(pid); err == nil && prj != nil {
			payload := map[string]string{"projectId": pid}
			b, _ := json.Marshal(payload)
			return &types.JSResp{Success: true, Data: string(b)}
		}
		// if not found, fallthrough to re-import
	}

	// pick a subtitle file
	if len(task.SubtitleFiles) == 0 {
		return &types.JSResp{Msg: "No subtitle files found for task"}
	}
	filename := task.SubtitleFiles[0]
	// try to resolve absolute path
	path := filename
	if !filepath.IsAbs(filename) {
		if task.OutputDir == "" {
			return &types.JSResp{Msg: "Output directory is empty"}
		}
		path = filepath.Join(task.OutputDir, filename)
	}

	// default processing options (aligned with import modal defaults)
	opts := types.TextProcessingOptions{
		RemoveEmptyLines:    true,
		TrimWhitespace:      true,
		NormalizeLineBreaks: true,
		FixEncoding:         true,
		FixCommonErrors:     true,
		ValidateGuidelines:  true,
		IsKidsContent:       false,
		GuidelineStandard:   types.GuideLineStandardNetflix,
	}

	// try reuse by source path first (handles legacy imports where projectId wasn't persisted)
	var project *types.SubtitleProject
	if prj, err := api.subs.FindSubtitleBySourcePath(path); err == nil && prj != nil {
		project = prj
	} else {
		// import
		project, err = api.subs.ImportSubtitle(path, opts)
		if err != nil {
			return &types.JSResp{Msg: fmt.Sprintf("Import subtitle failed: %v", err)}
		}
	}

	// persist bidirectional links and metadata
	if project != nil {
		// 1) task -> subtitle
		if task.SubtitleProcess.ProjectID == "" {
			task.SubtitleProcess.ProjectID = project.ID
		}
		// map per-language when possible (deduce from filename)
		base := filepath.Base(filename)
		ext := filepath.Ext(base)
		noext := strings.TrimSuffix(base, ext)
		lang := strings.TrimPrefix(filepath.Ext(noext), ".")
		if lang != "" {
			if task.SubtitleProcess.Projects == nil {
				task.SubtitleProcess.Projects = map[string]string{}
			}
			task.SubtitleProcess.Projects[lang] = project.ID
		}
		// Do NOT merge languages from subtitle project back to task.
		// Keep task.SubtitleProcess.Languages strictly based on downloaded subtitle files.
		_ = api.service.UpdateTask(task)

		// 2) subtitle -> task (origin) and ensure file dir persisted
		if project.Metadata.SourceInfo != nil {
			if project.Metadata.SourceInfo.FilePath == "" {
				project.Metadata.SourceInfo.FilePath = path
			}
			if project.Metadata.SourceInfo.FileDir == "" {
				project.Metadata.SourceInfo.FileDir = filepath.Dir(project.Metadata.SourceInfo.FilePath)
			}
		}
		project.Metadata.OriginTaskID = task.ID
		if _, err := api.subs.UpdateSubtitleProject(project); err != nil {
			logger.Warn("Failed to update subtitle origin metadata", zap.Error(err))
		}
	}

	payload := map[string]string{
		"projectId": project.ID,
	}
	b, _ := json.Marshal(payload)
	return &types.JSResp{Success: true, Data: string(b)}
}

// ImportTaskSubtitleByFile imports the specified subtitle file of a task into the Subtitle module
// and persists the created project ID back to the task's SubtitleProcess under its language key.
func (api *DowntasksAPI) ImportTaskSubtitleByFile(id string, filename string) (resp *types.JSResp) {
	if id == "" || filename == "" {
		return &types.JSResp{Msg: "ID and filename are required"}
	}
	if api.subs == nil {
		return &types.JSResp{Msg: "Subtitle service not available"}
	}
	task, err := api.service.GetTaskStatus(id)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	// find file in task
	var match string
	for _, f := range task.SubtitleFiles {
		if filepath.Base(f) == filepath.Base(filename) {
			match = f
			break
		}
	}
	if match == "" {
		return &types.JSResp{Msg: "Subtitle file not found in task"}
	}

	// resolve absolute path
	path := match
	if !filepath.IsAbs(match) {
		if task.OutputDir == "" {
			return &types.JSResp{Msg: "Output directory is empty"}
		}
		path = filepath.Join(task.OutputDir, match)
	}

	opts := types.TextProcessingOptions{
		RemoveEmptyLines:    true,
		TrimWhitespace:      true,
		NormalizeLineBreaks: true,
		FixEncoding:         true,
		FixCommonErrors:     true,
		ValidateGuidelines:  true,
		IsKidsContent:       false,
		GuidelineStandard:   types.GuideLineStandardNetflix,
	}

	// reuse existing by source path
	var project *types.SubtitleProject
	if prj, err := api.subs.FindSubtitleBySourcePath(path); err == nil && prj != nil {
		project = prj
	} else {
		project, err = api.subs.ImportSubtitle(path, opts)
		if err != nil {
			return &types.JSResp{Msg: fmt.Sprintf("Import subtitle failed: %v", err)}
		}
	}

	// persist mapping
	if project != nil {
		if task.SubtitleProcess.Projects == nil {
			task.SubtitleProcess.Projects = map[string]string{}
		}
		base := filepath.Base(match)
		ext := filepath.Ext(base)
		noext := strings.TrimSuffix(base, ext)
		lang := strings.TrimPrefix(filepath.Ext(noext), ".")
		key := lang
		if key == "" {
			key = base
		}
		task.SubtitleProcess.Projects[key] = project.ID
		if task.SubtitleProcess.ProjectID == "" {
			task.SubtitleProcess.ProjectID = project.ID
		}
		// Do NOT merge languages from subtitle project back to task.
		_ = api.service.UpdateTask(task)

		if project.Metadata.SourceInfo != nil {
			if project.Metadata.SourceInfo.FilePath == "" {
				project.Metadata.SourceInfo.FilePath = path
			}
			if project.Metadata.SourceInfo.FileDir == "" {
				project.Metadata.SourceInfo.FileDir = filepath.Dir(project.Metadata.SourceInfo.FilePath)
			}
		}
		project.Metadata.OriginTaskID = task.ID
		if _, err := api.subs.UpdateSubtitleProject(project); err != nil {
			logger.Warn("Failed to update subtitle origin metadata", zap.Error(err))
		}
	}

	payload := map[string]string{"projectId": project.ID}
	b, _ := json.Marshal(payload)
	return &types.JSResp{Success: true, Data: string(b)}
}

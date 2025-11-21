package api

import (
    "context"
    "dreamcreator/backend/consts"
    "dreamcreator/backend/core/subtitles"
    "dreamcreator/backend/pkg/events"
    "dreamcreator/backend/pkg/logger"
    "dreamcreator/backend/pkg/websockets"
    "dreamcreator/backend/types"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/wailsapp/wails/v2/pkg/runtime"

    "go.uber.org/zap"
)

type SubtitlesAPI struct {
	ctx      context.Context
	subs     *subtitles.Service
	eventBus events.EventBus
	ws       *websockets.Service
}

func NewSubtitlesAPI(subs *subtitles.Service, eventBus events.EventBus, ws *websockets.Service) *SubtitlesAPI {
	return &SubtitlesAPI{
		subs:     subs,
		eventBus: eventBus,
		ws:       ws,
	}
}

func (api *SubtitlesAPI) Subscribe(ctx context.Context) {
	api.ctx = ctx

	progressHandler := events.HandlerFunc(func(ctx context.Context, event events.Event) error {
		// WebSocket Logic: report current progress to client
		if data, ok := event.GetData().(types.ConversionTask); ok {
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_SUBTITLES,
				Event:     consts.EVENT_SUBTITLE_PROGRESS,
				Data:      data,
			})
		} else {
			logger.Warn("Failed to convert event data to SubtitleProgress")
		}
		return nil
	})

	api.eventBus.Subscribe(consts.TopicSubtitleProgress, progressHandler)

	// LLM 对话事件：单条消息/状态更新，通过 WS 推送给前端
	talkHandler := events.HandlerFunc(func(ctx context.Context, event events.Event) error {
		if data, ok := event.GetData().(*types.LLMChatEvent); ok {
			api.ws.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_SUBTITLES,
				Event:     consts.EVENT_SUBTITLE_CHAT,
				Data:      data,
			})
		} else {
			logger.Warn("Failed to convert event data to LLMChatEvent")
		}
		return nil
	})
	api.eventBus.Subscribe(consts.TopicSubtitleConversation, talkHandler)
}

func (api *SubtitlesAPI) OpenFileWithOptions(filePath string, options types.TextProcessingOptions) (resp *types.JSResp) {
	// filePath check
	if filePath == "" {
		return &types.JSResp{Msg: "filePath is empty"}
	}

	// get file info
	info, err := api.subs.ImportSubtitle(filePath, options)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}
	// 直接返回结构体，由前端按对象消费
	return &types.JSResp{Success: true, Data: info}
}

func (api *SubtitlesAPI) UpdateExportConfig(id string, config types.ExportConfigs) (resp *types.JSResp) {
	if id == "" {
		return &types.JSResp{Msg: "id is empty"}
	}

	// update export config
	project, err := api.subs.UpdateExportConfig(id, config)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: project}
}

// ExportSubtitleToFile 导出字幕到文件
func (api *SubtitlesAPI) ExportSubtitleToFile(id, langCode, targetFormat string) (resp *types.JSResp) {
	resp = &types.JSResp{Success: false}

	// 1. 获取项目信息用于生成默认文件名
	project, err := api.subs.GetSubtitle(id)
	if err != nil {
		resp.Msg = err.Error()
		return
	}

	// 2. 生成默认文件名
	projectName := "Subtitle_Project"
	if project.ProjectName != "" {
		projectName = project.ProjectName
	} else if project.Metadata.Name != "" {
		projectName = project.Metadata.Name
	}
	defaultFilename := fmt.Sprintf("%s.%s", projectName, strings.ToLower(targetFormat))

	// 3. 创建文件过滤器
	filters := []runtime.FileFilter{
		{
			DisplayName: fmt.Sprintf("%s Files (*.%s)", strings.ToUpper(targetFormat), strings.ToLower(targetFormat)),
			Pattern:     fmt.Sprintf("*.%s", strings.ToLower(targetFormat)),
		},
		{
			DisplayName: "All Files (*.*)",
			Pattern:     "*.*",
		},
	}

	// 4. 显示保存文件对话框
	filePath, err := runtime.SaveFileDialog(api.ctx, runtime.SaveDialogOptions{
		Title:           "Save Subtitle",
		DefaultFilename: defaultFilename,
		Filters:         filters,
		ShowHiddenFiles: true,
	})

	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "shellitem") || strings.Contains(msg, "cancel") || strings.Contains(msg, "canceled") || strings.Contains(msg, "cancelled") {
			resp.Success = true
			resp.Data = map[string]any{
				"filePath":  "",
				"fileName":  "",
				"cancelled": true,
			}
			return
		}
		resp.Msg = err.Error()
		return
	}

	if filePath == "" {
		resp.Success = true
		resp.Data = map[string]any{
			"filePath":  "",
			"fileName":  "",
			"cancelled": true,
		}
		return
	}

	// 5. 转换字幕内容
	subtitleData, err := api.subs.ConvertSubtitle(id, langCode, targetFormat)
	if err != nil {
		resp.Msg = err.Error()
		return
	}

	// 6. 写入文件
	err = os.WriteFile(filePath, subtitleData, 0644)
	if err != nil {
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	resp.Msg = "Save success"
	resp.Data = map[string]any{
		"filePath":  filePath,
		"fileName":  filepath.Base(filePath),
		"cancelled": false,
	}
	return
}

func (api *SubtitlesAPI) GetSubtitle(id string) (resp *types.JSResp) {
	if id == "" {
		return &types.JSResp{Msg: "id is empty"}
	}

	// get subtitle
	sub, err := api.subs.GetSubtitle(id)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}
	return &types.JSResp{Success: true, Data: sub}
}

func (api *SubtitlesAPI) ListSubtitles() (resp *types.JSResp) {
	subs, err := api.subs.ListSubtitles()
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}
	return &types.JSResp{Success: true, Data: subs}
}

func (api *SubtitlesAPI) DeleteSubtitle(id string) (resp *types.JSResp) {
	if id == "" {
		return &types.JSResp{Msg: "id is empty"}
	}

	// delete subtitle
	err := api.subs.DeleteSubtitle(id)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}
	return &types.JSResp{Success: true}
}

func (api *SubtitlesAPI) DeleteAllSubtitle() (resp *types.JSResp) {
	// delete subtitle
	err := api.subs.DeleteAllSubtitle()
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}
	return &types.JSResp{Success: true}
}

// UpdateProjectMetadata 更新项目元数据
func (api *SubtitlesAPI) UpdateProjectName(id, name string) (resp *types.JSResp) {
	if id == "" {
		return &types.JSResp{Msg: "id is empty"}
	}

	if name == "" {
		return &types.JSResp{Msg: "name is empty"}
	}

	project, err := api.subs.UpdateProjectName(id, name)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: project}
}

// UpdateProjectMetadata 更新项目元数据
func (api *SubtitlesAPI) UpdateProjectMetadata(id string, metadata types.ProjectMetadata) (resp *types.JSResp) {
	if id == "" {
		return &types.JSResp{Msg: "id is empty"}
	}

	project, err := api.subs.UpdateProjectMetadata(id, metadata)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: project}
}

// GetSubtitleLLMConversation 返回某个项目 + 目标语言下最近一次 LLM 翻译任务对应的对话记录
func (api *SubtitlesAPI) GetSubtitleLLMConversation(id string, langCode string) (resp *types.JSResp) {
	if id == "" || langCode == "" {
		return &types.JSResp{Msg: "id or langCode is empty"}
	}
	conv, err := api.subs.GetLLMConversationForLanguage(id, langCode)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}
	return &types.JSResp{Success: true, Data: conv}
}

// UpdateSubtitleSegment 更新单个字幕片段
func (api *SubtitlesAPI) UpdateSubtitleSegment(id string, segmentID string, segment *types.SubtitleSegment) (resp *types.JSResp) {
	if id == "" || segmentID == "" {
		return &types.JSResp{Msg: "id or segmentID is empty"}
	}

	project, err := api.subs.UpdateSubtitleSegment(id, segmentID, segment)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: project}
}

// UpdateLanguageContent 更新特定语言的内容
func (api *SubtitlesAPI) UpdateLanguageContent(id string, segmentID string, langCode string, content types.LanguageContent) (resp *types.JSResp) {
	if id == "" || segmentID == "" || langCode == "" {
		return &types.JSResp{Msg: "id, segmentID or langCode is empty"}
	}

	project, err := api.subs.UpdateLanguageContent(id, segmentID, langCode, content)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: project}
}

// UpdateLanguageMetadata 更新语言元数据
func (api *SubtitlesAPI) UpdateLanguageMetadata(id string, langCode string, metadata types.LanguageMetadata) (resp *types.JSResp) {
	if id == "" || langCode == "" {
		return &types.JSResp{Msg: "id or langCode is empty"}
	}

	project, err := api.subs.UpdateLanguageMetadata(id, langCode, metadata)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: project}
}

func (api *SubtitlesAPI) GetSupportedConverters() (resp *types.JSResp) {
	converters := api.subs.GetSupportedConverters()
	return &types.JSResp{Success: true, Data: converters}
}

func (api *SubtitlesAPI) ZHConvertSubtitle(id string, origin, converterString string) (resp *types.JSResp) {
    err := api.subs.ZHConvertSubtitle(id, origin, converterString)
    if err != nil {
        return &types.JSResp{Msg: err.Error()}
    }
    return &types.JSResp{Success: true}
}

// TranslateSubtitleLLM 触发基于 LLM 的字幕翻译（OpenAI 兼容）
// providerID: 已在“LLM Providers”中配置的 Provider 标识
// model: 具体模型名称，例如 gpt-4o-mini, qwen2.5, etc.
func (api *SubtitlesAPI) TranslateSubtitleLLM(id, origin, target, providerID, model string) (resp *types.JSResp) {
    logger.Info("API TranslateSubtitleLLM",
        zap.String("id", id),
        zap.String("origin", origin),
        zap.String("target", target),
        zap.String("providerID", providerID),
        zap.String("model", model),
    )
    if id == "" { return &types.JSResp{Msg: "id is empty"} }
    if origin == "" || target == "" { return &types.JSResp{Msg: "origin/target is empty"} }
    if providerID == "" || model == "" { return &types.JSResp{Msg: "providerID/model is empty"} }
    if err := api.subs.TranslateSubtitleLLM(id, origin, target, providerID, model); err != nil {
        return &types.JSResp{Msg: err.Error()}
    }
    // 异步执行：立即返回成功，后续通过 WebSocket 订阅进度
    return &types.JSResp{Success: true}
}

// TranslateSubtitleLLMWithOptions 支持选择全局术语表集合（多个）与本次任务仅用的临时术语。
// strictGlossary: 是否启用严格术语模式（不在 glossary 中暴露占位符，仅依赖占位符自身和后处理）。
func (api *SubtitlesAPI) TranslateSubtitleLLMWithOptions(id, origin, target, providerID, model string, setIDs []string, extras []types.GlossaryEntry, strictGlossary bool) (resp *types.JSResp) {
    logger.Info("API TranslateSubtitleLLMWithOptions",
        zap.String("id", id),
        zap.String("origin", origin),
        zap.String("target", target),
        zap.String("providerID", providerID),
        zap.String("model", model),
        zap.Int("setIDs", len(setIDs)),
        zap.Int("extras", len(extras)),
        zap.Bool("strictGlossary", strictGlossary),
    )
    if id == "" { return &types.JSResp{Msg: "id is empty"} }
    if origin == "" || target == "" { return &types.JSResp{Msg: "origin/target is empty"} }
    if providerID == "" || model == "" { return &types.JSResp{Msg: "providerID/model is empty"} }
    if err := api.subs.TranslateSubtitleLLMWithOptions(id, origin, target, providerID, model, setIDs, extras, strictGlossary); err != nil {
        return &types.JSResp{Msg: err.Error()}
    }
    return &types.JSResp{Success: true}
}

// TranslateSubtitleLLMRetryFailedWithOptions 仅重试失败/回退的片段。
// strictGlossary 同上。
func (api *SubtitlesAPI) TranslateSubtitleLLMRetryFailedWithOptions(id, origin, target, providerID, model string, setIDs []string, extras []types.GlossaryEntry, strictGlossary bool) (resp *types.JSResp) {
    logger.Info("API TranslateSubtitleLLMRetryFailedWithOptions",
        zap.String("id", id),
        zap.String("origin", origin),
        zap.String("target", target),
        zap.String("providerID", providerID),
        zap.String("model", model),
        zap.Int("setIDs", len(setIDs)),
        zap.Int("extras", len(extras)),
        zap.Bool("strictGlossary", strictGlossary),
    )
    if id == "" { return &types.JSResp{Msg: "id is empty"} }
    if origin == "" || target == "" { return &types.JSResp{Msg: "origin/target is empty"} }
    if providerID == "" || model == "" { return &types.JSResp{Msg: "providerID/model is empty"} }
    if err := api.subs.TranslateSubtitleLLMFailedOnlyWithOptions(id, origin, target, providerID, model, setIDs, extras, strictGlossary); err != nil {
        return &types.JSResp{Msg: err.Error()}
    }
    return &types.JSResp{Success: true}
}

// legacy bound-profile endpoints removed

// --- Global Profile versions (profile not bound to provider/model) ---
func (api *SubtitlesAPI) TranslateSubtitleLLMWithGlobalProfile(id, origin, target, providerID, model, profileID string) (resp *types.JSResp) {
    logger.Info("API TranslateSubtitleLLMWithGlobalProfile",
        zap.String("id", id),
        zap.String("origin", origin),
        zap.String("target", target),
        zap.String("providerID", providerID),
        zap.String("model", model),
        zap.String("profileID", profileID),
    )
    if id == "" { return &types.JSResp{Msg: "id is empty"} }
    if origin == "" || target == "" { return &types.JSResp{Msg: "origin/target is empty"} }
    if providerID == "" || model == "" || profileID == "" { return &types.JSResp{Msg: "providerID/model/profileID is empty"} }
    if err := api.subs.TranslateSubtitleLLMWithGlobalProfile(id, origin, target, providerID, model, profileID); err != nil {
        return &types.JSResp{Msg: err.Error()}
    }
    return &types.JSResp{Success: true}
}

// TranslateSubtitleLLMWithGlobalProfileOptions 支持全局 profile + 术语选项。
// strictGlossary: 是否启用严格术语模式。
func (api *SubtitlesAPI) TranslateSubtitleLLMWithGlobalProfileOptions(id, origin, target, providerID, model, profileID string, setIDs []string, extras []types.GlossaryEntry, strictGlossary bool) (resp *types.JSResp) {
    logger.Info("API TranslateSubtitleLLMWithGlobalProfileOptions",
        zap.String("id", id),
        zap.String("origin", origin),
        zap.String("target", target),
        zap.String("providerID", providerID),
        zap.String("model", model),
        zap.String("profileID", profileID),
        zap.Int("setIDs", len(setIDs)),
        zap.Int("extras", len(extras)),
        zap.Bool("strictGlossary", strictGlossary),
    )
    if id == "" { return &types.JSResp{Msg: "id is empty"} }
    if origin == "" || target == "" { return &types.JSResp{Msg: "origin/target is empty"} }
    if providerID == "" || model == "" || profileID == "" { return &types.JSResp{Msg: "providerID/model/profileID is empty"} }
    if err := api.subs.TranslateSubtitleLLMWithGlobalProfileWithOptions(id, origin, target, providerID, model, profileID, setIDs, extras, strictGlossary); err != nil {
        return &types.JSResp{Msg: err.Error()}
    }
    return &types.JSResp{Success: true}
}

// TranslateSubtitleLLMRetryFailedWithGlobalProfileOptions 仅重试失败/回退片段（全局 profile）。
// strictGlossary 同上。
func (api *SubtitlesAPI) TranslateSubtitleLLMRetryFailedWithGlobalProfileOptions(id, origin, target, providerID, model, profileID string, setIDs []string, extras []types.GlossaryEntry, strictGlossary bool) (resp *types.JSResp) {
    logger.Info("API TranslateSubtitleLLMRetryFailedWithGlobalProfileOptions",
        zap.String("id", id),
        zap.String("origin", origin),
        zap.String("target", target),
        zap.String("providerID", providerID),
        zap.String("model", model),
        zap.String("profileID", profileID),
        zap.Int("setIDs", len(setIDs)),
        zap.Int("extras", len(extras)),
        zap.Bool("strictGlossary", strictGlossary),
    )
    if id == "" { return &types.JSResp{Msg: "id is empty"} }
    if origin == "" || target == "" { return &types.JSResp{Msg: "origin/target is empty"} }
    if providerID == "" || model == "" || profileID == "" { return &types.JSResp{Msg: "providerID/model/profileID is empty"} }
    if err := api.subs.TranslateSubtitleLLMFailedOnlyWithGlobalProfileWithOptions(id, origin, target, providerID, model, profileID, setIDs, extras, strictGlossary); err != nil {
        return &types.JSResp{Msg: err.Error()}
    }
    return &types.JSResp{Success: true}
}

// --- Glossary CRUD ---
func (api *SubtitlesAPI) ListGlossary() (resp *types.JSResp) {
    list, err := api.subs.ListGlossary()
    if err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true, Data: list}
}

func (api *SubtitlesAPI) ListGlossaryBySet(setID string) (resp *types.JSResp) {
    list, err := api.subs.ListGlossaryBySet(setID)
    if err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true, Data: list}
}

func (api *SubtitlesAPI) UpsertGlossaryEntry(entry types.GlossaryEntry) (resp *types.JSResp) {
    e := entry // copy
    out, err := api.subs.UpsertGlossaryEntry(&e)
    if err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true, Data: out}
}

func (api *SubtitlesAPI) DeleteGlossaryEntry(id string) (resp *types.JSResp) {
    if id == "" { return &types.JSResp{Msg: "id is empty"} }
    if err := api.subs.DeleteGlossaryEntry(id); err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true}
}

// Glossary Sets
func (api *SubtitlesAPI) ListGlossarySets() (resp *types.JSResp) {
    list, err := api.subs.ListGlossarySets()
    if err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true, Data: list}
}

func (api *SubtitlesAPI) UpsertGlossarySet(gs types.GlossarySet) (resp *types.JSResp) {
    g := gs
    out, err := api.subs.UpsertGlossarySet(&g)
    if err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true, Data: out}
}

func (api *SubtitlesAPI) DeleteGlossarySet(id string) (resp *types.JSResp) {
    if id == "" { return &types.JSResp{Msg: "id is empty"} }
    if err := api.subs.DeleteGlossarySet(id); err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true}
}

// --- Target Languages CRUD ---
func (api *SubtitlesAPI) ListTargetLanguages() (resp *types.JSResp) {
    list, err := api.subs.ListTargetLanguages()
    if err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true, Data: list}
}

func (api *SubtitlesAPI) UpsertTargetLanguage(l types.TargetLanguage) (resp *types.JSResp) {
    tl := l
    out, err := api.subs.UpsertTargetLanguage(&tl)
    if err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true, Data: out}
}

func (api *SubtitlesAPI) DeleteTargetLanguage(code string) (resp *types.JSResp) {
    if code == "" { return &types.JSResp{Msg: "code is empty"} }
    if err := api.subs.DeleteTargetLanguage(code); err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true}
}

// RemoveProjectLanguage 删除项目中的一种翻译语言（不可删除原始语言）
func (api *SubtitlesAPI) RemoveProjectLanguage(id, langCode string) (resp *types.JSResp) {
    if id == "" || langCode == "" { return &types.JSResp{Msg: "id or langCode is empty"} }
    p, err := api.subs.RemoveProjectLanguage(id, langCode)
    if err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true, Data: p}
}

// ResetTargetLanguagesToDefault clears and restores default target languages
func (api *SubtitlesAPI) ResetTargetLanguagesToDefault() (resp *types.JSResp) {
    if err := api.subs.ResetTargetLanguagesToDefault(); err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true}
}

// --- LLM Tasks History (List/Get/Delete/Retry) ---

func (api *SubtitlesAPI) ListLLMTasks() (resp *types.JSResp) {
    list, err := api.subs.ListLLMTasks("")
    if err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true, Data: list}
}

func (api *SubtitlesAPI) ListLLMTasksByProject(projectID string) (resp *types.JSResp) {
    if projectID == "" { return &types.JSResp{Msg: "projectID is empty"} }
    list, err := api.subs.ListLLMTasks(projectID)
    if err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true, Data: list}
}

func (api *SubtitlesAPI) GetLLMTask(taskID string) (resp *types.JSResp) {
    if taskID == "" { return &types.JSResp{Msg: "taskID is empty"} }
    v, err := api.subs.GetLLMTask(taskID)
    if err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true, Data: v}
}

func (api *SubtitlesAPI) DeleteLLMTask(taskID string) (resp *types.JSResp) {
    if taskID == "" { return &types.JSResp{Msg: "taskID is empty"} }
    if err := api.subs.DeleteLLMTask(taskID); err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true}
}

func (api *SubtitlesAPI) RetryLLMTask(taskID, providerID, model string) (resp *types.JSResp) {
    if taskID == "" { return &types.JSResp{Msg: "taskID is empty"} }
    if providerID == "" || model == "" { return &types.JSResp{Msg: "providerID/model is empty"} }
    if err := api.subs.RetryLLMTaskFailedOnly(taskID, providerID, model); err != nil { return &types.JSResp{Msg: err.Error()} }
    return &types.JSResp{Success: true}
}

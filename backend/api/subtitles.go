package api

import (
	"CanMe/backend/core/subtitles"
	"CanMe/backend/types"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"encoding/json"
)

type SubtitlesAPI struct {
	ctx  context.Context
	subs *subtitles.Service
}

func NewSubtitlesAPI(subs *subtitles.Service) *SubtitlesAPI {
	return &SubtitlesAPI{
		subs: subs,
	}
}

func (api *SubtitlesAPI) Subscribe(ctx context.Context) {
	api.ctx = ctx
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
	contentString, err := json.Marshal(info)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}
	content := string(contentString)
	return &types.JSResp{Success: true, Data: content}
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

	contentString, err := json.Marshal(project)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	content := string(contentString)
	return &types.JSResp{Success: true, Data: content}
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
	projectName := "Subtile_Project"
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
		resp.Msg = err.Error()
		return
	}

	if filePath == "" {
		resp.Msg = "User canceled"
		return
	}

	// 5. 转换字幕内容
	subtitleData, err := api.subs.ConvertSubtile(id, langCode, targetFormat)
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
	resp.Data = map[string]string{
		"filePath": filePath,
		"fileName": filepath.Base(filePath),
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
	contentString, err := json.Marshal(sub)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}
	content := string(contentString)
	return &types.JSResp{Success: true, Data: content}
}

func (api *SubtitlesAPI) ListSubtitles() (resp *types.JSResp) {
	subs, err := api.subs.ListSubtitles()
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}
	contentString, err := json.Marshal(subs)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}
	content := string(contentString)
	return &types.JSResp{Success: true, Data: content}
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

	contentString, err := json.Marshal(project)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(contentString)}
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

	contentString, err := json.Marshal(project)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(contentString)}
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

	contentString, err := json.Marshal(project)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(contentString)}
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

	contentString, err := json.Marshal(project)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(contentString)}
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

	contentString, err := json.Marshal(project)
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true, Data: string(contentString)}
}

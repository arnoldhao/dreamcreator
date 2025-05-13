package types

import (
	"time"

	"github.com/lrstanley/go-ytdlp"
)

type DtDownloadRequest struct {
	URL      string `json:"url"`
	FormatID string `json:"formatId"`
	// 下载字幕
	DownloadSubs bool     `json:"downloadSubs"`
	SubLangs     []string `json:"subLangs"`  // 例如 ["en", "zh-CN"]
	SubFormat    string   `json:"subFormat"` // 例如 "srt", "vtt", "best"

	// 翻译字幕
	TranslateTo   string `json:"translateTo"`
	SubtitleStyle string `json:"subtitleStyle"`

	// Recode
	RecodeFormatNumber int `json:"recodeFormatNumber"`
}

type DtDownloadResponse struct {
	ID     string      `json:"id"`
	Status DtTaskStage `json:"status"`
}

type DtQuickDownloadRequest struct {
	URL                string `json:"url"`
	Video              string `json:"video"`
	BestCaption        bool   `json:"bestCaption"`
	Type               string `json:"type"`
	RecodeFormatNumber int    `json:"recodeFormatNumber"`
	RecodeExtention    string `json:"recodeExtention"`
}

type DtQuickDownloadResponse struct {
	ID     string      `json:"id"`
	Status DtTaskStage `json:"status"`
}

type DownloadVideoRequest struct {
	// type
	Type string `json:"type"`

	URL string `json:"url"`
	// best options
	Video       string `json:"video"`
	BestCaption bool   `json:"bestCaption"`
	// custom options
	FormatID     string   `json:"formatId"`
	DownloadSubs bool     `json:"downloadSubs"`
	SubLangs     []string `json:"subLangs"`  // 例如 ["en", "zh-CN"]
	SubFormat    string   `json:"subFormat"` // 例如 "srt", "vtt", "best"
	// translate options
	TranslateTo   string `json:"translateTo"`
	SubtitleStyle string `json:"subtitleStyle"`
}

// DtTaskStage 定义处理阶段
type DtTaskStage string

const (
	DtStageInitializing DtTaskStage = "initializing" // 初始化阶段
	DtStageDownloading  DtTaskStage = "downloading"  // 视频下载阶段
	DtStageTranslating  DtTaskStage = "translating"  // 字幕翻译阶段
	DtStageEmbedding    DtTaskStage = "embedding"    // 字幕嵌入阶段
	DtStageCompleted    DtTaskStage = "completed"    // 处理完成
	DtStageFailed       DtTaskStage = "failed"       // 处理失败
	DtStageCancelled    DtTaskStage = "cancelled"    // 处理取消
	DtStageInstalling   DtTaskStage = "installing"   // 安装阶段
	DtStageInstalled    DtTaskStage = "installed"    // 安装完成
	DtStageUpdating     DtTaskStage = "updating"     // 更新阶段
	DtStageUpdated      DtTaskStage = "updated"      // 更新完成
)

// DtProgress 表示处理进度信息
type DtProgress struct {
	// 基础信息
	ID string `json:"id"`

	// type
	Type string `json:"type"`

	// 状态
	Stage     DtTaskStage `json:"stage"`               // 当前处理阶段
	StageInfo string      `json:"stageInfo,omitempty"` // 当前阶段的额外信息
	Error     string      `json:"error,omitempty"`     // 错误信息

	// 进度信息
	Percentage    float64 `json:"percentage"`              // 当前阶段的进度百分比
	Speed         string  `json:"speed,omitempty"`         // 下载速度
	Downloaded    string  `json:"downloaded,omitempty"`    // 已下载大小（仅下载阶段有效）
	TotalSize     string  `json:"totalSize,omitempty"`     // 总大小（仅下载阶段有效）
	EstimatedTime string  `json:"estimatedTime,omitempty"` // 预计剩余时间
}

// DtTaskStatus 用于在数据库中存储任务状态
type DtTaskStatus struct {
	// 基础信息
	ID string `json:"id"`

	// type
	Type string `json:"type"` // custom, quick, mcp

	// 请求信息
	DownloadSubs  bool     `json:"downloadSubs"`
	SubLangs      []string `json:"subLangs"`  // 例如 ["en", "zh-CN"]
	SubFormat     string   `json:"subFormat"` // 例如 "srt", "vtt", "best"
	TranslateTo   string   `json:"translateTo"`
	SubtitleStyle string   `json:"subtitleStyle"`

	// Recode
	RecodeFormatNumber int    `json:"recodeFormatNumber"`
	RecodeExtention    string `json:"recodeExtention"`

	// 状态
	Stage     DtTaskStage `json:"stage"`               // 当前处理阶段
	StageInfo string      `json:"stageInfo,omitempty"` // 当前阶段的额外信息
	Error     string      `json:"error,omitempty"`     // 错误信息

	// 文件存储
	OutputDir          string   `json:"outputDir,omitempty"`          // 输出目录
	VideoFiles         []string `json:"videoFiles,omitempty"`         // 下载的视频文件名
	SubtitleFiles      []string `json:"subtitleFiles,omitempty"`      // 下载的原始字幕文件名
	AllDownloadedFiles []string `json:"allDownloadedFiles,omitempty"` // 所有最终下载的文件名
	TranslatedSubs     []string `json:"translatedSubs,omitempty"`     // 翻译后的字幕文件名
	EmbeddedVideoFiles []string `json:"embeddedVideoFiles,omitempty"` // 嵌入的视频文件名
	AllFiles           []string `json:"allFiles,omitempty"`           // 所有产生的文件名

	// 核心元数据字段
	Extractor  string  `json:"extractor,omitempty"`  // 提取器
	Title      string  `json:"title,omitempty"`      // 视频标题
	Thumbnail  string  `json:"thumbnail,omitempty"`  // 缩略图URL
	URL        string  `json:"url,omitempty"`        // 原始视频URL
	FormatID   string  `json:"formatId,omitempty"`   // 视频质量
	Resolution string  `json:"resolution,omitempty"` // 视频分辨率
	Uploader   string  `json:"uploader,omitempty"`   // 作者/频道名
	Duration   float64 `json:"duration,omitempty"`   // 视频时长（秒`)
	FileSize   int64   `json:"fileSize,omitempty"`   // 文件大小（字节）
	Format     string  `json:"format,omitempty"`     // 视频格式

	// 进度信息
	Percentage    float64 `json:"percentage"`              // 当前阶段的进度百分比
	Speed         string  `json:"speed,omitempty"`         // 下载速度
	EstimatedTime string  `json:"estimatedTime,omitempty"` // 预计剩余时间

	// 时间戳
	CreatedAt int64 `json:"createdAt"`
	UpdatedAt int64 `json:"updatedAt"`
}

// UpdateFromProgress updates the task status based on progress information
func (t *DtTaskStatus) UpdateFromProgress(progress *DtProgress) {
	// Update current stage
	t.Stage = progress.Stage
	t.UpdatedAt = time.Now().Unix()

	// 更新进度信息
	if progress.Percentage > 0 {
		t.Percentage = progress.Percentage
	}

	// 更新下载速度
	if progress.Speed != "" {
		t.Speed = progress.Speed
	}

	// 更新预计剩余时间
	if progress.EstimatedTime != "" {
		t.EstimatedTime = progress.EstimatedTime
	}

	// Update error information if present
	if progress.Error != "" {
		t.Error = progress.Error
	}
}

type SoftwareInfo struct {
	Available     bool   `json:"available"`
	Path          string `json:"path"`
	ExecPath      string `json:"execPath"`
	Version       string `json:"version"`
	LatestVersion string `json:"latestVersion"`
	NeedUpdate    bool   `json:"needUpdate"`
}

type FillTaskInfo struct {
	ID   string               `json:"id"`
	Info *ytdlp.ExtractedInfo `json:"info"`
}

type DTSignal struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Stage   DtTaskStage `json:"stage"` // 当前处理阶段
	Refresh bool        `json:"refresh"`
}

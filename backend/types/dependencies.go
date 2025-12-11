package types

import (
	"time"
)

// DependencyType 依赖类型
type DependencyType string

const (
	DependencyFFmpeg DependencyType = "ffmpeg"
	DependencyYTDLP  DependencyType = "yt-dlp"
	DependencyDeno   DependencyType = "deno"
)

// DependencyInfo 依赖信息
type DependencyInfo struct {
	Type          DependencyType `json:"type"`
	Name          string         `json:"name"`
	Version       string         `json:"version"`
	LatestVersion string         `json:"latestVersion"`
	Path          string         `json:"path"`
	ExecPath      string         `json:"execPath"`
	Available     bool           `json:"available"`
	NeedUpdate    bool           `json:"needUpdate"`
	LastCheck     time.Time      `json:"lastCheck"`
	// Last check result for update/version check
	LastCheckAttempted bool   `json:"lastCheckAttempted"`
	LastCheckSuccess   bool   `json:"lastCheckSuccess"`
	LastCheckError     string `json:"lastCheckError"`
	LastCheckErrorCode string `json:"lastCheckErrorCode"`
}

// DownloadConfig 下载配置
type DownloadConfig struct {
	Version     string
	BaseURL     string
	Mirror      string // 镜像源名称
	ForceUpdate bool
	Timeout     time.Duration
}

// MirrorInfo 镜像源信息
type MirrorInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Region      string `json:"region"`
	Speed       string `json:"speed"`
	Available   bool   `json:"available"`
	Recommended bool   `json:"recommended"`
}

// DependencyCleanStats 描述单个依赖类型的清理统计信息
type DependencyCleanStats struct {
	Type         DependencyType `json:"type"`
	RemovedPaths []string       `json:"removedPaths"`
	FreedBytes   int64          `json:"freedBytes"`
}

// DependencyCleanResult 描述一次依赖清理操作的总体结果
type DependencyCleanResult struct {
	TotalFreedBytes int64                  `json:"totalFreedBytes"`
	Stats           []DependencyCleanStats `json:"stats"`
}

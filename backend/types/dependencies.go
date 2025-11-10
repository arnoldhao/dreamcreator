package types

import (
	"time"
)

// DependencyType 依赖类型
type DependencyType string

const (
	DependencyFFmpeg DependencyType = "ffmpeg"
	DependencyYTDLP  DependencyType = "yt-dlp"
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

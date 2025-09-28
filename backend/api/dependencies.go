package api

import (
	"context"
	"dreamcreator/backend/consts"
	"encoding/json"
	"runtime"
	"time"

	"dreamcreator/backend/core/downtasks"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/types"

	"go.uber.org/zap"
)

// DependenciesAPI 依赖管理 API
type DependenciesAPI struct {
	ctx              context.Context
	downtasksService *downtasks.Service
}

func NewDependenciesAPI(downtasksService *downtasks.Service) *DependenciesAPI {
	return &DependenciesAPI{
		downtasksService: downtasksService,
	}
}

func (api *DependenciesAPI) Subscribe(ctx context.Context) {
	api.ctx = ctx
}

// ListDependencies 列出所有依赖
func (api *DependenciesAPI) ListDependencies() types.JSResp {
	deps, err := api.downtasksService.ListDependencies()
	if err != nil {
		return types.JSResp{
			Success: false,
			Msg:     err.Error(),
		}
	}

	data, _ := json.Marshal(deps)

	return types.JSResp{
		Success: true,
		Data:    string(data),
	}
}

func (api *DependenciesAPI) DependenciesReady() (resp *types.JSResp) {
	ready, err := api.downtasksService.DependenciesReady()
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	if !ready {
		return &types.JSResp{Msg: "Dependencies not ready"}
	}

	return &types.JSResp{Success: true}
}

func (api *DependenciesAPI) ValidateDependencies() (resp *types.JSResp) {
	err := api.downtasksService.ValidateDependencies()
	if err != nil {
		return &types.JSResp{Msg: err.Error()}
	}

	return &types.JSResp{Success: true}
}

// QuickValidateDependencies 仅快速验证本地可执行是否可用
func (api *DependenciesAPI) QuickValidateDependencies() types.JSResp {
	results, err := api.downtasksService.QuickValidateDependencies()
	if err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	data, _ := json.Marshal(results)
	return types.JSResp{Success: true, Data: string(data)}
}

// UpdateDependencyWithMirror 使用指定镜像更新依赖
func (api *DependenciesAPI) UpdateDependencyWithMirror(depType string, mirror string) types.JSResp {
	var depTypeEnum types.DependencyType
	switch depType {
	case "yt-dlp":
		depTypeEnum = types.DependencyYTDLP
	case "ffmpeg":
		depTypeEnum = types.DependencyFFmpeg
	default:
		return types.JSResp{
			Success: false,
			Msg:     "unsupported dependency type",
		}
	}

	config := types.DownloadConfig{
		Mirror:  mirror,
		Timeout: 30 * time.Minute,
	}

	info, err := api.downtasksService.UpdateDependencyWithMirror(depTypeEnum, config)
	if err != nil {
		return types.JSResp{
			Success: false,
			Msg:     err.Error(),
		}
	}

	data, _ := json.Marshal(info)
	return types.JSResp{
		Success: true,
		Data:    string(data),
	}
}

// CheckUpdates 检查更新
func (api *DependenciesAPI) CheckUpdates() types.JSResp {
	updates, err := api.downtasksService.CheckDependencyUpdates()
	if err != nil {
		return types.JSResp{
			Success: false,
			Msg:     err.Error(),
		}
	}

	data, _ := json.Marshal(updates)
	return types.JSResp{
		Success: true,
		Data:    string(data),
	}
}

// ListMirrors 获取指定依赖类型的可用镜像源
func (api *DependenciesAPI) ListMirrors(depType string) types.JSResp {
	var mirrors map[string]consts.Mirror
	var availableMirrors []types.MirrorInfo

	osType := runtime.GOOS
	arch := runtime.GOARCH

	switch depType {
	case "ffmpeg":
		mirrors = consts.FFmpegMirrors
		// 检查FFmpeg镜像可用性
		if platformURLs, exists := consts.FFmpegDownloadURLs[osType]; exists {
			if archURLs, exists := platformURLs[arch]; exists {
				for mirrorName := range archURLs {
					if mirror, exists := mirrors[mirrorName]; exists {
						availableMirrors = append(availableMirrors, types.MirrorInfo{
							Name:        mirror.Name,
							DisplayName: mirror.DisplayName,
							Description: mirror.Description,
							Region:      mirror.Region,
							Speed:       mirror.Speed,
							Available:   true,
							Recommended: mirrorName == consts.GetRecommendedMirror("ffmpeg", osType),
						})
					}
				}
			}
		}

	case "yt-dlp":
		mirrors = consts.YTDLPMirrors
		// YTDLP所有镜像都支持所有平台
		for mirrorName, mirror := range mirrors {
			availableMirrors = append(availableMirrors, types.MirrorInfo{
				Name:        mirror.Name,
				DisplayName: mirror.DisplayName,
				Description: mirror.Description,
				Region:      mirror.Region,
				Speed:       mirror.Speed,
				Available:   true,
				Recommended: mirrorName == consts.GetRecommendedMirror("yt-dlp", osType),
			})
		}

	default:
		return types.JSResp{
			Success: false,
			Msg:     "unsupported dependency type",
		}
	}

	data, _ := json.Marshal(availableMirrors)
	return types.JSResp{
		Success: true,
		Data:    string(data),
	}
}

// InstallDependencyWithMirror 使用指定镜像安装依赖
func (api *DependenciesAPI) InstallDependencyWithMirror(depType string, version string, mirror string) types.JSResp {
	logger.Info("Starting dependency installation",
		zap.String("type", depType),
		zap.String("version", version),
		zap.String("mirror", mirror))

	var depTypeEnum types.DependencyType
	switch depType {
	case "yt-dlp":
		depTypeEnum = types.DependencyYTDLP
	case "ffmpeg":
		depTypeEnum = types.DependencyFFmpeg
	default:
		logger.Error("Unsupported dependency type", zap.String("type", depType))
		return types.JSResp{
			Success: false,
			Msg:     "unsupported dependency type",
		}
	}

	config := types.DownloadConfig{
		Version: version,
		Mirror:  mirror,
		Timeout: 30 * time.Minute,
	}

	logger.Info("Calling downtasks service to install dependency",
		zap.String("type", string(depTypeEnum)),
		zap.Any("config", config))

	info, err := api.downtasksService.InstallDependency(depTypeEnum, config)
	if err != nil {
		logger.Error("Failed to install dependency",
			zap.String("type", string(depTypeEnum)),
			zap.Error(err),
			zap.String("version", version),
			zap.String("mirror", mirror))
		return types.JSResp{
			Success: false,
			Msg:     err.Error(),
		}
	}

	logger.Info("Dependency installation completed successfully",
		zap.String("type", string(depTypeEnum)),
		zap.String("execPath", info.ExecPath),
		zap.String("version", info.Version))

	data, _ := json.Marshal(info)
	return types.JSResp{
		Success: true,
		Data:    string(data),
	}
}

func (api *DependenciesAPI) RepairDependency(depType string) types.JSResp {
	var depTypeEnum types.DependencyType
	switch depType {
	case "yt-dlp":
		depTypeEnum = types.DependencyYTDLP
	case "ffmpeg":
		depTypeEnum = types.DependencyFFmpeg
	default:
		logger.Error("Unsupported dependency type", zap.String("type", depType))
		return types.JSResp{
			Success: false,
			Msg:     "unsupported dependency type",
		}
	}
	err := api.downtasksService.RepairDependency(depTypeEnum)
	if err != nil {
		logger.Error("Failed to repair dependency", zap.String("type", string(depTypeEnum)), zap.Error(err))
		return types.JSResp{
			Success: false,
			Msg:     err.Error(),
		}
	}
	return types.JSResp{
		Success: true,
	}
}

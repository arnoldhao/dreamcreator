package downtasks

import (
	"CanMe/backend/types"
)

// GetYTDLPPath 获取 yt-dlp 可执行文件的文件夹路径
func (s *Service) YTDLPPath() (string, error) {
	return s.executablePath(types.DependencyYTDLP)
}

// YTDLPExecPath 获取 yt-dlp 可执行文件路径
func (s *Service) YTDLPExecPath() (string, error) {
	return s.executableExecPath(types.DependencyYTDLP)
}

// GetFFMPEGPath 获取 FFmpeg 可执行文件的文件夹路径
func (s *Service) FFMPEGPath() (string, error) {
	return s.executablePath(types.DependencyFFmpeg)
}

// FFMPEGExecPath 获取 FFmpeg 可执行文件路径
func (s *Service) FFMPEGExecPath() (string, error) {
	return s.executableExecPath(types.DependencyFFmpeg)
}

// InstallDependency 安装依赖
func (s *Service) InstallDependency(depType types.DependencyType, config types.DownloadConfig) (*types.DependencyInfo, error) {
	return s.depManager.Install(s.ctx, depType, config)
}

// UpdateDependencyWithMirror 使用指定镜像更新依赖
func (s *Service) UpdateDependencyWithMirror(depType types.DependencyType, config types.DownloadConfig) (*types.DependencyInfo, error) {
	return s.depManager.UpdateWithMirror(s.ctx, depType, config)
}

// ListDependencies 列出所有依赖
func (s *Service) ListDependencies() (map[types.DependencyType]*types.DependencyInfo, error) {
	return s.depManager.List(s.ctx)
}

// CheckDependencyUpdates 检查依赖更新
func (s *Service) CheckDependencyUpdates() (map[types.DependencyType]*types.DependencyInfo, error) {
	return s.depManager.CheckUpdates(s.ctx)
}

// DependenciesReady 检查所有依赖是否已准备好
func (s *Service) DependenciesReady() (bool, error) {
	return s.depManager.DependenciesReady(s.ctx)
}

// ValidateDependencies 验证所有依赖可用性
func (s *Service) ValidateDependencies() error {
    return s.depManager.ValidateDependencies(s.ctx)
}

func (s *Service) RepairDependency(depType types.DependencyType) error {
    return s.depManager.RepairDependency(s.ctx, depType)
}

// QuickValidateDependencies 仅快速验证本地可执行是否可用
func (s *Service) QuickValidateDependencies() (map[types.DependencyType]*types.DependencyInfo, error) {
    return s.depManager.QuickValidate(s.ctx)
}

// executablePath 获取可执行文件的文件夹路径
func (s *Service) executablePath(depType types.DependencyType) (string, error) {
	info, err := s.depManager.Get(s.ctx, depType)
	if err != nil {
		return "", err
	}

	return info.Path, nil
}

// executableExecPath 获取可执行文件路径
func (s *Service) executableExecPath(depType types.DependencyType) (string, error) {
	info, err := s.depManager.Get(s.ctx, depType)
	if err != nil {
		return "", err
	}

	return info.ExecPath, nil
}

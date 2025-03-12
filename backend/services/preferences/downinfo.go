package preferences

import (
	"CanMe/backend/pkg/downinfo"
	"CanMe/backend/types"
	"fmt"
	"log"
)

// 下载配置变更回调函数类型
type DownloadInfoChangedCallback func(*downinfo.Config)

// 下载配置变更回调列表
var downloadInfoChangedCallbacks []DownloadInfoChangedCallback

// OnDownloadInfoChanged 注册下载配置变更回调
func (s *Service) OnDownloadInfoChanged(callback DownloadInfoChangedCallback) {
	downloadInfoChangedCallbacks = append(downloadInfoChangedCallbacks, callback)
}

// triggerDownloadInfoChangedCallbacks 触发所有下载配置变更回调
func (s *Service) triggerDownloadInfoChangedCallbacks(config *downinfo.Config) {
	for _, callback := range downloadInfoChangedCallbacks {
		callback(config)
	}
}

// SetDownloadConfig 设置下载配置
func (s *Service) SetDownloadConfig(config *downinfo.Config) (resp types.JSResp) {
	// 触发下载配置变更回调
	s.triggerDownloadInfoChangedCallbacks(config)

	// 将下载配置转换为偏好设置格式并保存
	pref := s.pref.GetPreferences()

	// 更新下载目录
	pref.Download.Dir = config.Dir

	// 保存更新后的偏好设置
	err := s.pref.SetPreferences(&pref)
	if err != nil {
		resp.Msg = fmt.Sprintf("save failed: %v", err)
		return
	}

	// 应用下载配置
	s.applyDownloadConfig(config)

	resp.Success = true
	return
}

// GetDownloadConfig 获取当前下载配置
func (s *Service) GetDownloadConfig() (resp types.JSResp) {
	// 从偏好设置中获取下载配置
	pref := s.pref.GetPreferences()

	// 创建下载配置对象
	config := &downinfo.Config{
		Dir: pref.Download.Dir,
	}

	// 如果下载目录为空，使用默认值
	if config.Dir == "" {
		config.Dir = downinfo.GetDefaultDownloadDir()
	}

	resp.Success = true
	resp.Data = config
	return
}

// CheckAndUpdateDownloadInfo 检查并更新下载配置
// 这个方法应该在应用启动时调用，以确保下载配置正确初始化
func (s *Service) CheckAndUpdateDownloadInfo() {
	resp := s.GetDownloadConfig()
	if resp.Success && resp.Data != nil {
		config, ok := resp.Data.(*downinfo.Config)
		if ok && config != nil {
			log.Printf("initializing download config")
			// 应用下载配置
			s.applyDownloadConfig(config)
			// 触发回调
			s.triggerDownloadInfoChangedCallbacks(config)
		}
	}
}

// applyDownloadConfig 应用下载配置到系统
// 这个方法封装了对底层下载实现的调用，使服务层和存储层解耦
func (s *Service) applyDownloadConfig(config *downinfo.Config) {
	// 检查下载客户端是否已设置
	if s.downloadClient == nil {
		log.Printf("warning: download client not set, cannot apply download config")
		return
	}

	// 设置下载目录
	if config.Dir != "" {
		s.downloadClient.SetConfig(config)
		log.Printf("download directory set: %s", config.Dir)
	}
}

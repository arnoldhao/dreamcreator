package preferences

import (
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/pkg/proxy"
	"dreamcreator/backend/types"
	"fmt"
	"go.uber.org/zap"
)

// 代理配置变更回调函数类型
type ProxyChangedCallback func(*proxy.Config)

// 代理配置变更回调列表
var proxyChangedCallbacks []ProxyChangedCallback

// OnProxyChanged 注册代理配置变更回调
func (s *Service) OnProxyChanged(callback ProxyChangedCallback) {
	proxyChangedCallbacks = append(proxyChangedCallbacks, callback)
}

// triggerProxyChangedCallbacks 触发所有代理配置变更回调
func (s *Service) triggerProxyChangedCallbacks(config *proxy.Config) {
	for _, callback := range proxyChangedCallbacks {
		callback(config)
	}
}

// SetProxyConfig 设置代理配置
func (s *Service) SetProxyConfig(config *proxy.Config) (resp types.JSResp) {
	// 触发代理配置变更回调
	s.triggerProxyChangedCallbacks(config)

	// 将代理配置转换为偏好设置格式并保存
	pref := s.pref.GetPreferences()

	if config.Type == "none" {
		// 禁用代理
		pref.Proxy.Type = "none"
		pref.Proxy.ProxyAddress = ""
	} else if config.Type == "system" {
		// 使用系统代理
		pref.Proxy.Type = "system"
		pref.Proxy.ProxyAddress = ""
	} else if config.Type == "manual" {
		// 启用手动代理
		pref.Proxy.Type = "manual"
		pref.Proxy.ProxyAddress = config.ProxyAddress
	}

	// 保存更新后的偏好设置
	err := s.pref.SetPreferences(&pref)
	if err != nil {
		resp.Msg = fmt.Sprintf("save failed: %v", err)
		return
	}

	// 应用代理配置
	s.applyProxyConfig(config)

	resp.Success = true
	return
}

// GetProxyConfig 获取当前代理配置
func (s *Service) GetProxyConfig() (resp types.JSResp) {
	// 从偏好设置中获取代理配置
	pref := s.pref.GetPreferences()

	// 创建代理配置对象
	config := &proxy.Config{}

	if pref.Proxy.Type == "manual" {
		// 设置为手动代理
		config.Type = "manual"
		config.ProxyAddress = pref.Proxy.ProxyAddress
	} else if pref.Proxy.Type == "system" {
		// 设置为系统代理
		config.Type = "system"
	} else {
		// 禁用代理
		config.Type = "none"
	}

	resp.Success = true
	resp.Data = config
	return
}

// CheckAndUpdateProxy 检查并更新代理配置
// 这个方法应该在应用启动时调用，以确保代理配置正确初始化
func (s *Service) CheckAndUpdateProxy() {
	resp := s.GetProxyConfig()
	if resp.Success && resp.Data != nil {
		config, ok := resp.Data.(*proxy.Config)
		if ok && config != nil {
			logger.Info("initializing proxy config")
			// 应用代理配置
			s.applyProxyConfig(config)
			// 触发回调
			s.triggerProxyChangedCallbacks(config)
		}
	}
}

// applyProxyConfig 应用代理配置到系统
// 这个方法封装了对底层代理实现的调用，使服务层和存储层解耦
func (s *Service) applyProxyConfig(config *proxy.Config) {
	// 检查代理客户端是否已设置
	if s.proxyManager == nil {
		logger.Warn("proxy client not set, cannot apply proxy config")
		return
	}

	switch config.Type {
	case "none":
		if err := s.proxyManager.DisableProxy(); err != nil {
			logger.Error("disable proxy failed", zap.Error(err))
		} else {
			logger.Info("proxy disabled")
		}
	case "system":
		if err := s.proxyManager.UpdateSystemProxy(); err != nil {
			logger.Error("enable system proxy failed", zap.Error(err))
		} else {
			logger.Info("using system proxy")
		}
	case "manual":
		if config.ProxyAddress == "" {
			logger.Warn("manual proxy requested but address is empty")
			return
		}
		if err := s.proxyManager.SetManualProxy(config.ProxyAddress); err != nil {
			logger.Error("set manual proxy failed", zap.String("proxy", config.ProxyAddress), zap.Error(err))
		} else {
			logger.Info("proxy set", zap.String("proxy", config.ProxyAddress))
		}
	default:
		// 回退为禁用
		if err := s.proxyManager.DisableProxy(); err != nil {
			logger.Error("fallback disable proxy failed", zap.Error(err))
		} else {
			logger.Info("proxy disabled (fallback)")
		}
	}
}

package proxy

import "net/http"

// ProxyManager 代理管理器接口 - 统一的对外接口
type ProxyManager interface {
	// 核心功能
	GetHTTPClient() *http.Client
	GetProxyString() string
	// 解析给定 URL 实际将使用的代理（若有）。返回空字符串表示直连
	ResolveProxy(rawurl string) (string, error)

	// 配置管理
	UpdateConfig(config *Config) error
	GetConfig() *Config

	// 便捷方法
	SetManualProxy(proxyAddress string) error
	UpdateSystemProxy() error
	DisableProxy() error

	// 环境变量设置
	SetupEnv()
}

// ConfigChangeCallback 配置变更回调函数类型
type ConfigChangeCallback func(oldConfig, newConfig *Config)

package proxy

import (
    "CanMe/backend/pkg/events"
    "context"
    "net/http"
    "net/url"
    "sync"
)

// Manager 代理管理器 - 统一的对外接口
type Manager struct {
	ctx      context.Context
	client   *client // 注意：client 现在是私有的
	eventBus events.EventBus
	mu       sync.RWMutex
}

// NewManager 创建代理管理器
func NewManager(config *Config, eventBus events.EventBus) *Manager {
	return &Manager{
		client:   newClient(config), // 使用私有构造函数
		eventBus: eventBus,
	}
}

// SetContext 设置上下文
func (m *Manager) SetContext(ctx context.Context) {
	m.ctx = ctx
	m.client.setContext(ctx)
}

// GetHTTPClient 获取HTTP客户端
func (m *Manager) GetHTTPClient() *http.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client.httpClient()
}

// GetProxyString 获取代理字符串
func (m *Manager) GetProxyString() string {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.client.getProxyString()
}

// ResolveProxy 返回给定原始 URL 将要使用的代理地址（字符串形式）。
// 若返回空字符串，表示直连。
func (m *Manager) ResolveProxy(rawurl string) (string, error) {
    u, err := url.Parse(rawurl)
    if err != nil {
        return "", err
    }
    m.mu.RLock()
    defer m.mu.RUnlock()
    // 使用与 http.Transport 相同的 Proxy 解析逻辑
    f := m.client.proxyFunc()
    ru := &http.Request{URL: u}
    p, err := f(ru)
    if err != nil || p == nil {
        return "", err
    }
    return p.String(), nil
}

// GetConfig 获取当前配置
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client.getConfig()
}

// UpdateConfig 更新配置并发布事件
func (m *Manager) UpdateConfig(config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldConfig := m.client.getConfig()
	if err := m.client.updateConfig(config); err != nil {
		return err
	}

	// 统一的事件发布
	// 暂时没有监听事件，以后可能会有
	if m.eventBus != nil {
		event := NewProxyConfigChangedEvent("proxy.manager", oldConfig, config)
		m.eventBus.PublishAsync(m.ctx, event)
	}

	return nil
}

// SetManualProxy 设置手动代理
func (m *Manager) SetManualProxy(proxyAddress string) error {
	config := &Config{
		Type:         "manual",
		ProxyAddress: proxyAddress,
		Timeout:      m.client.getConfig().Timeout,
		NoProxy:      m.client.getConfig().NoProxy,
	}
	return m.UpdateConfig(config)
}

// UpdateSystemProxy 更新为系统代理
func (m *Manager) UpdateSystemProxy() error {
	config := &Config{
		Type:    "system",
		Timeout: m.client.getConfig().Timeout,
		NoProxy: m.client.getConfig().NoProxy,
	}
	return m.UpdateConfig(config)
}

// DisableProxy 禁用代理
func (m *Manager) DisableProxy() error {
	config := &Config{
		Type:    "none",
		Timeout: m.client.getConfig().Timeout,
		NoProxy: m.client.getConfig().NoProxy,
	}
	return m.UpdateConfig(config)
}

// SetupEnv 设置环境变量
func (m *Manager) SetupEnv() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.client.setupEnv()
}

// Do 执行HTTP请求
func (m *Manager) Do(req *http.Request) (*http.Response, error) {
	return m.GetHTTPClient().Do(req)
}

// Get 执行GET请求
func (m *Manager) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return m.Do(req)
}

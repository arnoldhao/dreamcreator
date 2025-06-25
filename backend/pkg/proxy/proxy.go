package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Config 代理配置
type Config struct {
	// 代理类型：none, system, manual
	Type string `json:"type"`

	// 代理地址，格式为 http://ip:port 或 https://ip:port 或 socks5://ip:port
	ProxyAddress string `json:"proxy_address,omitempty"`

	// 超时设置
	Timeout time.Duration `json:"timeout,omitempty"`

	// 绕过代理的主机列表
	NoProxy []string `json:"no_proxy,omitempty"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if c.Type == "manual" && c.ProxyAddress == "" {
		return fmt.Errorf("proxy address is required for manual proxy")
	}

	if c.Type == "manual" {
		if _, err := url.Parse(c.ProxyAddress); err != nil {
			return fmt.Errorf("invalid proxy address: %w", err)
		}
	}

	return nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Type:    "system",
		Timeout: 30 * time.Minute, // 默认30分钟
	}
}

// client 代理客户端 - 现在是私有的
type client struct {
	ctx    context.Context
	config *Config
	hc     *http.Client
	mu     sync.RWMutex
}

// newClient 创建新的代理客户端 - 私有构造函数
func newClient(config *Config) *client {
	if config == nil {
		config = DefaultConfig()
	}

	c := &client{
		config: config,
	}

	// 初始化HTTP客户端
	c.resetHTTPClient()

	return c
}

// setContext 设置上下文
func (c *client) setContext(ctx context.Context) {
	c.ctx = ctx
}

// resetHTTPClient 重置HTTP客户端
func (c *client) resetHTTPClient() {
	transport := &http.Transport{
		Proxy: c.proxyFunc(),
		DialContext: (&net.Dialer{
			Timeout:   c.config.Timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	c.hc = &http.Client{
		Transport: transport,
		Timeout:   c.config.Timeout,
	}
}

// proxyFunc 返回代理函数
func (c *client) proxyFunc() func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		switch c.config.Type {
		case "none":
			return nil, nil
		case "system":
			// 在Windows上优先使用注册表代理设置
			if runtime.GOOS == "windows" {
				if proxy, err := getSystemProxyFromRegistry(); err == nil && proxy != "" {
					// ignore bypass list, just proxy it
					return parseWindowsProxy(proxy)
				}
				// Windows fallback: 如果注册表读取失败，使用环境变量
				return http.ProxyFromEnvironment(req)
			} else if runtime.GOOS == "darwin" {
				return getDarwinProxy(req)
			} else {
				// 在其他平台上，使用环境变量代理设置
				return http.ProxyFromEnvironment(req)
			}
		case "manual":
			return c.manualProxyFunc(req)
		default:
			return http.ProxyFromEnvironment(req)
		}

		return nil, nil
	}
}

// manualProxyFunc 处理手动设置的代理
func (c *client) manualProxyFunc(req *http.Request) (*url.URL, error) {
	// 检查是否应该跳过代理
	if c.shouldBypassProxy(req.URL.Hostname()) {
		return nil, nil
	}

	// 使用ProxyAddress作为代理URL
	if c.config.ProxyAddress == "" {
		return nil, nil
	}

	proxyURL, err := url.Parse(c.config.ProxyAddress)
	if err != nil {
		return nil, err
	}

	return proxyURL, nil
}

// shouldBypassProxy 判断是否应该绕过代理
func (c *client) shouldBypassProxy(host string) bool {
	// localhost和127.0.0.1不使用代理
	if host == "localhost" || strings.HasPrefix(host, "127.") {
		return true
	}

	// 检查NoProxy列表
	for _, noProxyHost := range c.config.NoProxy {
		if strings.Contains(host, noProxyHost) {
			return true
		}
	}

	return false
}

// updateConfig 更新配置 - 内部方法
func (c *client) updateConfig(newConfig *Config) error {
	if err := newConfig.Validate(); err != nil {
		return err
	}

	c.mu.Lock()
	c.config = newConfig
	c.resetHTTPClient()
	c.mu.Unlock()

	return nil
}

// getConfig 返回当前配置的副本
func (c *client) getConfig() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建配置的副本，避免外部修改
	configCopy := *c.config
	if c.config.NoProxy != nil {
		configCopy.NoProxy = make([]string, len(c.config.NoProxy))
		copy(configCopy.NoProxy, c.config.NoProxy)
	}
	return &configCopy
}

// httpClient 返回HTTP客户端
func (c *client) httpClient() *http.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hc
	configType := c.config.Type
	hc := c.hc
	c.mu.RUnlock()

	// 只有系统代理模式才需要重置
	if configType == "system" {
		c.mu.Lock()
		c.resetHTTPClient()
		hc = c.hc
		c.mu.Unlock()
	}

	return hc
}

// setupEnv 设置环境变量
func (c *client) setupEnv() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 清除现有环境变量
	envVars := []string{"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy", "NO_PROXY", "no_proxy"}
	for _, env := range envVars {
		os.Unsetenv(env)
	}

	if c.config.Type == "none" {
		return
	}

	if c.config.Type == "manual" && c.config.ProxyAddress != "" {
		os.Setenv("HTTP_PROXY", c.config.ProxyAddress)
		os.Setenv("http_proxy", c.config.ProxyAddress)
		os.Setenv("HTTPS_PROXY", c.config.ProxyAddress)
		os.Setenv("https_proxy", c.config.ProxyAddress)

		if len(c.config.NoProxy) > 0 {
			noProxy := strings.Join(c.config.NoProxy, ",")
			os.Setenv("NO_PROXY", noProxy)
			os.Setenv("no_proxy", noProxy)
		}
	}
}

// getProxyString 获取代理字符串
func (c *client) getProxyString() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.config.Type == "none" {
		return ""
	}

	req := &http.Request{
		URL: &url.URL{
			Scheme: "http",
			Host:   "example.com",
		},
	}

	proxyURL, err := c.proxyFunc()(req)
	if err != nil || proxyURL == nil {
		return ""
	}

	if proxyURL.Port() != "" {
		return fmt.Sprintf("%s:%s", proxyURL.Hostname(), proxyURL.Port())
	}

	return proxyURL.Hostname()
}

// parseWindowsProxy 解析Windows代理字符串
func parseWindowsProxy(proxyStr string) (*url.URL, error) {
	// 处理格式: "http=127.0.0.1:8888;https=127.0.0.1:8888"
	if strings.Contains(proxyStr, "=") {
		parts := strings.Split(proxyStr, ";")
		for _, part := range parts {
			if strings.HasPrefix(part, "http=") {
				proxyStr = strings.TrimPrefix(part, "http=")
				break
			}
		}
	}

	// 确保有协议前缀
	if !strings.HasPrefix(proxyStr, "http://") && !strings.HasPrefix(proxyStr, "https://") {
		proxyStr = "http://" + proxyStr
	}

	return url.Parse(proxyStr)
}

// getDarwinProxy 获取macOS代理设置
func getDarwinProxy(req *http.Request) (*url.URL, error) {
	// 1: from env
	proxy, err := http.ProxyFromEnvironment(req)
	if err != nil || proxy == nil {
		// 2: from network
		return getDarwinCurrentProxy()
	} else {
		return proxy, nil
	}
}

// getDarwinCurrentNetworkService 获取当前网络服务
func getDarwinCurrentNetworkService() (string, error) {
	// 1. get default network interface
	routeOut, err := exec.Command("route", "get", "default").Output()
	if err != nil {
		return "", err
	}

	// 2. parse interface name from route output
	lines := strings.Split(string(routeOut), "\n")
	var interfaceName string
	for _, line := range lines {
		if strings.Contains(line, "interface:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				interfaceName = parts[1]
				break
			}
		}
	}

	if interfaceName == "" {
		return "", fmt.Errorf("cannot find default network interface")
	}

	// 3. get current network service
	servicesOut, err := exec.Command("networksetup", "-listallhardwareports").Output()
	if err != nil {
		return "", err
	}

	// 4. find current network service
	lines = strings.Split(string(servicesOut), "\n")
	var currentService string
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], "Device: "+interfaceName) {
			// 前一行是服务名
			if i > 0 && strings.Contains(lines[i-1], "Hardware Port:") {
				parts := strings.Split(lines[i-1], ":")
				if len(parts) >= 2 {
					currentService = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	if currentService == "" {
		return "", fmt.Errorf("cannot find current network service")
	}

	return currentService, nil
}

// getDarwinCurrentProxy 获取macOS当前代理设置
func getDarwinCurrentProxy() (*url.URL, error) {
	// 1. get current network service
	currentService, err := getDarwinCurrentNetworkService()
	if err != nil {
		return nil, err
	}

	// 2. get proxy settings
	proxyOut, err := exec.Command("networksetup", "-getwebproxy", currentService).Output()
	if err != nil {
		return nil, err
	}

	// 3. parse proxy settings
	lines := strings.Split(string(proxyOut), "\n")
	var proxyEnabled bool
	var proxyServer, proxyPort string

	for _, line := range lines {
		if strings.HasPrefix(line, "Enabled: ") {
			proxyEnabled = strings.Contains(line, "Yes")
		} else if strings.HasPrefix(line, "Server: ") {
			proxyServer = strings.TrimSpace(line[8:])
		} else if strings.HasPrefix(line, "Port: ") {
			proxyPort = strings.TrimSpace(line[6:])
		}
	}

	if !proxyEnabled || proxyServer == "" {
		return nil, nil // proxy is disabled or no server specified
	}

	// 4. construct proxy URL
	proxyStr := fmt.Sprintf("http://%s:%s", proxyServer, proxyPort)
	return url.Parse(proxyStr)
}

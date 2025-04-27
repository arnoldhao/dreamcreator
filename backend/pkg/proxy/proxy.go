package proxy

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
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
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Type:    "system",
		Timeout: 30 * time.Second,
	}
}

// Client 代理客户端
type Client struct {
	config     *Config
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewClient 创建新的代理客户端
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	client := &Client{
		config: config,
	}

	// 初始化HTTP客户端
	client.resetHTTPClient()

	return client
}

// resetHTTPClient 重置HTTP客户端
func (c *Client) resetHTTPClient() {
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

	c.httpClient = &http.Client{
		Transport: transport,
		Timeout:   c.config.Timeout,
	}
}

// proxyFunc 返回代理函数
func (c *Client) proxyFunc() func(*http.Request) (*url.URL, error) {
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
					return parseWindowsProxy(proxy)
				}
			}
			return http.ProxyFromEnvironment(req)
		case "manual":
			return c.manualProxyFunc(req)
		default:
			return http.ProxyFromEnvironment(req)
		}
	}
}

// manualProxyFunc 处理手动设置的代理
func (c *Client) manualProxyFunc(req *http.Request) (*url.URL, error) {
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
func (c *Client) shouldBypassProxy(host string) bool {
	// localhost和127.0.0.1不使用代理
	if host == "localhost" || strings.HasPrefix(host, "127.") {
		return true
	}

	return false
}

// SetConfig 设置新的配置
func (c *Client) SetConfig(config *Config) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config = config
	c.resetHTTPClient()
}

// GetConfig 返回当前配置的副本
func (c *Client) GetConfig() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建配置的副本，避免外部修改
	configCopy := *c.config
	return &configCopy
}

// UpdateSystemProxy 更新为系统代理
func (c *Client) UpdateSystemProxy() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config.Type = "system"
	c.resetHTTPClient()
}

// DisableProxy 禁用代理
func (c *Client) DisableProxy() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config.Type = "none"
	c.resetHTTPClient()
}

// SetManualProxy 设置手动代理
func (c *Client) SetManualProxy(proxyAddress string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config.Type = "manual"
	c.config.ProxyAddress = proxyAddress
	c.resetHTTPClient()
}

// HTTPClient 返回HTTP客户端
func (c *Client) HTTPClient() *http.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.httpClient
}

// Do 执行HTTP请求
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.HTTPClient().Do(req)
}

// Get 执行GET请求
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// SetupEnv 设置环境变量
func (c *Client) SetupEnv() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 清除现有环境变量
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("http_proxy")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("https_proxy")
	os.Unsetenv("NO_PROXY")
	os.Unsetenv("no_proxy")

	if c.config.Type == "none" {
		return
	}

	if c.config.Type == "manual" {
		if c.config.ProxyAddress != "" {
			os.Setenv("HTTP_PROXY", c.config.ProxyAddress)
			os.Setenv("http_proxy", c.config.ProxyAddress)
		}
	}
}

// GetProxyString 获取代理字符串（主机:端口格式）
func (c *Client) GetProxyString() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.config.Type == "manual" && c.config.ProxyAddress != "" {
		return c.config.ProxyAddress
	} else if c.config.Type == "system" {
		// 在Windows上尝试从注册表获取系统代理
		if runtime.GOOS == "windows" {
			if proxy, err := getSystemProxyFromRegistry(); err == nil && proxy != "" {
				return proxy
			}
		}

		// 从环境变量中获取系统代理
		httpProxy := os.Getenv("HTTP_PROXY")
		if httpProxy == "" {
			httpProxy = os.Getenv("http_proxy")
		}

		httpsProxy := os.Getenv("HTTPS_PROXY")
		if httpsProxy == "" {
			httpsProxy = os.Getenv("https_proxy")
		}

		// 优先返回 HTTPS 代理，因为它更安全
		if httpsProxy != "" {
			return httpsProxy
		}
		return httpProxy
	} else {
		return ""
	}
}

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

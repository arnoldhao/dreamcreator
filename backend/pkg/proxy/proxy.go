package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
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
		Type:    "none",
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

	// Configure SOCKS5 when needed
	// - Manual: if proxy address is socks5://host:port, use SOCKS dialer and disable HTTP Proxy
	// - macOS System: if no HTTP(S) proxy is set but system SOCKS is enabled, use SOCKS dialer
	if c.config.Type == "manual" && c.config.ProxyAddress != "" {
		if u, err := url.Parse(c.config.ProxyAddress); err == nil && strings.EqualFold(u.Scheme, "socks5") {
			if u.Host != "" {
				transport.Proxy = nil
				transport.DialContext = socks5DialContext(u.Host, c.config.Timeout)
			}
		}
	} else if c.config.Type == "system" && runtime.GOOS == "darwin" {
		if dpi, err := getDarwinEffectiveProxies(); err == nil && dpi != nil {
			httpAny := (dpi.HTTPEnabled && dpi.HTTPProxy != "" && dpi.HTTPPort != "") || (dpi.HTTPSEnabled && dpi.HTTPSProxy != "" && dpi.HTTPSPort != "")
			if !httpAny && dpi.SOCKSEnabled && dpi.SOCKSProxy != "" && dpi.SOCKSPort != "" {
				transport.Proxy = nil // avoid double-proxy via HTTP when using SOCKS at dial layer
				transport.DialContext = socks5DialContext(fmt.Sprintf("%s:%s", dpi.SOCKSProxy, dpi.SOCKSPort), c.config.Timeout)
			}
		}
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
	configType := c.config.Type
	hc := c.hc
	c.mu.RUnlock() // 先释放读锁

	// 只有系统代理模式才需要重置
	if configType == "system" {
		c.mu.Lock() // 现在安全地获取写锁
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

	// For macOS system SOCKS-only, proxyFunc returns nil; handle explicitly
	if c.config.Type == "system" && runtime.GOOS == "darwin" {
		if dpi, err := getDarwinEffectiveProxies(); err == nil && dpi != nil {
			httpAny := (dpi.HTTPEnabled && dpi.HTTPProxy != "" && dpi.HTTPPort != "") || (dpi.HTTPSEnabled && dpi.HTTPSProxy != "" && dpi.HTTPSPort != "")
			if !httpAny && dpi.SOCKSEnabled && dpi.SOCKSProxy != "" && dpi.SOCKSPort != "" {
				// Return scheme-qualified SOCKS5 for CLI env (ALL_PROXY)
				return fmt.Sprintf("socks5://%s:%s", dpi.SOCKSProxy, dpi.SOCKSPort)
			}
		}
	}

	proxyURL, err := c.proxyFunc()(req)
	if err != nil || proxyURL == nil {
		return ""
	}

	// Preserve socks5 scheme for CLI usage; strip scheme for http(s)
	scheme := strings.ToLower(proxyURL.Scheme)
	host := proxyURL.Hostname()
	port := proxyURL.Port()
	if scheme == "socks5" {
		if port != "" {
			return fmt.Sprintf("socks5://%s:%s", host, port)
		}
		return "socks5://" + host
	}
	if port != "" {
		return fmt.Sprintf("%s:%s", host, port)
	}
	return host
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
	// 1: environment variables (HTTP(S)_PROXY)
	if proxy, err := http.ProxyFromEnvironment(req); err == nil && proxy != nil {
		return proxy, nil
	}

	// 2: effective system proxies via scutil --proxy (handles VPN/PAC cases better)
	if dpi, err := getDarwinEffectiveProxies(); err == nil && dpi != nil {
		scheme := req.URL.Scheme
		// Map ws/wss to http/https
		if scheme == "ws" {
			scheme = "http"
		} else if scheme == "wss" {
			scheme = "https"
		}
		// Prefer scheme-specific HTTP proxy
		if scheme == "https" {
			if dpi.HTTPSEnabled && dpi.HTTPSProxy != "" && dpi.HTTPSPort != "" {
				return url.Parse(fmt.Sprintf("http://%s:%s", dpi.HTTPSProxy, dpi.HTTPSPort))
			}
			if dpi.HTTPEnabled && dpi.HTTPProxy != "" && dpi.HTTPPort != "" {
				return url.Parse(fmt.Sprintf("http://%s:%s", dpi.HTTPProxy, dpi.HTTPPort))
			}
		} else { // http and others
			if dpi.HTTPEnabled && dpi.HTTPProxy != "" && dpi.HTTPPort != "" {
				return url.Parse(fmt.Sprintf("http://%s:%s", dpi.HTTPProxy, dpi.HTTPPort))
			}
			if dpi.HTTPSEnabled && dpi.HTTPSProxy != "" && dpi.HTTPSPort != "" {
				return url.Parse(fmt.Sprintf("http://%s:%s", dpi.HTTPSProxy, dpi.HTTPSPort))
			}
		}
		// If only PAC/SOCKS is present, Proxy() should return nil here.
		// SOCKS is handled via DialContext in resetHTTPClient.
		return nil, nil
	}

	// 3: fallback to older networksetup-based detection (HTTP only)
	if p, err := getDarwinCurrentProxy(); err == nil {
		return p, nil
	}
	return nil, nil
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

// darwinProxyInfo holds effective proxy settings from `scutil --proxy`.
type darwinProxyInfo struct {
	HTTPEnabled  bool
	HTTPProxy    string
	HTTPPort     string
	HTTPSEnabled bool
	HTTPSProxy   string
	HTTPSPort    string
	SOCKSEnabled bool
	SOCKSProxy   string
	SOCKSPort    string
	PACEnabled   bool
	PACURL       string
}

// getDarwinEffectiveProxies parses `scutil --proxy` for effective system proxy settings.
func getDarwinEffectiveProxies() (*darwinProxyInfo, error) {
	out, err := exec.Command("scutil", "--proxy").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	dpi := &darwinProxyInfo{}
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "<") {
			continue
		}
		parts := strings.SplitN(ln, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "HTTPEnable":
			dpi.HTTPEnabled = val == "1"
		case "HTTPProxy":
			dpi.HTTPProxy = val
		case "HTTPPort":
			dpi.HTTPPort = val
		case "HTTPSEnable":
			dpi.HTTPSEnabled = val == "1"
		case "HTTPSProxy":
			dpi.HTTPSProxy = val
		case "HTTPSPort":
			dpi.HTTPSPort = val
		case "SOCKSEnable":
			dpi.SOCKSEnabled = val == "1"
		case "SOCKSProxy":
			dpi.SOCKSProxy = val
		case "SOCKSPort":
			dpi.SOCKSPort = val
		case "ProxyAutoConfigEnable":
			dpi.PACEnabled = val == "1"
		case "ProxyAutoConfigURLString":
			dpi.PACURL = val
		}
	}
	return dpi, nil
}

// socks5DialContext returns a DialContext that tunnels connections via a SOCKS5 proxy (no-auth).
func socks5DialContext(socksAddr string, baseTimeout time.Duration) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		var d net.Dialer
		if deadline, ok := ctx.Deadline(); ok {
			d.Timeout = time.Until(deadline)
		} else {
			d.Timeout = baseTimeout
		}
		// 1. connect to SOCKS server
		conn, err := d.DialContext(ctx, "tcp", socksAddr)
		if err != nil {
			return nil, err
		}

		// Ensure we close on any error
		defer func() {
			if err != nil {
				conn.Close()
			}
		}()

		// 2. greeting: version 5, 1 method, no-auth (0x00)
		if _, err = conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
			return nil, err
		}
		// read method selection: ver, method
		buf := make([]byte, 2)
		if _, err = ioReadFull(ctx, conn, buf); err != nil {
			return nil, err
		}
		if buf[0] != 0x05 || buf[1] != 0x00 {
			return nil, fmt.Errorf("socks5: unsupported method %d", buf[1])
		}

		// 3. connect command
		host, portStr, err2 := net.SplitHostPort(address)
		if err2 != nil {
			return nil, err2
		}
		portNum, err2 := parsePort(portStr)
		if err2 != nil {
			return nil, err2
		}

		// build request: VER=5, CMD=1, RSV=0, ATYP, DST.ADDR, DST.PORT
		req := []byte{0x05, 0x01, 0x00}
		ip := net.ParseIP(host)
		if ip4 := ip.To4(); ip4 != nil {
			req = append(req, 0x01)
			req = append(req, ip4...)
		} else if ip6 := ip.To16(); ip6 != nil {
			req = append(req, 0x04)
			req = append(req, ip6...)
		} else {
			// domain name
			if len(host) > 255 {
				return nil, fmt.Errorf("socks5: host name too long")
			}
			req = append(req, 0x03, byte(len(host)))
			req = append(req, []byte(host)...)
		}
		// append port big-endian
		req = append(req, byte(portNum>>8), byte(portNum))

		if _, err = conn.Write(req); err != nil {
			return nil, err
		}

		// 4. read response: VER, REP, RSV, ATYP, BND.ADDR, BND.PORT
		// first 4 bytes
		hdr := make([]byte, 4)
		if _, err = ioReadFull(ctx, conn, hdr); err != nil {
			return nil, err
		}
		if hdr[0] != 0x05 || hdr[1] != 0x00 {
			return nil, fmt.Errorf("socks5: connect failed, rep=%d", hdr[1])
		}
		// consume addr per ATYP
		var toRead int
		switch hdr[3] {
		case 0x01:
			toRead = 4
		case 0x04:
			toRead = 16
		case 0x03:
			// next byte is len
			lb := make([]byte, 1)
			if _, err = ioReadFull(ctx, conn, lb); err != nil {
				return nil, err
			}
			toRead = int(lb[0])
		default:
			return nil, fmt.Errorf("socks5: invalid atyp %d", hdr[3])
		}
		if toRead > 0 {
			dummy := make([]byte, toRead)
			if _, err = ioReadFull(ctx, conn, dummy); err != nil {
				return nil, err
			}
		}
		// read bound port (2 bytes)
		dummy := make([]byte, 2)
		if _, err = ioReadFull(ctx, conn, dummy); err != nil {
			return nil, err
		}

		// success; return tunneled connection
		return conn, nil
	}
}

// ioReadFull honors context cancellation while reading from conn.
func ioReadFull(ctx context.Context, conn net.Conn, buf []byte) (int, error) {
	type res struct {
		n   int
		err error
	}
	ch := make(chan res, 1)
	go func() {
		n, err := io.ReadFull(conn, buf)
		ch <- res{n, err}
	}()
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case r := <-ch:
		return r.n, r.err
	}
}

// parsePort converts string to uint16 with range check.
func parsePort(s string) (uint16, error) {
	var p int
	var err error
	if s == "" {
		return 0, fmt.Errorf("invalid port")
	}
	p, err = strconv.Atoi(s)
	if err != nil || p <= 0 || p > 65535 {
		return 0, fmt.Errorf("invalid port")
	}
	return uint16(p), nil
}

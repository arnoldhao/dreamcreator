package downinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Config 下载配置
type Config struct {
	// 下载目录路径
	Dir string `json:"dir"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Dir: GetDefaultDownloadDir(),
	}
}

// GetDefaultDownloadDir 根据操作系统获取默认下载目录
func GetDefaultDownloadDir() string {
	var dir string

	switch runtime.GOOS {
	case "windows":
		// Windows 系统
		dir = filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	case "darwin":
		// macOS 系统
		dir = filepath.Join(os.Getenv("HOME"), "Downloads")
	case "linux":
		// Linux 系统
		dir = filepath.Join(os.Getenv("HOME"), "Downloads")
	default:
		// 其他系统，使用当前目录下的 downloads 文件夹
		dir = "downloads"
	}

	// make sure dir exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}

	return dir
}

// Client 下载管理客户端
type Client struct {
	config *Config
	mu     sync.RWMutex
}

// NewClient 创建新的下载管理客户端
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		config: config,
	}
}

// SetConfig 设置新的配置
func (c *Client) SetConfig(config *Config) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config = config
}

// GetConfig 返回当前配置的副本
func (c *Client) GetConfig() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建配置的副本，避免外部修改
	configCopy := *c.config
	return &configCopy
}

// SetDir 设置下载目录
func (c *Client) SetDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config.Dir = dir
}

// GetDir 获取下载目录
func (c *Client) GetDir() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.config.Dir
}

// GetDownloadDirWithDreamcreator 获取带有 dreamcreator 子目录的下载路径
func (c *Client) GetDownloadDirWithDreamcreator() string {
	return filepath.Join(c.GetDir(), "dreamcreator")
}

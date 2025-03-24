package logger

import (
	"os"
	"path/filepath"
)

// Config 日志配置
type Config struct {
	// 日志级别: debug, info, warn, error
	Level string `json:"level"`
	// 日志输出目录
	Directory string `json:"directory"`
	// 是否输出到控制台
	EnableConsole bool `json:"enable_console"`
	// 是否输出到文件
	EnableFile bool `json:"enable_file"`
	// 单个日志文件最大大小（MB）
	MaxSize int `json:"max_size"`
	// 保留天数
	MaxAge int `json:"max_age"`
	// 保留文件个数
	MaxBackups int `json:"max_backups"`
	// 是否压缩
	Compress bool `json:"compress"`
}

// DefaultConfig 返回默认配置，使用绝对路径
func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return &Config{
		Level:         "info",
		Directory:     filepath.Join(homeDir, ".canme/logs"), // 使用绝对路径
		EnableConsole: true,
		EnableFile:    true,
		MaxSize:       10,   // 10MB
		MaxAge:        7,    // 7天
		MaxBackups:    5,    // 保留5个备份
		Compress:      true, // 压缩旧日志
	}
}

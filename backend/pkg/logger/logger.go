package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// 全局logger实例
	globalLogger *zap.Logger
)

// InitLogger 初始化日志
func InitLogger(cfg *Config) error {
	// 确保日志目录存在
	if cfg.EnableFile {
		if err := os.MkdirAll(cfg.Directory, 0755); err != nil {
			return fmt.Errorf("create log directory failed: %w", err)
		}
	}

	// 配置zapcore
	var cores []zapcore.Core

	// 解析日志级别
	level := zap.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return fmt.Errorf("parse log level failed: %w", err)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 添加控制台输出
	if cfg.EnableConsole {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// 添加文件输出
	if cfg.EnableFile {
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
		logFile := filepath.Join(cfg.Directory, "canme.log")
		writer := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    cfg.MaxSize,    // MB
			MaxAge:     cfg.MaxAge,     // days
			MaxBackups: cfg.MaxBackups, // files
			Compress:   cfg.Compress,
		}
		fileCore := zapcore.NewCore(
			fileEncoder,
			zapcore.AddSync(writer),
			level,
		)
		cores = append(cores, fileCore)
	}

	// 创建logger
	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller())

	// 设置全局logger
	globalLogger = logger
	return nil
}

// GetLogger 获取全局logger实例
func GetLogger() *zap.Logger {
	if globalLogger == nil {
		// 如果全局logger未初始化，使用默认配置初始化
		if err := InitLogger(DefaultConfig()); err != nil {
			// 如果初始化失败，使用一个基本的console logger
			globalLogger = zap.NewExample()
		}
	}
	return globalLogger
}

// Debug level log
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Info level log
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Warn level log
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error level log
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

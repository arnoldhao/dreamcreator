package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync" // 用于在重新初始化期间保护对全局变量的并发访问

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	globalLogger      *zap.Logger
	globalAtomicLevel zap.AtomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel) // 默认Info级别
	currentConfig     *Config                                               // 保存当前配置用于比较
	configMutex       sync.RWMutex
	initOnce          sync.Once // 懒初始化默认配置
)

// InitLogger 初始化或重新初始化日志记录器
func InitLogger(cfg *Config) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	var newZapCoreLevel zapcore.Level
	if err := newZapCoreLevel.UnmarshalText([]byte(cfg.Level)); err != nil {
		return fmt.Errorf("解析日志级别失败: %w", err)
	}

	// 始终更新原子级别，这样即使核心未重建，级别也会立即生效
	globalAtomicLevel.SetLevel(newZapCoreLevel)

	// 检查是否需要重建核心（例如，输出目标、文件设置等发生变化）
	needsCoreRebuild := false
	if globalLogger == nil || currentConfig == nil ||
		currentConfig.EnableConsole != cfg.EnableConsole ||
		currentConfig.EnableFile != cfg.EnableFile ||
		(cfg.EnableFile && (currentConfig.Directory != cfg.Directory ||
			currentConfig.MaxSize != cfg.MaxSize ||
			currentConfig.MaxAge != cfg.MaxAge ||
			currentConfig.MaxBackups != cfg.MaxBackups ||
			currentConfig.Compress != cfg.Compress)) {
		needsCoreRebuild = true
	}

	if !needsCoreRebuild {
		// 如果核心不需要重建（例如，只有Level字段在Config中变化，但Level已通过AtomicLevel更新）
		// 我们仍然需要更新currentConfig以反映传入的cfg
		// 保存副本，避免外部指针被修改造成别名问题
		cfgCopy := *cfg
		currentConfig = &cfgCopy
		if globalLogger != nil {
			// 使用已存在的logger（其级别已通过AtomicLevel更新）记录级别变更事件
			globalLogger.Info("Logger level updated via AtomicLevel", zap.String("newLevel", cfg.Level))
		}
		return nil
	}

	// --- 需要重建核心 ---
	if cfg.EnableFile {
		if err := os.MkdirAll(cfg.Directory, 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level", // 将反映AtomicLevel的当前设置
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder, // 使用ISO8601格式
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var cores []zapcore.Core

	if cfg.EnableConsole {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), globalAtomicLevel)
		cores = append(cores, consoleCore)
	}

	if cfg.EnableFile {
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)        // 文件通常使用JSON格式
		logFile := filepath.Join(cfg.Directory, "dreamcreator.log") // 日志文件名
		writer := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge,
			MaxBackups: cfg.MaxBackups,
			Compress:   cfg.Compress,
		}
		fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(writer), globalAtomicLevel)
		cores = append(cores, fileCore)
	}

	var newLogger *zap.Logger
	var oldLogger *zap.Logger = globalLogger
	if len(cores) == 0 {
		// 显式使用 Nop logger，确保禁用所有输出时生效
		core := zapcore.NewNopCore()
		newLogger = zap.New(core)
	} else {
		combinedCore := zapcore.NewTee(cores...)
		newLogger = zap.New(
			combinedCore,
			zap.AddCaller(),
			zap.AddCallerSkip(1), // 适配 package 级封装函数，指向实际调用方
			zap.ErrorOutput(zapcore.AddSync(os.Stderr)),
		) // 添加ErrorOutput以捕获zap内部错误
	}

	globalLogger = newLogger
	// 保存新的配置状态（值拷贝）
	cfgCopy := *cfg
	currentConfig = &cfgCopy // 保存新的配置状态

	// 使用新的（或刚更新级别的）logger记录配置已更新
	globalLogger.Info("Logger (re)configured", zap.Any("newConfig", cfg))
	// 尝试刷新旧 logger 输出
	if oldLogger != nil {
		_ = oldLogger.Sync()
	}
	return nil
}

// IsDebugEnabled 返回当前日志级别是否为 Debug（或更详细，如 Debug/Trace）。
func IsDebugEnabled() bool {
	// globalAtomicLevel.Level() 是原子读取，线程安全。
	return globalAtomicLevel.Level() <= zap.DebugLevel
}

// GetLogger 获取全局logger实例
func GetLogger() *zap.Logger {
	configMutex.RLock()
	if globalLogger != nil {
		defer configMutex.RUnlock()
		return globalLogger
	}
	configMutex.RUnlock()

	initOnce.Do(func() {
		if globalLogger != nil {
			return
		}
		if err := InitLogger(DefaultConfig()); err != nil {
			configMutex.Lock()
			defer configMutex.Unlock()
			if globalLogger == nil {
				globalLogger = zap.NewExample()
			}
		}
	})

	configMutex.RLock()
	defer configMutex.RUnlock()
	if globalLogger == nil {
		return zap.NewExample()
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

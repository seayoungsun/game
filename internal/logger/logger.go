package logger

import (
	"os"
	"path/filepath"

	"github.com/kaifa/game-platform/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger *zap.Logger
	Sugar  *zap.SugaredLogger
)

// InitLogger 初始化日志系统
func InitLogger(cfg config.LogConfig) error {
	// 确保日志目录存在
	if err := os.MkdirAll(cfg.OutputPath, 0755); err != nil {
		return err
	}

	// 设置日志级别
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 文件日志输出
	fileWriter := &lumberjack.Logger{
		Filename:   filepath.Join(cfg.OutputPath, "app.log"),
		MaxSize:    cfg.MaxSize,    // MB
		MaxBackups: cfg.MaxBackups, // 保留文件数
		MaxAge:     cfg.MaxAge,     // 天
		Compress:   true,           // 压缩旧文件
	}

	// 控制台日志输出
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// 多个输出源
	core := zapcore.NewTee(
		// 控制台输出（开发环境）
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
		// 文件输出
		zapcore.NewCore(fileEncoder, zapcore.AddSync(fileWriter), level),
	)

	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	Sugar = Logger.Sugar()

	return nil
}

// Sync 同步日志缓冲区
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

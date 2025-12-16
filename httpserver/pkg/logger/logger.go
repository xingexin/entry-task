package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Logger 全局日志实例
	Logger *zap.Logger
	Sugar  *zap.SugaredLogger
)

// Config 日志配置
type Config struct {
	Level    string // debug, info, warn, error
	Output   string // stdout, file
	FilePath string // 文件路径
}

// Init 初始化日志
func Init(cfg *Config) error {
	// 1. 设置日志级别
	level := zapcore.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	// 2. 设置输出位置
	var writeSyncer zapcore.WriteSyncer
	if cfg.Output == "file" {
		// 输出到文件
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		writeSyncer = zapcore.AddSync(file)
	} else {
		// 输出到控制台
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// 3. 自定义编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,

		// 自定义日志级别格式 [INFO]
		EncodeLevel: customLevelEncoder,

		// 自定义时间格式
		EncodeTime: zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),

		// 持续时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,

		// 调用者格式
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// 4. 创建 core
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig), // 使用控制台编码器（带颜色）
		writeSyncer,
		level,
	)

	// 5. 创建 logger
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Sugar = Logger.Sugar()

	return nil
}

// customLevelEncoder 自定义日志级别编码器（带颜色）
func customLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	// ANSI 颜色代码
	const (
		colorReset  = "\033[0m"
		colorRed    = "\033[31m"
		colorGreen  = "\033[32m"
		colorYellow = "\033[33m"
		colorBlue   = "\033[34m"
	)

	var coloredLevel string
	switch level {
	case zapcore.DebugLevel:
		coloredLevel = colorBlue + "[DEBUG]" + colorReset
	case zapcore.InfoLevel:
		coloredLevel = colorGreen + "[INFO] " + colorReset
	case zapcore.WarnLevel:
		coloredLevel = colorYellow + "[WARN] " + colorReset
	case zapcore.ErrorLevel:
		coloredLevel = colorRed + "[ERROR]" + colorReset
	case zapcore.DPanicLevel, zapcore.PanicLevel:
		coloredLevel = colorRed + "[PANIC]" + colorReset
	case zapcore.FatalLevel:
		coloredLevel = colorRed + "[FATAL]" + colorReset
	default:
		coloredLevel = "[UNKNOWN]"
	}

	enc.AppendString(coloredLevel)
}

// Info 记录 Info 级别日志
func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

// Warn 记录 Warn 级别日志
func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

// Error 记录 Error 级别日志
func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

// Debug 记录 Debug 级别日志
func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

// Fatal 记录 Fatal 级别日志（会退出程序）
func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}

// Sync 同步日志（程序退出前调用）
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

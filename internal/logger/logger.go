package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger *zap.Logger

// InitLogger 初始化日志
func InitLogger(level string, output string, filePath string, maxSize int, maxBackups int, maxAge int) error {
	// 解析日志级别
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,    // 大写级别名
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // ISO8601时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder, // 秒级持续时间
		EncodeCaller:   zapcore.ShortCallerEncoder,     // 短路径编码
	}

	// 输出配置
	var writeSyncer zapcore.WriteSyncer
	if output == "file" {
		// 文件输出（支持日志轮转）
		lumberJackLogger := &lumberjack.Logger{
			Filename:   filePath,
			MaxSize:    maxSize,    // MB
			MaxBackups: maxBackups, // 保留旧文件数量
			MaxAge:     maxAge,     // 保留天数
			Compress:   true,       // 压缩
		}
		writeSyncer = zapcore.AddSync(lumberJackLogger)
	} else {
		// 标准输出
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// 创建核心
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig), // JSON格式
		writeSyncer,
		zapLevel,
	)

	// 创建Logger
	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return nil
}

// Info 记录信息日志
func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

// Debug 记录调试日志
func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

// Warn 记录警告日志
func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

// Error 记录错误日志
func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

// Fatal 记录致命错误日志并退出
func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}

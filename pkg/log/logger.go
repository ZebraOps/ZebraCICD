package log

import (
	"os"

	"github.com/ZebraOps/ZebraCICD/internal/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.Logger
var sugar *zap.SugaredLogger

// InitWithConfig initializes the logger with custom configuration
func InitWithConfig(cfg types.LoggingConfig) error {
	// 创建 lumberjack 配置用于日志轮转
	var writers []zapcore.WriteSyncer

	for _, path := range cfg.OutputPaths {
		if path == "stdout" {
			writers = append(writers, zapcore.AddSync(os.Stdout))
		} else if path == "stderr" {
			writers = append(writers, zapcore.AddSync(os.Stderr))
		} else {
			// 文件输出，启用日志轮转
			lumberjackLogger := &lumberjack.Logger{
				Filename:   path,
				MaxSize:    cfg.MaxSize,
				MaxAge:     cfg.MaxAge,
				MaxBackups: cfg.MaxBackups,
				Compress:   cfg.Compress,
			}
			writers = append(writers, zapcore.AddSync(lumberjackLogger))
		}
	}

	// 配置日志级别
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// 创建 encoder 配置
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg.Encoding == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 创建 core
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		level,
	)

	// 创建 logger
	logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	sugar = logger.Sugar()

	// 替换全局 logger
	zap.ReplaceGlobals(logger)

	return nil
}

// Sync flushes any buffered logs.
func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}

// S returns the sugared logger for simple formatting.
func S() *zap.SugaredLogger {
	return sugar
}

func L() *zap.Logger {
	return logger
}

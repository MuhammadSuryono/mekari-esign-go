package logger

import (
	"mekari-esign/internal/config"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(cfg *config.Config) (*zap.Logger, error) {
	var zapConfig zap.Config

	if cfg.IsDevelopment() {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	switch cfg.Logging.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	encoder := zapcore.NewJSONEncoder(zapConfig.EncoderConfig)
	if cfg.IsDevelopment() || cfg.Logging.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(zapConfig.EncoderConfig)
	}

	level := zapConfig.Level

	consoleCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)

	_ = os.MkdirAll("logs", 0755)
	f, err := os.OpenFile(filepath.Join("logs", "app.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	fileCore := zapcore.NewCore(encoder, zapcore.AddSync(f), level)

	core := zapcore.NewTee(consoleCore, fileCore)
	logger := zap.New(core)
	return logger, nil
}

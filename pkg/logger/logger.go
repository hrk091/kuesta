package logger

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type _keyLogger struct{}

var (
	config     zap.Config
	rootLogger *zap.Logger
)

func init() {
	config = zap.NewProductionConfig()
	rootLogger, _ = config.Build()
}

func Setup(isDevel bool, lvl uint8, opts ...zap.Option) {
	if isDevel {
		config = zap.NewDevelopmentConfig()
	}
	config.Level = zap.NewAtomicLevelAt(ConvertLevel(lvl))
	rootLogger, _ = config.Build(opts...)
}

func ConvertLevel(lvl uint8) zapcore.Level {
	if lvl < 3 {
		return zapcore.Level(1 - lvl)
	} else {
		return zapcore.DebugLevel
	}
}

func NewLogger() *zap.SugaredLogger {
	return rootLogger.Sugar()
}

func WithLogger(parent context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(parent, _keyLogger{}, logger)
}

func FromContext(ctx context.Context) *zap.SugaredLogger {
	v, ok := ctx.Value(_keyLogger{}).(*zap.SugaredLogger)
	if !ok {
		return NewLogger()
	} else {
		return v
	}
}

/*
 Copyright 2022 NTT Communications Corporation.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package logger

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type _keyLogger struct{}

var (
	config     zap.Config
	rootLogger *zap.Logger
)

func init() {
	SetDefault()
}

func SetDefault() {
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

func Error(ctx context.Context, err error, msg string, kvs ...interface{}) {
	l := FromContext(ctx).WithOptions(zap.AddCallerSkip(1))
	if st := GetStackTrace(err); st != "" {
		l = l.With("stacktrace", st)
	}
	l.Errorw(fmt.Sprintf("%s: %v", msg, err), kvs...)
}

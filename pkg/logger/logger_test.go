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

package logger_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"testing"

	"github.com/hrk091/nwctl/pkg/logger"
)

func TestConvertLevel(t *testing.T) {
	tests := []struct {
		given uint8
		want  zapcore.Level
	}{
		{0, zapcore.WarnLevel},
		{1, zapcore.InfoLevel},
		{2, zapcore.DebugLevel},
		{3, zapcore.DebugLevel},
	}

	for _, tt := range tests {
		assert.Equal(t, logger.ConvertLevel(tt.given), tt.want)
	}
}

func TestFromContext(t *testing.T) {
	want := logger.NewLogger()
	ctx := logger.WithLogger(context.Background(), want)
	assert.Equal(t, want, logger.FromContext(ctx))
}

func TestSetup(t *testing.T) {
	core := logger.NewLogger().Desugar().Core()
	assert.Equal(t, false, core.Enabled(zapcore.DebugLevel))

	logger.Setup(true, 2)
	core = logger.NewLogger().Desugar().Core()
	assert.Equal(t, true, core.Enabled(zapcore.DebugLevel))
	logger.SetDefault()
}

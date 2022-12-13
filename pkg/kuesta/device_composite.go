/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package kuesta

import (
	"context"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"fmt"
	"github.com/nttcom/kuesta/pkg/common"
	kcue "github.com/nttcom/kuesta/pkg/cue"
	"github.com/nttcom/kuesta/pkg/logger"
)

type DeviceCompositeCfg struct {
	RootCfg

	Device string `validate:"required"`
}

// Validate validates exposed fields according to the `validate` tag.
func (c *DeviceCompositeCfg) Validate() error {
	return common.Validate(c)
}

// RunDeviceComposite runs the main process of the `device composite` command.
func RunDeviceComposite(ctx context.Context, cfg *DeviceCompositeCfg) error {
	l := logger.FromContext(ctx)
	l.Debug("device composite called")

	cctx := cuecontext.New()
	sp := ServicePath{RootDir: cfg.ConfigRootPath}
	dp := DevicePath{RootDir: cfg.ConfigRootPath, Device: cfg.Device}

	files, err := CollectPartialDeviceConfig(sp.ServiceDirPath(IncludeRoot), cfg.Device)
	if err != nil {
		return fmt.Errorf("collect files: %w", err)
	}
	l.Debug("files: ", files)

	// composite all partial device configs into one CUE instance
	deviceConfig, err := kcue.NewValueWithInstance(cctx, files, nil)
	if err != nil {
		return fmt.Errorf("composite files: %w", err)
	}
	l.Debug("merged device config cue instance: ", deviceConfig)

	buf, err := kcue.FormatCue(deviceConfig, cue.Concrete(true))
	if err != nil {
		return fmt.Errorf("format merged config: %w", err)
	}

	if err := dp.WriteDeviceConfigFile(buf); err != nil {
		return fmt.Errorf("write merged config: %w", err)
	}

	return nil
}

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

package nwctl

import (
	"context"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"fmt"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/logger"
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
	sp := ServicePath{RootDir: cfg.RootPath}
	dp := DevicePath{RootDir: cfg.RootPath, Device: cfg.Device}

	files, err := CollectPartialDeviceConfig(sp.ServiceDirPath(IncludeRoot), cfg.Device)
	if err != nil {
		return fmt.Errorf("collect files: %w", err)
	}
	l.Debug("files: ", files)

	// composite all partial device configs into one CUE instance
	deviceConfig, err := NewValueWithInstance(cctx, files, nil)
	if err != nil {
		return fmt.Errorf("composite files: %w", err)
	}
	l.Debug("merged device config cue instance: ", deviceConfig)

	buf, err := FormatCue(deviceConfig, cue.Concrete(true))
	if err != nil {
		return fmt.Errorf("format merged config: %w", err)
	}

	if err := dp.WriteDeviceConfigFile(buf); err != nil {
		return fmt.Errorf("write merged config: %w", err)
	}

	return nil
}

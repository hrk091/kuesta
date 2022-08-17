package nwctl

import (
	"context"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"fmt"
	"github.com/hrk091/nwctl/pkg/logger"
)

type DeviceCompositeCfg struct {
	RootCfg

	Device string `validate:"required"`
}

// Validate validates exposed fields according to the `validate` tag.
func (c *DeviceCompositeCfg) Validate() error {
	return validate(c)
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

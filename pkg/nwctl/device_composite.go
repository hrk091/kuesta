package nwctl

import (
	"context"
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
	return nil
}

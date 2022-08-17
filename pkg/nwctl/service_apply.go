package nwctl

import (
	"context"
	"github.com/hrk091/nwctl/pkg/logger"
)

type ServiceApplyCfg struct {
	RootCfg
}

// RunServiceApply runs the main process of the `service apply` command.
func RunServiceApply(ctx context.Context, cfg *ServiceApplyCfg) error {
	l := logger.FromContext(ctx)
	l.Debug("service apply called")
	return nil
}

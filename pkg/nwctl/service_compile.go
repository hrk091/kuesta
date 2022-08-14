package nwctl

import (
	"context"
	"github.com/hrk091/nwctl/pkg/logger"
)

type ServiceCompileCfg struct {
	RootCfg

	Service string
	Keys    []string
}

func RunServiceCompile(ctx context.Context, config ServiceCompileCfg) error {
	l := logger.FromContext(ctx)
	l.Info("service compile called")

	return nil
}

package nwctl

import (
	"context"
	"github.com/hrk091/nwctl/pkg/logger"
)

type ServiceCompileCfg struct {
	RootCfg

	Service string   `validate:"required"`
	Keys    []string `validate:"gt=0,dive,required"`
}

// Validate validates exposed fields according to the `validate` tag.
func (c *ServiceCompileCfg) Validate() error {
	return validate(c)
}

func RunServiceCompile(ctx context.Context, cfg *ServiceCompileCfg) error {
	l := logger.FromContext(ctx)
	l.Info("service compile called")

	return nil
}

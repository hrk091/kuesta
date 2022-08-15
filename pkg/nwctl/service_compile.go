package nwctl

import (
	"context"
	"github.com/hrk091/nwctl/pkg/logger"
	"io"
)

type ServiceCompileCfg struct {
	RootCfg

	Service string   `validate:"required"`
	Keys    []string `validate:"gt=0,dive,required"`
}

func (c *ServiceCompileCfg) Validate() error {
	return validate(c)
}

type ServiceCompileCfgBuilder struct {
	cfg *ServiceCompileCfg

	Err error
}

func RunServiceCompile(ctx context.Context, buf io.Writer, config *ServiceCompileCfg) error {
	l := logger.FromContext(ctx)
	l.Info("service compile called")

	return nil
}

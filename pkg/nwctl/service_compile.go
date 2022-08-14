package nwctl

import (
	"context"
	"github.com/hrk091/nwctl/pkg/logger"
	"go.uber.org/multierr"
)

type ServiceCompileCfg struct {
	RootCfg

	Service string
	Keys    []string
}

type ServiceCompileCfgBuilder struct {
	cfg *ServiceCompileCfg

	Err error
}

// NewServiceCompileCfg creates ServiceCompileCfg builder.
func NewServiceCompileCfg() *ServiceCompileCfgBuilder {
	return &ServiceCompileCfgBuilder{
		Err: nil,
		cfg: &ServiceCompileCfg{},
	}
}

// Build creates new ServiceCompileCfg.
func (b *ServiceCompileCfgBuilder) Build() (*ServiceCompileCfg, error) {
	if b.Err != nil {
		return nil, b.Err
	}
	out := *b.cfg
	copy(out.Keys, b.cfg.Keys)
	return &out, nil
}

// AddErr appends error in a uber-go/multierr manner.
func (b *ServiceCompileCfgBuilder) AddErr(err error) {
	b.Err = multierr.Append(b.Err, err)
}

// Service sets service parameter to ServiceCompileCfg.
func (b *ServiceCompileCfgBuilder) Service(v string) *ServiceCompileCfgBuilder {
	if v == "" {
		b.AddErr(&ErrConfigValue{"service must be specified"})
	}
	b.cfg.Service = v
	return b
}

// Keys sets keys parameter to ServiceCompileCfg.
func (b *ServiceCompileCfgBuilder) Keys(v []string) *ServiceCompileCfgBuilder {
	if len(v) == 0 {
		b.AddErr(&ErrConfigValue{"target keys must be specified"})
	}
	b.cfg.Keys = v
	return b
}

func RunServiceCompile(ctx context.Context, config ServiceCompileCfg) error {
	l := logger.FromContext(ctx)
	l.Info("service compile called")

	return nil
}

package nwctl

import (
	"context"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"fmt"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/logger"
)

type ServiceCompileCfg struct {
	RootCfg

	Service string   `validate:"required"`
	Keys    []string `validate:"gt=0,dive,required"`
}

// Validate validates exposed fields according to the `validate` tag.
func (c *ServiceCompileCfg) Validate() error {
	return common.Validate(c)
}

// RunServiceCompile runs the main process of the `service compile` command.
func RunServiceCompile(ctx context.Context, cfg *ServiceCompileCfg) error {
	l := logger.FromContext(ctx)
	l.Debug("service compile called")

	cctx := cuecontext.New()

	sp := ServicePath{
		RootDir: cfg.RootPath,
		Service: cfg.Service,
		Keys:    cfg.Keys,
	}
	if err := sp.Validate(); err != nil {
		return fmt.Errorf("validate ServicePath: %w", err)
	}

	buf, err := sp.ReadServiceInput()
	if err != nil {
		return fmt.Errorf("read input file: %w", err)
	}
	inputVal, err := NewValueFromBuf(cctx, buf)
	if err != nil {
		return fmt.Errorf("load input file: %w", err)
	}

	transformVal, err := NewValueWithInstance(cctx, []string{sp.ServiceTransformPath(ExcludeRoot)}, &load.Config{Dir: sp.RootPath()})
	if err != nil {
		return fmt.Errorf("load transform file: %w", err)
	}

	it, err := ApplyTransform(cctx, inputVal, transformVal)
	if err != nil {
		return fmt.Errorf("apply transform: %w", err)
	}

	for it.Next() {
		device := it.Label()
		buf, err := ExtractDeviceConfig(it.Value())
		if err != nil {
			return fmt.Errorf("extract device config: %w", err)
		}

		if err := sp.WriteServiceComputedFile(device, buf); err != nil {
			return fmt.Errorf("save partial device config: %w", err)
		}
	}

	return nil
}

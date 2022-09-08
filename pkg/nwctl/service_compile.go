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
	"cuelang.org/go/cue/cuecontext"
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
		RootDir: cfg.ConfigRootPath,
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
	inputVal, err := NewValueFromBytes(cctx, buf)
	if err != nil {
		return fmt.Errorf("load input file: %w", err)
	}

	transformer, err := sp.ReadServiceTransform(cctx)
	if err != nil {
		return fmt.Errorf("load transform file: %w", err)
	}
	it, err := transformer.Apply(inputVal)
	if err != nil {
		return fmt.Errorf("apply transform: %w", err)
	}

	for it.Next() {
		device := it.Label()
		buf, err := NewDevice(it.Value()).Config()
		if err != nil {
			return fmt.Errorf("extract device config: %w", err)
		}

		if err := sp.WriteServiceComputedFile(device, buf); err != nil {
			return fmt.Errorf("save partial device config: %w", err)
		}
	}

	return nil
}

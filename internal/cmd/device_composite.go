/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package cmd

import (
	"fmt"

	"github.com/nttcom/kuesta/internal/core"
	"github.com/nttcom/kuesta/internal/logger"
	"github.com/spf13/cobra"
)

func newDeviceCompositeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "composite <device>",
		Short: "Composite service config compilation results",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newDeviceCompositeCfg(cmd, args)
			if err != nil {
				return err
			}
			logger.Setup(cfg.Devel, cfg.Verbose)

			return core.RunDeviceComposite(cmd.Context(), cfg)
		},
	}
	return cmd
}

func newDeviceCompositeCfg(cmd *cobra.Command, args []string) (*core.DeviceCompositeCfg, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("device is not specified")
	}
	rootCfg, err := newRootCfg(cmd)
	if err != nil {
		return nil, err
	}
	cfg := &core.DeviceCompositeCfg{
		RootCfg: *rootCfg,
		Device:  args[0],
	}
	return cfg, cfg.Validate()
}

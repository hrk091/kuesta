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

	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/nttcom/kuesta/pkg/logger"
	"github.com/spf13/cobra"
)

func newServiceCompileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compile <service> <key>...",
		Short: "Compile service config to partial device config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newServiceCompileCfg(cmd, args)
			if err != nil {
				return err
			}
			logger.Setup(cfg.Devel, cfg.Verbose)

			return kuesta.RunServiceCompile(cmd.Context(), cfg)
		},
	}
	return cmd
}

func newServiceCompileCfg(cmd *cobra.Command, args []string) (*kuesta.ServiceCompileCfg, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("service is not specified")
	}
	rootCfg, err := newRootCfg(cmd)
	if err != nil {
		return nil, err
	}
	cfg := &kuesta.ServiceCompileCfg{
		RootCfg: *rootCfg,
		Service: args[0],
		Keys:    args[1:],
	}
	return cfg, cfg.Validate()
}

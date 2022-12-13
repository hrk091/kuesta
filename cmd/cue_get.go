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

func newCueGetCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get cue type from the given go file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newCueGetCfg(cmd, args)
			if err != nil {
				return err
			}
			logger.Setup(cfg.Devel, cfg.Verbose)

			return kuesta.RunCueGet(cmd.Context(), cfg)
		},
	}
	cmd.PersistentFlags().StringP(FlagAggregateAddr, "", ":8000", "Bind address of device aggregator")
	mustBindToViper(cmd)

	return cmd
}

func newCueGetCfg(cmd *cobra.Command, args []string) (*kuesta.CueGetCfg, error) {
	rootCfg, err := newRootCfg(cmd)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("target file is not specified")
	}
	cfg := &kuesta.CueGetCfg{
		RootCfg:  *rootCfg,
		FilePath: args[0],
	}
	return cfg, cfg.Validate()
}
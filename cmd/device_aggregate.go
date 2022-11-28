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
	"github.com/hrk091/nwctl/pkg/logger"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	FlagAggregateAddr = "aggregate-addr"
)

func newDeviceAggregateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aggregate",
		Short: "Aggregate device config update and push to git",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newDeviceAggregateCfg(cmd, args)
			if err != nil {
				return err
			}
			logger.Setup(cfg.Devel, cfg.Verbose)

			return nwctl.RunDeviceAggregate(cmd.Context(), cfg)
		},
	}
	cmd.PersistentFlags().StringP(FlagAggregateAddr, "", ":8000", "Bind address of device aggregator")
	mustBindToViper(cmd)

	return cmd
}

func newDeviceAggregateCfg(cmd *cobra.Command, args []string) (*nwctl.DeviceAggregateCfg, error) {
	rootCfg, err := newRootCfg(cmd)
	if err != nil {
		return nil, err
	}
	cfg := &nwctl.DeviceAggregateCfg{
		RootCfg: *rootCfg,
		Addr:    viper.GetString(FlagAggregateAddr),
	}
	return cfg, cfg.Validate()
}

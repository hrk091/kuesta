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

package cmd

import (
	"fmt"
	"github.com/hrk091/nwctl/pkg/logger"
	"github.com/hrk091/nwctl/pkg/nwctl"
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

			return nwctl.RunDeviceComposite(cmd.Context(), cfg)
		},
	}
	return cmd
}

func newDeviceCompositeCfg(cmd *cobra.Command, args []string) (*nwctl.DeviceCompositeCfg, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("device is not specified")
	}
	rootCfg, err := newRootCfg(cmd)
	if err != nil {
		return nil, err
	}
	cfg := &nwctl.DeviceCompositeCfg{
		RootCfg: *rootCfg,
		Device:  args[0],
	}
	return cfg, cfg.Validate()
}

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

			return nwctl.RunServiceCompile(cmd.Context(), cfg)
		},
	}
	return cmd
}

func newServiceCompileCfg(cmd *cobra.Command, args []string) (*nwctl.ServiceCompileCfg, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("service is not specified")
	}
	rootCfg, err := newRootCfg(cmd)
	if err != nil {
		return nil, err
	}
	cfg := &nwctl.ServiceCompileCfg{
		RootCfg: *rootCfg,
		Service: args[0],
		Keys:    args[1:],
	}
	return cfg, cfg.Validate()
}

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
	"github.com/hrk091/nwctl/pkg/logger"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/spf13/cobra"
)

func newGitMergeDevicesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge-devices",
		Short: "Merge subscribed device config updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newGitMergeDevicesCfg(cmd, args)
			if err != nil {
				return err
			}
			logger.Setup(cfg.Devel, cfg.Verbose)

			return nwctl.RunGitMergeDevicesCfg(cmd.Context(), cfg)
		},
	}
	return cmd
}

func newGitMergeDevicesCfg(cmd *cobra.Command, args []string) (*nwctl.GitMergeDevicesCfg, error) {
	rootCfg, err := newRootCfg(cmd)
	if err != nil {
		return nil, err
	}
	cfg := &nwctl.GitMergeDevicesCfg{
		RootCfg: *rootCfg,
	}
	return cfg, cfg.Validate()
}

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
	"github.com/spf13/viper"
)

func newGitCommitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Commit and push to remote your local changes both staged and not staged",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newGitCommitCfg(cmd, args)
			if err != nil {
				return err
			}
			logger.Setup(cfg.Devel, cfg.Verbose)

			return nwctl.RunGitCommit(cmd.Context(), cfg)
		},
	}
	cmd.Flags().BoolP(FlagPushToMain, "", false, "push to main (otherwise create new branch)")
	mustBindToViper(cmd)

	return cmd
}

func newGitCommitCfg(cmd *cobra.Command, args []string) (*nwctl.GitCommitCfg, error) {
	rootCfg, err := newRootCfg(cmd)
	if err != nil {
		return nil, err
	}
	cfg := &nwctl.GitCommitCfg{
		RootCfg:    *rootCfg,
		PushToMain: viper.GetBool(FlagPushToMain),
	}
	return cfg, cfg.Validate()
}

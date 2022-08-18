package cmd

import (
	"github.com/hrk091/nwctl/pkg/logger"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/spf13/cobra"
)

func newServiceApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply all changed service config and generate new device config of affected devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newServiceApplyCfg(cmd, args)
			if err != nil {
				return err
			}
			logger.Setup(cfg.Devel, cfg.Verbose)

			return nwctl.RunServiceApply(cmd.Context(), cfg)
		},
	}
	return cmd
}

func newServiceApplyCfg(cmd *cobra.Command, args []string) (*nwctl.ServiceApplyCfg, error) {
	rootCfg, err := newRootCfg(cmd)
	if err != nil {
		return nil, err
	}
	cfg := &nwctl.ServiceApplyCfg{
		RootCfg: *rootCfg,
	}
	return cfg, cfg.Validate()
}

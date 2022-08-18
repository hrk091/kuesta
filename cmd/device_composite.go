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

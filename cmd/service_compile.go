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

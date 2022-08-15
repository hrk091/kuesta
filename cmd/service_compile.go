package cmd

import (
	"context"
	"fmt"
	"github.com/hrk091/nwctl/pkg/logger"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/spf13/cobra"
)

// NewServiceCompileCmd creates the service-compile command
func NewServiceCompileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile service config to partial device config",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := newServiceCompileCfg(cmd, args)
			logger.Setup(cfg.Devel, cfg.Verbose)
			ctx := logger.WithLogger(context.Background(), logger.NewLogger())
			cobra.CheckErr(nwctl.RunServiceCompile(ctx, cfg))
		},
	}
	return cmd
}

func newServiceCompileCfg(cmd *cobra.Command, args []string) nwctl.ServiceCompileCfg {
	if len(args) == 0 {
		cobra.CheckErr(fmt.Errorf(""))
	}
	cfg := nwctl.ServiceCompileCfg{
		RootCfg: *newRootCfg(cmd),
		Service: args[0],
		Keys:    args[1:],
	}
	cobra.CheckErr(cfg.Validate())
	return cfg
}

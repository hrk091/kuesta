package cmd

import (
	"context"
	"github.com/hrk091/nwctl/pkg/logger"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/spf13/cobra"
)

// serviceCompileCmd represents the service-compile command
var serviceCompileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile service config to partial device config",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := newServiceCompileCfg(cmd, args)
		logger.Setup(cfg.Devel, cfg.Verbose)
		ctx := logger.WithLogger(context.Background(), logger.NewLogger())
		cobra.CheckErr(nwctl.RunServiceCompile(ctx, cfg))
	},
}

func init() {
}

func newServiceCompileCfg(cmd *cobra.Command, args []string) nwctl.ServiceCompileCfg {
	return nwctl.ServiceCompileCfg{
		RootCfg: *newRootCfg(cmd),
	}
}

package cmd

import (
	"github.com/spf13/cobra"
)

func newServiceCmd() *cobra.Command {
	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Manage services",
	}
	serviceCmd.AddCommand(newServiceCompileCmd())
	serviceCmd.AddCommand(newServiceApplyCmd())
	return serviceCmd
}

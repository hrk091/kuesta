package cmd

import (
	"github.com/spf13/cobra"
)

func newServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage services",
	}
	cmd.AddCommand(newServiceCompileCmd())
	cmd.AddCommand(newServiceApplyCmd())
	return cmd
}

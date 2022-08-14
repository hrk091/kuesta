package cmd

import (
	"github.com/spf13/cobra"
)

func NewServiceCmd() *cobra.Command {
	var serviceCmd = &cobra.Command{
		Use:   "service",
		Short: "Manage services",
	}
	serviceCmd.AddCommand(NewServiceCompileCmd())
	return serviceCmd
}

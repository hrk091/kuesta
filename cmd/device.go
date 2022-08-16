package cmd

import "github.com/spf13/cobra"

func newDeviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "Manage devices",
	}
	cmd.AddCommand(newDeviceCompositeCmd())
	return cmd
}

package cmd

import (
	"github.com/spf13/cobra"
)

func newGitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git",
		Short: "Execute Git operations",
	}
	cmd.AddCommand(newGitCommitCmd())
	return cmd
}

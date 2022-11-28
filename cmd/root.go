/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package cmd

import (
	"fmt"
	"github.com/nttcom/kuesta/pkg/gogit"
	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/nttcom/kuesta/pkg/logger"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var cfgFile string

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		logger.ShowStackTrace(os.Stderr, err)
		// NOTE add show cmd.UsageString() for the specific error if needed
		os.Exit(1)
	}
}

const (
	FlagConfig         = "config"
	FlagDevel          = "devel"
	FlagVerbose        = "verbose"
	FlagConfigRootPath = "config-root-path"
	FlagStatusRootPath = "status-root-path"
	FlagConfigRepoUrl  = "config-repo-url"
	FlagStatusRepoUrl  = "status-repo-url"
	FlagGitTrunk       = "git-trunk"
	FlagGitRemote      = "git-remote-name"
	FlagGitToken       = "git-token"
	FlagGitUser        = "git-user"
	FlagGitEmail       = "git-email"
	FlagPushToMain     = "push-to-main"
)

// NewRootCmd creates command root.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "nwctl",
		Short:        "nwctl controls Network Element Configurations.",
		SilenceUsage: true,
	}

	cobra.OnInitialize(initConfig)

	cmd.PersistentFlags().StringVar(&cfgFile, FlagConfig, "", "config file (default is $HOME/.nwctl.yaml)")

	cmd.PersistentFlags().Uint8P(FlagVerbose, "v", 0, "verbose level")
	cmd.PersistentFlags().BoolP(FlagDevel, "", false, "enable development mode")
	cmd.PersistentFlags().StringP(FlagConfigRootPath, "p", "", "path to the config repository root")
	cmd.PersistentFlags().StringP(FlagStatusRootPath, "", "", "path to the status repository root")
	cmd.PersistentFlags().StringP(FlagConfigRepoUrl, "r", "", "git config repository url")
	cmd.PersistentFlags().StringP(FlagStatusRepoUrl, "", "", "git status repository url")
	cmd.PersistentFlags().StringP(FlagGitTrunk, "", gogit.DefaultTrunkBranch, "git trunk branch")
	cmd.PersistentFlags().StringP(FlagGitRemote, "", gogit.DefaultRemoteName, "git remote name to be used for gitops")
	cmd.PersistentFlags().StringP(FlagGitToken, "", "", "git auth token")
	cmd.PersistentFlags().StringP(FlagGitUser, "", gogit.DefaultGitUser, "git username")
	cmd.PersistentFlags().StringP(FlagGitEmail, "", gogit.DefaultGitEmail, "git email")

	mustBindToViper(cmd)
	cmd.Version = getVcsRevision()

	cmd.AddCommand(newServiceCmd())
	cmd.AddCommand(newDeviceCmd())
	cmd.AddCommand(newGitCmd())
	cmd.AddCommand(newServeCmd())

	return cmd
}

func newRootCfg(cmd *cobra.Command) (*kuesta.RootCfg, error) {
	gitUser := viper.GetString(FlagGitUser)
	gitEmail := viper.GetString(FlagGitEmail)
	if gitUser != gogit.DefaultGitUser && gitEmail == gogit.DefaultGitEmail {
		gitEmail = fmt.Sprintf("%s@example.com", gitUser)
	}

	cfg := &kuesta.RootCfg{
		Verbose:        cast.ToUint8(viper.GetUint(FlagVerbose)),
		Devel:          viper.GetBool(FlagDevel),
		ConfigRootPath: viper.GetString(FlagConfigRootPath),
		ConfigRepoUrl:  viper.GetString(FlagConfigRepoUrl),
		StatusRootPath: viper.GetString(FlagStatusRootPath),
		StatusRepoUrl:  viper.GetString(FlagStatusRepoUrl),
		GitTrunk:       viper.GetString(FlagGitTrunk),
		GitToken:       viper.GetString(FlagGitToken),
		GitRemote:      viper.GetString(FlagGitRemote),
		GitUser:        gitUser,
		GitEmail:       gitEmail,
	}
	return cfg, cfg.Validate()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".nwctl" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".nwctl")
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("NWCTL")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

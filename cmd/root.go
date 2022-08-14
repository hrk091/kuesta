/*
 * Copyright (c) 2022. Hiroki Okui
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package cmd

import (
	"fmt"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nwctl",
	Short: "nwctl controls Network Element Configurations.",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

const (
	FlagConfig   = "config"
	FlagDevel    = "devel"
	FlagRootPath = "rootpath"
	FlagVerbose  = "verbose"
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, FlagConfig, "", "config file (default is $HOME/.nwctl.yaml)")

	rootCmd.PersistentFlags().Uint8P(FlagVerbose, "v", 0, "verbose level")
	rootCmd.PersistentFlags().BoolP(FlagDevel, "d", false, "enable development mode")
	rootCmd.PersistentFlags().StringP(FlagRootPath, "r", "", "path to the repository root")
	_ = rootCmd.MarkPersistentFlagRequired(FlagRootPath)

	mustBindToViper(rootCmd)

	rootCmd.Version = getVcsRevision()

	rootCmd.AddCommand(serviceCmd)
}

func newRootCfg(cmd *cobra.Command) *nwctl.RootCfg {
	verbose := cast.ToUint8(viper.GetUint(FlagVerbose))
	devel := viper.GetBool(FlagDevel)
	rootpath := viper.GetString(FlagRootPath)

	cfg, err := nwctl.NewRootCfg().Verbose(verbose).Devel(devel).RootPath(rootpath).Build()
	cobra.CheckErr(err)
	return cfg
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

	viper.SetEnvPrefix("NWCTL")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

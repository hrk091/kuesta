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

package main

import (
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	Devel         bool
	Verbose       uint8
	Addr          string `validate:"required"`
	Username      string
	Password      string
	Device        string `validate:"required"`
	AggregatorURL string `mapstructure:"aggregator-url" validate:"required"`
}

// Validate validates exposed fields according to the `validate` tag.
func (c *Config) Validate() error {
	return common.Validate(c)
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "nwctl-subscribe",
		Short:        "device-subscribe subscribes Network Element Configuration Update.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg Config
			if err := viper.Unmarshal(&cfg); err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return err
			}
			logger.Setup(cfg.Devel, cfg.Verbose)
			return Run(cfg)
		},
	}

	cmd.Flags().BoolP("devel", "", false, "enable development mode")
	cmd.Flags().Uint8P("verbose", "v", 0, "verbose level")
	cmd.Flags().StringP("addr", "a", "", "Address of the target device, address:port or just :port")
	cmd.Flags().StringP("username", "u", "admin", "Username of the target device")
	cmd.Flags().StringP("password", "p", "admin", "Password of the target device")
	cmd.Flags().StringP("device", "d", "", "Name of the target device")
	cmd.Flags().StringP("aggregator-url", "", "", "URL of the aggregator")

	cobra.CheckErr(viper.BindPFlags(cmd.Flags()))
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("NWCTL")
	viper.AutomaticEnv()

	return cmd
}

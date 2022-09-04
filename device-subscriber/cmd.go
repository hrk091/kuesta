/*
 Copyright 2022 NTT Communications Corporation.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
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

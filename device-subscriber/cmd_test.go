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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfig_Validate(t *testing.T) {

	newValidStruct := func(t func(*Config)) *Config {
		cfg := &Config{
			Device:        "device1",
			Addr:          ":9339",
			AggregatorURL: "http://localhost:8000",
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *Config)
		wantErr   bool
	}{
		{
			"ok",
			func(cfg *Config) {},
			false,
		},
		{
			"bad: device is empty",
			func(cfg *Config) {
				cfg.Device = ""
			},
			true,
		},
		{
			"bad: addr is empty",
			func(cfg *Config) {
				cfg.Addr = ""
			},
			true,
		},
		{
			"bad: aggregator-url is empty",
			func(cfg *Config) {
				cfg.AggregatorURL = ""
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidStruct(tt.transform)
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestNewRootCmd(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			"bad: device not set",
			[]string{"nwctl-subscribe", "-addr=:9339", "-aggregator-url=http://localhost:8080"},
			true,
		},
		{
			"bad: addr not set",
			[]string{"nwctl-subscribe", "-d=device1", "-aggregator-url=http://localhost:8080"},
			true,
		},
		{
			"bad: aggregator-url not set",
			[]string{"nwctl-subscribe", "-d=device1", "-addr=:9339"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRootCmd()
			c.SetArgs(tt.args)
			err := c.Execute()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

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

package nwctl_test

import (
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRootCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(*nwctl.RootCfg)) *nwctl.RootCfg {
		cfg := &nwctl.RootCfg{
			Verbose:  0,
			Devel:    false,
			RootPath: "./",
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *nwctl.RootCfg)
		wantError bool
	}{
		{
			"ok",
			func(cfg *nwctl.RootCfg) {},
			false,
		},
		{
			"bad: Verbose is over range",
			func(cfg *nwctl.RootCfg) {
				cfg.Verbose = 4
			},
			true,
		},
		{
			"bad: RootPath is empty",
			func(cfg *nwctl.RootCfg) {
				cfg.RootPath = ""
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidStruct(tt.transform)
			err := cfg.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

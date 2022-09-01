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

package cmd_test

import (
	"github.com/hrk091/nwctl/cmd"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRootCmd_ServiceCompile(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			"bad: service not set",
			[]string{"service", "compile", "-r=./"},
			true,
		},
		{
			"bad: keys not set",
			[]string{"service", "compile", "abc", "-r=./"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cmd.NewRootCmd()
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

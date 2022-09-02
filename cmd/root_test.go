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
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	dummyToken := "dummy-git-token"
	exitOnErr(t, os.Setenv("NWCTL_GIT_TOKEN", dummyToken))

	dummyRootpath := "dummy-rootpath"
	exitOnErr(t, os.Setenv("NWCTL_ROOTPATH", dummyRootpath))

	_ = cmd.NewRootCmd()
	assert.Equal(t, dummyToken, viper.GetString(cmd.FlagGitToken))
	assert.Equal(t, dummyRootpath, viper.GetString(cmd.FlagRootPath))
}

func exitOnErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

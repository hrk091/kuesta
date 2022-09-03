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

package nwctl

import (
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/gogit"
)

type RootCfg struct {
	Verbose    uint8 `validate:"min=0,max=3"`
	Devel      bool
	RootPath   string `validate:"required"`
	GitRepoUrl string
	GitTrunk   string
	GitRemote  string
	GitToken   string
	GitUser    string
	GitEmail   string
}

// Validate validates exposed fields according to the `validate` tag.
func (c *RootCfg) Validate() error {
	return common.Validate(c)
}

func (c *RootCfg) GitOptions() *gogit.GitOptions {
	return &gogit.GitOptions{
		RepoUrl:     c.GitRepoUrl,
		Path:        c.RootPath,
		TrunkBranch: c.GitTrunk,
		RemoteName:  c.GitRemote,
		Token:       c.GitToken,
		User:        c.GitUser,
		Email:       c.GitEmail,
	}
}

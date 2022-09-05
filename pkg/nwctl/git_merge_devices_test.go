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
	"context"
	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGitMergeDevicesCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(cfg *nwctl.GitMergeDevicesCfg)) *nwctl.GitMergeDevicesCfg {
		cfg := &nwctl.GitMergeDevicesCfg{
			RootCfg: nwctl.RootCfg{
				ConfigRootPath: "./",
			},
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *nwctl.GitMergeDevicesCfg)
		wantError bool
	}{
		{
			"ok",
			func(cfg *nwctl.GitMergeDevicesCfg) {},
			false,
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

func TestRunGitMergeDevicesCfg(t *testing.T) {
	testRemote := "test-remote"
	syncBranch := "SYNC-test"

	t.Run("ok", func(t *testing.T) {
		repo, _, dirBare := setupGitRepoWithRemote(t, testRemote)

		exitOnErr(t, createBranch(repo, syncBranch))
		wantHash, err := commit(repo, time.Now())
		exitOnErr(t, err)
		exitOnErr(t, push(repo, syncBranch, testRemote))

		repoPuller, dirPuller := cloneRepo(t, &extgogit.CloneOptions{
			URL:           dirBare,
			RemoteName:    testRemote,
			ReferenceName: plumbing.NewBranchReferenceName("main"),
		})
		assert.True(t, hasSyncBranch(t, repo, testRemote))

		err = nwctl.RunGitMergeDevicesCfg(context.Background(), &nwctl.GitMergeDevicesCfg{
			RootCfg: nwctl.RootCfg{
				ConfigRootPath: dirPuller,
				GitRemote:      testRemote,
			},
		})
		exitOnErr(t, err)
		h, err := repoPuller.Head()
		exitOnErr(t, err)
		assert.Equal(t, wantHash, h.Hash())
		assert.Equal(t, "main", h.Name().Short())
		assert.False(t, hasSyncBranch(t, repo, testRemote))
	})

	t.Run("ok: no sync branch", func(t *testing.T) {
		repo, _, dirBare := setupGitRepoWithRemote(t, testRemote)

		repoPuller, dirPuller := cloneRepo(t, &extgogit.CloneOptions{
			URL:           dirBare,
			RemoteName:    testRemote,
			ReferenceName: plumbing.NewBranchReferenceName("main"),
		})
		assert.False(t, hasSyncBranch(t, repo, testRemote))

		err := nwctl.RunGitMergeDevicesCfg(context.Background(), &nwctl.GitMergeDevicesCfg{
			RootCfg: nwctl.RootCfg{
				ConfigRootPath: dirPuller,
				GitRemote:      testRemote,
			},
		})
		assert.Nil(t, err)
		h, err := repoPuller.Head()
		exitOnErr(t, err)
		assert.Equal(t, "main", h.Name().Short())
		assert.False(t, hasSyncBranch(t, repo, testRemote))
	})

}

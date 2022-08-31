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
				RootPath: "./",
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
				RootPath:  dirPuller,
				GitRemote: testRemote,
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
				RootPath:  dirPuller,
				GitRemote: testRemote,
			},
		})
		assert.Nil(t, err)
		h, err := repoPuller.Head()
		exitOnErr(t, err)
		assert.Equal(t, "main", h.Name().Short())
		assert.False(t, hasSyncBranch(t, repo, testRemote))
	})

}

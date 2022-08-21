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
	"github.com/go-git/go-git/v5/config"
	"github.com/hrk091/nwctl/pkg/gogit"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestRunGitCommit(t *testing.T) {
	setup := func(t *testing.T) (*extgogit.Repository, string) {
		_, url := initBareRepo(t)

		repo, dir := initRepo(t, "main")
		_, err := repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{url},
		})
		ExitOnErr(t, err)

		ExitOnErr(t, addFile(repo, "services/foo/one/input.cue", "{}"))
		ExitOnErr(t, addFile(repo, "devices/device1/config.cue", "{}"))
		_, err = commit(repo, time.Now())
		ExitOnErr(t, err)

		ExitOnErr(t, deleteFile(repo, "services/foo/one/input.cue"))
		ExitOnErr(t, addFile(repo, "services/foo/two/input.cue", "{}"))
		ExitOnErr(t, addFile(repo, "services/foo/three/input.cue", "{}"))
		ExitOnErr(t, deleteFile(repo, "devices/device1/config.cue"))
		ExitOnErr(t, addFile(repo, "devices/device2/config.cue", "{}"))
		ExitOnErr(t, addFile(repo, "devices/device3/config.cue", "{}"))
		return repo, dir
	}

	wantMsg := `Updated: services/foo/three,services/foo/two,services/foo/one

Services:
	added:     services/foo/three
	added:     services/foo/two
	deleted:   services/foo/one

Devices:
	added:     device2
	added:     device3
	deleted:   device1`

	t.Run("ok: push to main", func(t *testing.T) {
		repo, dir := setup(t)
		err := nwctl.RunGitCommit(context.Background(), &nwctl.GitCommitCfg{
			RootCfg: nwctl.RootCfg{
				RootPath:  dir,
				GitBranch: "main",
			},
			PushToMain: true,
		})
		assert.Nil(t, err)
		assert.Equal(t, "main", getBranch(t, repo))

		g, err := gogit.NewGit(gogit.GitOptions{
			Path:       dir,
			MainBranch: "main",
		})
		ExitOnErr(t, err)
		h, err := g.Head()
		ExitOnErr(t, err)
		assert.Equal(t, wantMsg, h.Message)
	})

	t.Run("ok: push to new branch", func(t *testing.T) {
		repo, dir := setup(t)
		err := nwctl.RunGitCommit(context.Background(), &nwctl.GitCommitCfg{
			RootCfg: nwctl.RootCfg{
				RootPath:  dir,
				GitBranch: "main",
			},
			PushToMain: false,
		})
		assert.Nil(t, err)
		assert.True(t, strings.HasPrefix(getBranch(t, repo), "REV-"))

		g, err := gogit.NewGit(gogit.GitOptions{
			Path:       dir,
			MainBranch: "main",
		})
		ExitOnErr(t, err)
		h, err := g.Head()
		ExitOnErr(t, err)
		assert.Equal(t, wantMsg, h.Message)
	})

}

func TestMakeCommitMessage(t *testing.T) {
	stmap := extgogit.Status{
		"services/svc1/k1/input.cue":       &extgogit.FileStatus{Staging: extgogit.Added},
		"services/svc2/k1/k2/input.cue":    &extgogit.FileStatus{Staging: extgogit.Deleted},
		"services/svc3/k1/k2/k3/input.cue": &extgogit.FileStatus{Staging: extgogit.Modified},
		"devices/dvc1/config.cue":          &extgogit.FileStatus{Staging: extgogit.Added},
		"devices/dvc2/config.cue":          &extgogit.FileStatus{Staging: extgogit.Deleted},
		"devices/dvc3/config.cue":          &extgogit.FileStatus{Staging: extgogit.Modified},
	}
	want := `Updated: services/svc1/k1,services/svc2/k1/k2,services/svc3/k1/k2/k3

Services:
	added:     services/svc1/k1
	deleted:   services/svc2/k1/k2
	modified:  services/svc3/k1/k2/k3

Devices:
	added:     dvc1
	deleted:   dvc2
	modified:  dvc3`
	assert.Equal(t, want, nwctl.MakeCommitMessage(stmap))
}

func TestCheckGitFileIsStagedOrUnmodified(t *testing.T) {
	tests := []struct {
		path    string
		st      extgogit.FileStatus
		wantErr bool
	}{
		{
			"ok/staging_modified",
			extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Unmodified},
			false,
		},
		{
			"ok/worktree_modified",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
			false,
		},
		{
			"bad/both_modified",
			extgogit.FileStatus{Staging: extgogit.Added, Worktree: extgogit.Modified},
			true,
		},
		{
			"bad/worktree_updated_but_unmerged",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.UpdatedButUnmerged},
			true,
		},
	}

	for _, tt := range tests {
		err := nwctl.CheckGitFileIsStagedOrUnmodified(tt.path, tt.st)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestCheckGitIsStagedOrUnmodified(t *testing.T) {
	tests := []struct {
		st      extgogit.Status
		wantErr int
	}{
		{
			extgogit.Status{
				"ok/staging_modified":  &extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Unmodified},
				"ok/worktree_modified": &extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
			},
			0,
		},
		{
			extgogit.Status{
				"ok/staging_modified":               &extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Unmodified},
				"bad/both_modified":                 &extgogit.FileStatus{Staging: extgogit.Added, Worktree: extgogit.Modified},
				"bad/worktree_updated_but_unmerged": &extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.UpdatedButUnmerged},
			},
			2,
		},
	}

	for _, tt := range tests {
		err := nwctl.CheckGitIsStagedOrUnmodified(tt.st)
		if tt.wantErr > 0 {
			reg := regexp.MustCompile("bad")
			assert.Equal(t, tt.wantErr, len(reg.FindAllString(err.Error(), -1)))
		} else {
			assert.Nil(t, err)
		}
	}

}

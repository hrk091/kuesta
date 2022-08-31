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

package gogit_test

import (
	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hrk091/nwctl/pkg/gogit"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGitOptions_Validate(t *testing.T) {
	newValidStruct := func(t func(git *gogit.GitOptions)) *gogit.GitOptions {
		g := &gogit.GitOptions{
			Path: "./",
		}
		t(g)
		return g
	}

	tests := []struct {
		name      string
		transform func(g *gogit.GitOptions)
		wantErr   bool
	}{
		{
			"ok",
			func(g *gogit.GitOptions) {},
			false,
		},
		{
			"bad: path is empty",
			func(g *gogit.GitOptions) {
				g.Path = ""
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newValidStruct(tt.transform)
			err := g.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestGit_BasicAuth(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  *githttp.BasicAuth
	}{
		{
			"both username and password",
			"user:pass",
			&githttp.BasicAuth{
				Username: "user",
				Password: "pass",
			},
		},
		{
			"only password",
			"pass",
			&githttp.BasicAuth{
				Username: "anonymous",
				Password: "pass",
			},
		},
		{
			"not set",
			"",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gogit.NewGitWithoutRepo(&gogit.GitOptions{
				Token: tt.token,
			})
			assert.Equal(t, tt.want, g.BasicAuth())
		})
	}
}

func TestGit_Signature(t *testing.T) {
	t.Run("given user/email", func(t *testing.T) {
		g := gogit.NewGitWithoutRepo(&gogit.GitOptions{
			User:  "test-user",
			Email: "test-email",
		})
		got := g.Signature()
		assert.Equal(t, "test-user", got.Name)
		assert.Equal(t, "test-email", got.Email)
	})
	t.Run("default", func(t *testing.T) {
		g := gogit.NewGitWithoutRepo(&gogit.GitOptions{})
		got := g.Signature()
		assert.Equal(t, gogit.DefaultGitUser, got.Name)
		assert.Equal(t, gogit.DefaultGitEmail, got.Email)
	})
}

func TestGit_Head(t *testing.T) {
	repo, dir := initRepo(t, "main")
	exitOnErr(t, addFile(repo, "test", "hash"))
	want, err := commit(repo, time.Now())
	exitOnErr(t, err)

	g, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	exitOnErr(t, err)

	c, err := g.Head()
	exitOnErr(t, err)
	assert.Equal(t, want, c.Hash)
}

func TestGit_Checkout(t *testing.T) {
	t.Run("ok: checkout to main", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		exitOnErr(t, err)

		_, err = g.Checkout()
		exitOnErr(t, err)

		b, err := g.Branch()
		exitOnErr(t, err)
		assert.Equal(t, "main", b)
	})

	t.Run("ok: checkout to specified trunk", func(t *testing.T) {
		branchName := "test-branch"
		_, dir := initRepo(t, branchName)
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:        dir,
			TrunkBranch: branchName,
		})
		exitOnErr(t, err)

		_, err = g.Checkout()
		exitOnErr(t, err)

		b, err := g.Branch()
		exitOnErr(t, err)
		assert.Equal(t, branchName, b)
	})

	t.Run("ok: checkout to new branch", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		exitOnErr(t, err)

		_, err = g.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		exitOnErr(t, err)

		b, err := g.Branch()
		exitOnErr(t, err)
		assert.Equal(t, "test", b)
	})

	t.Run("bad: checkout to existing branch with create opt", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		exitOnErr(t, err)
		exitOnErr(t, createBranch(repo, "test"))

		_, err = g.Checkout(gogit.CheckoutOptsTo("main"), gogit.CheckoutOptsCreateNew())
		assert.Error(t, err)
	})

	t.Run("bad: checkout to new branch without create opt", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		exitOnErr(t, err)

		_, err = g.Checkout(gogit.CheckoutOptsTo("test"))
		assert.Error(t, err)
	})
}

func TestGit_Commit(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		exitOnErr(t, addFile(repo, "test", "dummy"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		exitOnErr(t, err)
		_, err = g.Commit("added: test")
		assert.Nil(t, err)
	})

	t.Run("ok: other trunk branch", func(t *testing.T) {
		testTrunk := "test-branch"
		repo, dir := initRepo(t, testTrunk)
		exitOnErr(t, addFile(repo, "test", "dummy"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:        dir,
			TrunkBranch: testTrunk,
		})
		exitOnErr(t, err)
		h, err := g.Commit("added: test")
		assert.Nil(t, err)

		b, err := g.Branch()
		assert.Equal(t, testTrunk, b)

		c, err := g.Head()
		assert.Equal(t, h, c.Hash)
	})

	t.Run("ok: commit even when no change", func(t *testing.T) {
		_, dir := initRepo(t, "main")

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		exitOnErr(t, err)
		_, err = g.Commit("no change")
		assert.Nil(t, err)
	})
}

func TestGit_Push(t *testing.T) {
	remoteRepo, url := initBareRepo(t)

	t.Run("ok", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		testRemote := "test-remote"
		_, err := repo.CreateRemote(&config.RemoteConfig{
			Name: testRemote,
			URLs: []string{url},
		})
		exitOnErr(t, err)

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dir,
			RemoteName: testRemote,
		})
		exitOnErr(t, err)
		exitOnErr(t, addFile(repo, "test", "push"))
		wantMsg := "git commit which should be pushed to remote"
		_, err = g.Commit(wantMsg)
		exitOnErr(t, err)

		err = g.Push("main")
		exitOnErr(t, err)

		ref, err := remoteRepo.Reference(plumbing.NewBranchReferenceName("main"), false)
		exitOnErr(t, err)
		c, err := repo.CommitObject(ref.Hash())
		exitOnErr(t, err)

		assert.Equal(t, wantMsg, c.Message)
	})

	t.Run("bad: remote not exist", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		noExistRemote := "not-exist"

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dir,
			RemoteName: noExistRemote,
		})
		exitOnErr(t, err)
		exitOnErr(t, addFile(repo, "test", "push"))
		_, err = g.Commit("added: test")
		exitOnErr(t, err)

		err = g.Push("main")
		assert.Error(t, err)
	})
}

func TestGit_SetUpstream(t *testing.T) {
	_, dirBare := initBareRepo(t)
	testRemote := "test-remote"

	repo, dir := initRepo(t, "main")
	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: testRemote,
		URLs: []string{dirBare},
	})
	exitOnErr(t, err)
	git, err := gogit.NewGit(&gogit.GitOptions{
		Path:       dir,
		RemoteName: testRemote,
	})
	exitOnErr(t, err)

	_, err = git.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
	exitOnErr(t, err)

	err = git.SetUpstream("test")
	assert.Nil(t, err)

	c, err := repo.Storer.Config()
	exitOnErr(t, err)

	exists := false
	for name, r := range c.Branches {
		if name == "test" {
			assert.Equal(t, testRemote, r.Remote)
			assert.Equal(t, plumbing.NewBranchReferenceName("test"), r.Merge)
			exists = true
		}
	}
	assert.True(t, exists)
}

func TestGit_Branches(t *testing.T) {
	_, dir := initRepo(t, "main")
	git, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	exitOnErr(t, err)

	_, err = git.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
	exitOnErr(t, err)

	branches, err := git.Branches()
	want := []string{"main", "test"}
	for _, w := range want {
		assert.Contains(t, branches, w)
	}
}

func TestGit_Pull(t *testing.T) {
	testRemote := "test-remote"

	setup := func(t *testing.T, beforeCloneFn func(*gogit.Git)) (*gogit.Git, *gogit.Git) {
		// setup remote bare repo
		_, dirBare := initBareRepo(t)

		// setup pusher
		repoPusher, dirPusher := initRepo(t, "main")
		_, err := repoPusher.CreateRemote(&config.RemoteConfig{
			Name: testRemote,
			URLs: []string{dirBare},
		})
		exitOnErr(t, err)
		gitPusher, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dirPusher,
			RemoteName: testRemote,
		})
		exitOnErr(t, err)

		_, err = gitPusher.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		exitOnErr(t, err)

		if beforeCloneFn != nil {
			beforeCloneFn(gitPusher)
		}

		// setup puller by git clone
		_, dirPuller := cloneRepo(t, &extgogit.CloneOptions{
			URL:        dirBare,
			RemoteName: testRemote,
		})
		gitPuller, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dirPuller,
			RemoteName: testRemote,
		})
		exitOnErr(t, err)

		return gitPusher, gitPuller
	}

	t.Run("ok", func(t *testing.T) {
		gitPusher, gitPuller := setup(t, func(pusher *gogit.Git) {
			exitOnErr(t, pusher.Push("master"))
			exitOnErr(t, pusher.Push("test"))
		})

		// push branch
		exitOnErr(t, addFile(gitPuller.Repo(), "test", "push"))
		wantMsg := "git commit which should be pushed to remote"
		want, err := gitPusher.Commit(wantMsg)
		exitOnErr(t, err)

		exitOnErr(t, gitPusher.Push("test"))

		// pull branch
		_, err = gitPuller.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		exitOnErr(t, err)

		err = gitPuller.Pull()
		exitOnErr(t, err)

		got, err := gitPuller.Head()
		assert.Equal(t, want.String(), got.Hash.String())
	})

	t.Run("ok: no update", func(t *testing.T) {
		gitPusher, gitPuller := setup(t, func(pusher *gogit.Git) {
			exitOnErr(t, pusher.Push("master"))
			exitOnErr(t, pusher.Push("test"))
		})

		head, err := gitPusher.Head()
		exitOnErr(t, err)
		want := head.Hash

		// pull branch
		_, err = gitPuller.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		exitOnErr(t, err)

		err = gitPuller.Pull()
		exitOnErr(t, err)

		got, err := gitPuller.Head()
		assert.Equal(t, want.String(), got.Hash.String())
	})

	t.Run("err: upstream branch not exist", func(t *testing.T) {
		_, gitPuller := setup(t, func(g *gogit.Git) {
			exitOnErr(t, g.Push("master"))
		})

		_, err := gitPuller.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		exitOnErr(t, err)

		err = gitPuller.Pull()
		assert.Error(t, err)
	})

	t.Run("err: remote repo not exist", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		git, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dir,
			RemoteName: testRemote,
		})
		exitOnErr(t, err)

		err = git.Pull()
		assert.Error(t, err)
	})

}

func TestGit_Reset(t *testing.T) {
	repo, dir := initRepo(t, "main")
	git, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	exitOnErr(t, err)
	exitOnErr(t, addFile(repo, "test", "hash"))

	w, err := repo.Worktree()
	exitOnErr(t, err)

	st, err := w.Status()
	exitOnErr(t, err)
	assert.Greater(t, len(st), 0)

	err = git.Reset(gogit.ResetOptsHard())
	assert.Nil(t, err)

	st, err = w.Status()
	exitOnErr(t, err)
	assert.Equal(t, len(st), 0)
}

func TestIsTrackedAndChanged(t *testing.T) {
	tests := []struct {
		given extgogit.StatusCode
		want  bool
	}{
		{extgogit.Unmodified, false},
		{extgogit.Untracked, false},
		{extgogit.Modified, true},
		{extgogit.Added, true},
		{extgogit.Deleted, true},
		{extgogit.Renamed, true},
		{extgogit.Copied, true},
		{extgogit.UpdatedButUnmerged, true},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, gogit.IsTrackedAndChanged(tt.given))
	}
}

func TestIsBothWorktreeAndStagingTrackedAndChanged(t *testing.T) {
	tests := []struct {
		given extgogit.FileStatus
		want  bool
	}{
		{extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Modified}, true},
		{extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Unmodified}, false},
		{extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified}, false},
		{extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified}, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, gogit.IsBothWorktreeAndStagingTrackedAndChanged(tt.given))
	}
}

func TestIsEitherWorktreeOrStagingTrackedAndChanged(t *testing.T) {
	tests := []struct {
		given extgogit.FileStatus
		want  bool
	}{
		{extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Modified}, true},
		{extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Unmodified}, true},
		{extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified}, true},
		{extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified}, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, gogit.IsEitherWorktreeOrStagingTrackedAndChanged(tt.given))
	}
}

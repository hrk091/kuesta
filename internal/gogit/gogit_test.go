/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package gogit_test

import (
	"testing"
	"time"

	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/nttcom/kuesta/internal/gogit"
	"github.com/nttcom/kuesta/pkg/common"
	"github.com/stretchr/testify/assert"
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
			"err: path is empty",
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

func TestNewGit(t *testing.T) {
	t.Run("ok: use existing", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		opt := &gogit.GitOptions{
			Path: dir,
		}
		common.ExitOnErr(t, opt.Validate())

		g, err := gogit.NewGit(opt)
		assert.Nil(t, err)
		assert.Equal(t, opt, g.Options())

		want, _ := repo.Head()
		got, _ := g.Repo().Head()
		assert.Equal(t, want, got)
	})

	t.Run("ok: clone", func(t *testing.T) {
		repoPusher, dir, remoteUrl := initRepoWithRemote(t, "main")
		common.ExitOnErr(t, push(repoPusher, "main", "origin"))

		opt := &gogit.GitOptions{
			RepoUrl: remoteUrl,
			Path:    dir,
		}
		common.ExitOnErr(t, opt.Validate())
		opt.ShouldCloneIfNotExist()

		g, err := gogit.NewGit(opt)
		assert.Nil(t, err)
		assert.Equal(t, opt, g.Options())

		want, _ := repoPusher.Head()
		got, _ := g.Repo().Head()
		assert.Equal(t, want, got)
	})

	t.Run("err: no repo without shouldClone flag", func(t *testing.T) {
		repoPusher, _, remoteUrl := initRepoWithRemote(t, "main")
		common.ExitOnErr(t, push(repoPusher, "main", "origin"))

		dir := t.TempDir()
		opt := &gogit.GitOptions{
			RepoUrl: remoteUrl,
			Path:    dir,
		}
		common.ExitOnErr(t, opt.Validate())

		_, err := gogit.NewGit(opt)
		assert.Error(t, err)
	})
}

func TestGit_Clone(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		repoPusher, _, remoteUrl := initRepoWithRemote(t, "main")
		common.ExitOnErr(t, push(repoPusher, "main", "origin"))

		dir := t.TempDir()
		g := gogit.NewGitWithoutRepo(&gogit.GitOptions{
			RepoUrl: remoteUrl,
			Path:    dir,
		})
		common.ExitOnErr(t, g.Options().Validate())

		_, err := g.Clone()
		assert.Nil(t, err)
	})

	t.Run("err: url not given", func(t *testing.T) {
		dir := t.TempDir()
		g := gogit.NewGitWithoutRepo(&gogit.GitOptions{
			Path: dir,
		})
		common.ExitOnErr(t, g.Options().Validate())

		_, err := g.Clone()
		assert.Error(t, err)
	})
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
		o := &gogit.GitOptions{Path: "dummy"}
		common.ExitOnErr(t, o.Validate())
		g := gogit.NewGitWithoutRepo(o)
		got := g.Signature()
		assert.Equal(t, gogit.DefaultGitUser, got.Name)
		assert.Equal(t, gogit.DefaultGitEmail, got.Email)
	})
}

func TestGit_Head(t *testing.T) {
	repo, dir := initRepo(t, "main")
	common.ExitOnErr(t, addFile(repo, "test", "hash"))
	want, err := commit(repo, time.Now())
	common.ExitOnErr(t, err)

	g, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	common.ExitOnErr(t, err)

	c, err := g.Head()
	common.ExitOnErr(t, err)
	assert.Equal(t, want, c.Hash)
}

func TestGit_Checkout(t *testing.T) {
	t.Run("ok: checkout to main", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		common.ExitOnErr(t, err)

		_, err = g.Checkout()
		common.ExitOnErr(t, err)

		b, err := g.Branch()
		common.ExitOnErr(t, err)
		assert.Equal(t, "main", b)
	})

	t.Run("ok: checkout to specified trunk", func(t *testing.T) {
		branchName := "test-branch"
		_, dir := initRepo(t, branchName)
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:        dir,
			TrunkBranch: branchName,
		})
		common.ExitOnErr(t, err)

		_, err = g.Checkout()
		common.ExitOnErr(t, err)

		b, err := g.Branch()
		common.ExitOnErr(t, err)
		assert.Equal(t, branchName, b)
	})

	t.Run("ok: checkout to new branch", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		common.ExitOnErr(t, err)

		_, err = g.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		common.ExitOnErr(t, err)

		b, err := g.Branch()
		common.ExitOnErr(t, err)
		assert.Equal(t, "test", b)
	})

	t.Run("err: checkout to existing branch with create opt", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		common.ExitOnErr(t, err)
		common.ExitOnErr(t, createBranch(repo, "test"))

		_, err = g.Checkout(gogit.CheckoutOptsTo("main"), gogit.CheckoutOptsCreateNew())
		assert.Error(t, err)
	})

	t.Run("err: checkout to new branch without create opt", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		common.ExitOnErr(t, err)

		_, err = g.Checkout(gogit.CheckoutOptsTo("test"))
		assert.Error(t, err)
	})
}

func TestGit_Commit(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		common.ExitOnErr(t, addFile(repo, "test", "dummy"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		common.ExitOnErr(t, err)
		_, err = g.Commit("added: test")
		assert.Nil(t, err)
	})

	t.Run("ok: other trunk branch", func(t *testing.T) {
		testTrunk := "test-branch"
		repo, dir := initRepo(t, testTrunk)
		common.ExitOnErr(t, addFile(repo, "test", "dummy"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:        dir,
			TrunkBranch: testTrunk,
		})
		common.ExitOnErr(t, err)
		h, err := g.Commit("added: test")
		assert.Nil(t, err)

		b, err := g.Branch()
		common.ExitOnErr(t, err)
		assert.Equal(t, testTrunk, b)

		c, err := g.Head()
		common.ExitOnErr(t, err)
		assert.Equal(t, h, c.Hash)
	})

	t.Run("ok: commit even when no change", func(t *testing.T) {
		_, dir := initRepo(t, "main")

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		common.ExitOnErr(t, err)
		_, err = g.Commit("no change")
		assert.Nil(t, err)
	})
}

func TestGit_Add(t *testing.T) {
	t.Run("ok: create new", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		wt, err := repo.Worktree()
		common.ExitOnErr(t, err)

		filepath := "test/added.txt"
		common.ExitOnErr(t, createFile(wt, filepath, "dummy"))
		common.ExitOnErr(t, modifyFile(wt, "README.md", "foobar"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		common.ExitOnErr(t, err)
		err = g.Add("test")
		assert.Nil(t, err)

		stmap, err := wt.Status()
		common.ExitOnErr(t, err)
		count := 0
		for fpath, st := range stmap {
			if st.Staging == extgogit.Unmodified {
				continue
			}
			count += 1
			assert.Equal(t, fpath, filepath)
			assert.Equal(t, st.Staging, extgogit.Added)
		}
		assert.Equal(t, count, 1)
	})

	t.Run("ok: modify existing", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		wt, err := repo.Worktree()
		common.ExitOnErr(t, err)

		common.ExitOnErr(t, modifyFile(wt, "README.md", "foobar"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		common.ExitOnErr(t, err)
		err = g.Add("")
		assert.Nil(t, err)

		stmap, err := wt.Status()
		common.ExitOnErr(t, err)
		count := 0
		for fpath, st := range stmap {
			if st.Staging == extgogit.Unmodified {
				continue
			}
			count += 1
			assert.Equal(t, fpath, "README.md")
			assert.Equal(t, st.Staging, extgogit.Modified)
		}
		assert.Equal(t, count, 1)
	})

	t.Run("ok: delete existing", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		wt, err := repo.Worktree()
		common.ExitOnErr(t, err)

		t.Log("#################", dir)
		common.ExitOnErr(t, deleteFile(wt, "README.md"))

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		common.ExitOnErr(t, err)
		err = g.Add("")
		assert.Nil(t, err)

		stmap, err := wt.Status()
		common.ExitOnErr(t, err)
		count := 0
		for fpath, st := range stmap {
			if st.Staging == extgogit.Unmodified {
				continue
			}
			count += 1
			assert.Equal(t, fpath, "README.md")
			assert.Equal(t, st.Staging, extgogit.Deleted)
		}
		assert.Equal(t, count, 1)
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
		common.ExitOnErr(t, err)

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dir,
			RemoteName: testRemote,
		})
		common.ExitOnErr(t, err)
		common.ExitOnErr(t, addFile(repo, "test", "push"))
		wantMsg := "git commit which should be pushed to remote"
		_, err = g.Commit(wantMsg)
		common.ExitOnErr(t, err)

		err = g.Push(gogit.PushOptBranch("main"))
		common.ExitOnErr(t, err)

		ref, err := remoteRepo.Reference(plumbing.NewBranchReferenceName("main"), false)
		common.ExitOnErr(t, err)
		c, err := repo.CommitObject(ref.Hash())
		common.ExitOnErr(t, err)

		assert.Equal(t, wantMsg, c.Message)
	})

	t.Run("err: remote not exist", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		noExistRemote := "not-exist"

		g, err := gogit.NewGit(&gogit.GitOptions{
			Path:        dir,
			RemoteName:  noExistRemote,
			TrunkBranch: "main",
		})
		common.ExitOnErr(t, err)
		common.ExitOnErr(t, addFile(repo, "test", "push"))
		_, err = g.Commit("added: test")
		common.ExitOnErr(t, err)

		err = g.Push()
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
	common.ExitOnErr(t, err)
	git, err := gogit.NewGit(&gogit.GitOptions{
		Path:       dir,
		RemoteName: testRemote,
	})
	common.ExitOnErr(t, err)

	_, err = git.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
	common.ExitOnErr(t, err)

	err = git.SetUpstream("test")
	assert.Nil(t, err)

	c, err := repo.Storer.Config()
	common.ExitOnErr(t, err)

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
	common.ExitOnErr(t, err)

	_, err = git.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
	common.ExitOnErr(t, err)

	refs, err := git.Branches()
	assert.Nil(t, err)
	assert.Len(t, refs, 3)
	for _, ref := range refs {
		assert.Contains(t, []string{"main", "test", "master"}, ref.Name().Short())
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
		common.ExitOnErr(t, err)
		gitPusher, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dirPusher,
			RemoteName: testRemote,
		})
		common.ExitOnErr(t, err)

		_, err = gitPusher.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		common.ExitOnErr(t, err)

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
		common.ExitOnErr(t, err)

		return gitPusher, gitPuller
	}

	t.Run("ok", func(t *testing.T) {
		gitPusher, gitPuller := setup(t, func(pusher *gogit.Git) {
			common.ExitOnErr(t, pusher.Push(gogit.PushOptBranch("master")))
			common.ExitOnErr(t, pusher.Push(gogit.PushOptBranch("test")))
		})

		// push branch
		common.ExitOnErr(t, addFile(gitPuller.Repo(), "test", "push"))
		wantMsg := "git commit which should be pushed to remote"
		want, err := gitPusher.Commit(wantMsg)
		common.ExitOnErr(t, err)

		common.ExitOnErr(t, gitPusher.Push(gogit.PushOptBranch("test")))

		// pull branch
		_, err = gitPuller.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		common.ExitOnErr(t, err)

		err = gitPuller.Pull()
		common.ExitOnErr(t, err)

		got, err := gitPuller.Head()
		common.ExitOnErr(t, err)
		assert.Equal(t, want.String(), got.Hash.String())
	})

	t.Run("ok: no update", func(t *testing.T) {
		gitPusher, gitPuller := setup(t, func(pusher *gogit.Git) {
			common.ExitOnErr(t, pusher.Push(gogit.PushOptBranch("master")))
			common.ExitOnErr(t, pusher.Push(gogit.PushOptBranch("test")))
		})

		head, err := gitPusher.Head()
		common.ExitOnErr(t, err)
		want := head.Hash

		// pull branch
		_, err = gitPuller.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		common.ExitOnErr(t, err)

		err = gitPuller.Pull()
		common.ExitOnErr(t, err)

		got, err := gitPuller.Head()
		common.ExitOnErr(t, err)
		assert.Equal(t, want.String(), got.Hash.String())
	})

	t.Run("err: upstream branch not exist", func(t *testing.T) {
		_, gitPuller := setup(t, func(g *gogit.Git) {
			common.ExitOnErr(t, g.Push(gogit.PushOptBranch("master")))
		})

		_, err := gitPuller.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		common.ExitOnErr(t, err)

		err = gitPuller.Pull()
		assert.Error(t, err)
	})

	t.Run("err: remote repo not exist", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		git, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dir,
			RemoteName: testRemote,
		})
		common.ExitOnErr(t, err)

		err = git.Pull()
		assert.Error(t, err)
	})
}

func TestGit_Reset(t *testing.T) {
	repo, dir := initRepo(t, "main")
	git, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	common.ExitOnErr(t, err)
	common.ExitOnErr(t, addFile(repo, "test", "hash"))

	w, err := repo.Worktree()
	common.ExitOnErr(t, err)

	st, err := w.Status()
	common.ExitOnErr(t, err)
	assert.Greater(t, len(st), 0)

	err = git.Reset(gogit.ResetOptsHard())
	assert.Nil(t, err)

	st, err = w.Status()
	common.ExitOnErr(t, err)
	assert.Equal(t, len(st), 0)
}

func TestGit_RemoveBranch(t *testing.T) {
	_, dir := initRepo(t, "main")
	git, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	common.ExitOnErr(t, err)

	_, err = git.Checkout(gogit.CheckoutOptsTo("foo"), gogit.CheckoutOptsCreateNew())
	common.ExitOnErr(t, err)
	refs, err := git.Branches()
	common.ExitOnErr(t, err)
	assert.Len(t, refs, 3)

	common.ExitOnErr(t, git.RemoveBranch(plumbing.NewBranchReferenceName("foo")))

	refs, err = git.Branches()
	common.ExitOnErr(t, err)
	assert.Len(t, refs, 2)
}

func TestGit_RemoveGoneBranches(t *testing.T) {
	repo, dir, _ := initRepoWithRemote(t, "main")
	common.ExitOnErr(t, push(repo, "main", "origin"))

	common.ExitOnErr(t, createBranch(repo, "foo"))
	common.ExitOnErr(t, push(repo, "foo", "origin"))

	common.ExitOnErr(t, createBranch(repo, "bar"))
	common.ExitOnErr(t, push(repo, "bar", "origin"))

	g, err := gogit.NewGit(&gogit.GitOptions{
		Path: dir,
	})
	common.ExitOnErr(t, err)

	refs, err := g.Branches()
	common.ExitOnErr(t, err)
	assert.Len(t, refs, 4)

	remote, err := g.Remote("origin")
	common.ExitOnErr(t, err)
	common.ExitOnErr(t, remote.RemoveBranch(plumbing.NewBranchReferenceName("bar")))

	err = g.RemoveGoneBranches()
	assert.Nil(t, err)

	refs, err = g.Branches()
	common.ExitOnErr(t, err)
	assert.Len(t, refs, 2)
}

func TestGit_Branch(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		_, dirBare := initBareRepo(t)
		repo, dir := initRepo(t, "main")
		_, err := repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{dirBare},
		})
		common.ExitOnErr(t, err)
		git, err := gogit.NewGit(&gogit.GitOptions{
			Path: dir,
		})
		common.ExitOnErr(t, err)

		remote, err := git.Remote("origin")
		assert.Nil(t, err)
		assert.Equal(t, "origin", remote.Name())
	})

	t.Run("ok: use default", func(t *testing.T) {
		_, dirBare := initBareRepo(t)
		repo, dir := initRepo(t, "main")
		_, err := repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{dirBare},
		})
		common.ExitOnErr(t, err)
		git, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dir,
			RemoteName: "origin",
		})
		common.ExitOnErr(t, err)

		remote, err := git.Remote("")
		assert.Nil(t, err)
		assert.Equal(t, "origin", remote.Name())
	})

	t.Run("err: remote not exist", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		git, err := gogit.NewGit(&gogit.GitOptions{
			Path:       dir,
			RemoteName: "origin",
		})
		common.ExitOnErr(t, err)

		_, err = git.Remote("origin")
		assert.Error(t, err)
	})
}

func TestGitBranch_BasicAuth(t *testing.T) {
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
			opt := &gogit.GitOptions{
				Token: tt.token,
			}
			remote, _, _ := setupRemoteRepo(t, opt)
			assert.Equal(t, tt.want, remote.BasicAuth())
		})
	}
}

func TestGitRemote_Branches(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		remote, git, _ := setupRemoteRepo(t, &gogit.GitOptions{})

		_, err := git.Checkout(gogit.CheckoutOptsTo("foo"), gogit.CheckoutOptsCreateNew())
		common.ExitOnErr(t, err)
		common.ExitOnErr(t, git.Push(gogit.PushOptBranch("foo")))

		_, err = git.Checkout(gogit.CheckoutOptsTo("bar"), gogit.CheckoutOptsCreateNew())
		common.ExitOnErr(t, err)
		common.ExitOnErr(t, git.Push(gogit.PushOptBranch("bar")))

		branches, err := remote.Branches()
		assert.Nil(t, err)
		assert.Len(t, branches, 2)
		for _, b := range branches {
			assert.Contains(t, []string{"refs/heads/foo", "refs/heads/bar"}, b.Name().String())
		}
	})

	t.Run("err: no branch", func(t *testing.T) {
		remote, _, _ := setupRemoteRepo(t, &gogit.GitOptions{})
		_, err := remote.Branches()
		assert.Error(t, err)
	})
}

func TestGitRemote_RemoveBranch(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		remote, git, _ := setupRemoteRepo(t, &gogit.GitOptions{})

		_, err := git.Checkout(gogit.CheckoutOptsTo("foo"), gogit.CheckoutOptsCreateNew())
		common.ExitOnErr(t, err)
		common.ExitOnErr(t, git.Push(gogit.PushOptBranch("foo")))

		_, err = git.Checkout(gogit.CheckoutOptsTo("bar"), gogit.CheckoutOptsCreateNew())
		common.ExitOnErr(t, err)
		common.ExitOnErr(t, git.Push(gogit.PushOptBranch("bar")))

		err = remote.RemoveBranch(plumbing.NewBranchReferenceName("foo"))
		assert.Nil(t, err)

		branches, err := remote.Branches()
		common.ExitOnErr(t, err)
		assert.Len(t, branches, 1)
		assert.Equal(t, "refs/heads/bar", branches[0].Name().String())
	})

	t.Run("ok: branch not found", func(t *testing.T) {
		remote, _, _ := setupRemoteRepo(t, &gogit.GitOptions{})
		err := remote.RemoveBranch(plumbing.NewBranchReferenceName("foo"))
		assert.Nil(t, err)
	})
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

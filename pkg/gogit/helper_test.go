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
	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hrk091/nwctl/pkg/gogit"
	"io/ioutil"
	"os"
	"runtime/debug"
	"testing"
	"time"
)

func exitOnErr(t *testing.T, err error) {
	if err != nil {
		t.Log(string(debug.Stack()))
		t.Fatal(err)
	}
}

func initRepo(t *testing.T, branch string) (*extgogit.Repository, string) {
	dir, err := ioutil.TempDir("", "gittest-*")
	exitOnErr(t, err)

	//dir := t.TempDir()
	repo, err := extgogit.PlainInit(dir, false)
	exitOnErr(t, err)

	exitOnErr(t, addFile(repo, "README.md", "# test"))
	_, err = commit(repo, time.Now())
	exitOnErr(t, err)
	exitOnErr(t, createBranch(repo, branch))
	return repo, dir
}

func initBareRepo(t *testing.T) (*extgogit.Repository, string) {
	dir, err := ioutil.TempDir("", "gittest-*")
	exitOnErr(t, err)
	//dir := t.TempDir()
	repo, err := extgogit.PlainInit(dir, true)
	exitOnErr(t, err)
	return repo, dir
}

func initRepoWithRemote(t *testing.T, branch string) (*extgogit.Repository, string, string) {
	_, dirBare := initBareRepo(t)
	repo, dir := initRepo(t, branch)
	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{dirBare},
	})
	exitOnErr(t, err)

	return repo, dir, dirBare
}

func setupRemoteRepo(t *testing.T, opt *gogit.GitOptions) (*gogit.GitRemote, *gogit.Git, string) {
	_, dir, _ := initRepoWithRemote(t, "main")

	opt.Path = dir
	git, err := gogit.NewGit(opt)
	exitOnErr(t, err)

	remote, err := git.Remote("origin")
	exitOnErr(t, err)
	return remote, git, dir
}

func cloneRepo(t *testing.T, opts *extgogit.CloneOptions) (*extgogit.Repository, string) {
	dir, err := ioutil.TempDir("", "gittest-*")
	exitOnErr(t, err)
	//dir := t.TempDir()
	repo, err := extgogit.PlainClone(dir, false, opts)
	exitOnErr(t, err)
	return repo, dir
}

func addFile(repo *extgogit.Repository, path, content string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	f, err := wt.Filesystem.Create(path)
	if err != nil {
		return err
	}
	if _, err = f.Write([]byte(content)); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if _, err = wt.Add(path); err != nil {
		return err
	}
	return nil
}

func commit(repo *extgogit.Repository, time time.Time) (plumbing.Hash, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return plumbing.Hash{}, err
	}
	return wt.Commit("Updated", &extgogit.CommitOptions{
		Author:    mockSignature(time),
		Committer: mockSignature(time),
	})
}

func push(repo *extgogit.Repository, branch, remote string) error {
	o := &extgogit.PushOptions{
		RemoteName: remote,
		Progress:   os.Stdout,
		RefSpecs: []config.RefSpec{
			config.RefSpec(plumbing.NewBranchReferenceName(branch) + ":" + plumbing.NewBranchReferenceName(branch)),
		},
	}
	return repo.Push(o)
}

func createBranch(repo *extgogit.Repository, branch string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	h, err := repo.Head()
	if err != nil {
		return err
	}
	return wt.Checkout(&extgogit.CheckoutOptions{
		Hash:   h.Hash(),
		Branch: plumbing.ReferenceName("refs/heads/" + branch),
		Create: true,
	})
}

func mockSignature(time time.Time) *object.Signature {
	return &object.Signature{
		Name:  "Test User",
		Email: "test@example.com",
		When:  time,
	}
}

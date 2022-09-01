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

package gogit_test

import (
	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hrk091/nwctl/pkg/gogit"
	"io/ioutil"
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

func setupRemoteRepo(t *testing.T, opt *gogit.GitOptions) (*gogit.GitRemote, *gogit.Git, string) {
	_, dirBare := initBareRepo(t)
	repo, dir := initRepo(t, "main")
	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{dirBare},
	})
	exitOnErr(t, err)

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

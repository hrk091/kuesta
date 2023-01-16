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

package kuesta_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/nttcom/kuesta/pkg/common"
	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/stretchr/testify/assert"
)

func TestWriteFileWithMkdir(t *testing.T) {
	dir := t.TempDir()
	buf := []byte("foobar")

	t.Run("ok: new dir", func(t *testing.T) {
		path := filepath.Join(dir, "foo", "bar", "tmp.txt")
		err := kuesta.WriteFileWithMkdir(path, buf)
		common.ExitOnErr(t, err)

		got, err := os.ReadFile(path)
		assert.Nil(t, err)
		assert.Equal(t, buf, got)
	})

	t.Run("ok: existing dir", func(t *testing.T) {
		err := os.MkdirAll(filepath.Join(dir, "foo", "bar"), 0o750)
		common.ExitOnErr(t, err)

		path := filepath.Join(dir, "foo", "bar", "tmp.txt")
		err = kuesta.WriteFileWithMkdir(path, buf)
		common.ExitOnErr(t, err)

		got, err := os.ReadFile(path)
		assert.Nil(t, err)
		assert.Equal(t, buf, got)
	})

	t.Run("ok: write multiple times", func(t *testing.T) {
		err := os.MkdirAll(filepath.Join(dir, "foo", "bar"), 0o750)
		common.ExitOnErr(t, err)

		path := filepath.Join(dir, "foo", "bar", "tmp.txt")
		err = kuesta.WriteFileWithMkdir(path, buf)
		common.ExitOnErr(t, err)
		err = kuesta.WriteFileWithMkdir(path, buf)
		common.ExitOnErr(t, err)

		got, err := os.ReadFile(path)
		assert.Nil(t, err)
		assert.Equal(t, buf, got)
	})
}

// test helpers

func initRepo(t *testing.T, branch string) (*extgogit.Repository, string) {
	dir, err := os.MkdirTemp("", "gittest-*")
	common.ExitOnErr(t, err)

	// dir := t.TempDir()
	repo, err := extgogit.PlainInit(dir, false)
	common.ExitOnErr(t, err)

	common.ExitOnErr(t, addFile(repo, "README.md", "# test"))
	_, err = commit(repo, time.Now())
	common.ExitOnErr(t, err)
	common.ExitOnErr(t, createBranch(repo, branch))
	return repo, dir
}

func initBareRepo(t *testing.T) (*extgogit.Repository, string) {
	dir, err := os.MkdirTemp("", "gittest-*")
	common.ExitOnErr(t, err)
	// dir := t.TempDir()
	repo, err := extgogit.PlainInit(dir, true)
	common.ExitOnErr(t, err)
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

func checkout(repo *extgogit.Repository, branch string) error {
	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	opt := &extgogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
		Keep:   true,
	}
	return w.Checkout(opt)
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
		Keep:   true,
	})
}

func mockSignature(time time.Time) *object.Signature {
	return &object.Signature{
		Name:  "Test User",
		Email: "test@example.com",
		When:  time,
	}
}

func deleteFile(repo *extgogit.Repository, path string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if _, err = wt.Remove(path); err != nil {
		return err
	}
	return nil
}

func getStatus(t *testing.T, repo *extgogit.Repository) extgogit.Status {
	w, err := repo.Worktree()
	common.ExitOnErr(t, err)
	stmap, err := w.Status()
	common.ExitOnErr(t, err)

	return stmap
}

func getBranch(t *testing.T, repo *extgogit.Repository) string {
	ref, err := repo.Head()
	common.ExitOnErr(t, err)

	return ref.Name().Short()
}

func getRemoteBranches(t *testing.T, repo *extgogit.Repository, remoteName string) []*plumbing.Reference {
	remote, err := repo.Remote(remoteName)
	common.ExitOnErr(t, err)
	branches, err := remote.List(&extgogit.ListOptions{})
	common.ExitOnErr(t, err)

	return branches
}

func getRemoteBranch(t *testing.T, repo *extgogit.Repository, remoteName, branchName string) *plumbing.Reference {
	branches := getRemoteBranches(t, repo, remoteName)
	for _, ref := range branches {
		if ref.Name().Short() == branchName {
			return ref
		}
	}
	return nil
}

func setupGitRepoWithRemote(t *testing.T, remote string) (*extgogit.Repository, string, string) {
	_, url := initBareRepo(t)

	repo, dir := initRepo(t, "main")
	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: remote,
		URLs: []string{url},
	})
	common.ExitOnErr(t, err)

	common.ExitOnErr(t, addFile(repo, "devices/device1/actual_config.cue", "{will_deleted: _}"))
	common.ExitOnErr(t, addFile(repo, "devices/device2/actual_config.cue", "{will_updated: _}"))
	_, err = commit(repo, time.Now())
	common.ExitOnErr(t, err)

	common.ExitOnErr(t, repo.CreateBranch(&config.Branch{
		Name:   "main",
		Remote: remote,
		Merge:  plumbing.NewBranchReferenceName("main"),
	}))

	common.ExitOnErr(t, push(repo, "main", remote))

	common.ExitOnErr(t, deleteFile(repo, "devices/device1/actual_config.cue"))
	common.ExitOnErr(t, addFile(repo, "devices/device2/actual_config.cue", "{updated: _}"))
	common.ExitOnErr(t, addFile(repo, "devices/device3/actual_config.cue", "{added: _}"))

	return repo, dir, url
}

func cloneRepo(t *testing.T, opts *extgogit.CloneOptions) (*extgogit.Repository, string) {
	dir, err := os.MkdirTemp("", "gittest-*")
	common.ExitOnErr(t, err)
	// dir := t.TempDir()
	repo, err := extgogit.PlainClone(dir, false, opts)
	common.ExitOnErr(t, err)
	return repo, dir
}

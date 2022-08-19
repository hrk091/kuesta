package nwctl_test

import (
	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"
)

func TestWriteFileWithMkdir(t *testing.T) {
	dir := t.TempDir()
	buf := []byte("foobar")

	t.Run("ok: new dir", func(t *testing.T) {
		path := filepath.Join(dir, "foo", "bar", "tmp.txt")
		err := nwctl.WriteFileWithMkdir(path, buf)
		ExitOnErr(t, err)

		got, err := os.ReadFile(path)
		assert.Nil(t, err)
		assert.Equal(t, buf, got)
	})

	t.Run("ok: existing dir", func(t *testing.T) {
		err := os.MkdirAll(filepath.Join(dir, "foo", "bar"), 750)
		ExitOnErr(t, err)

		path := filepath.Join(dir, "foo", "bar", "tmp.txt")
		err = nwctl.WriteFileWithMkdir(path, buf)
		ExitOnErr(t, err)

		got, err := os.ReadFile(path)
		assert.Nil(t, err)
		assert.Equal(t, buf, got)
	})

	t.Run("ok: write multiple times", func(t *testing.T) {
		err := os.MkdirAll(filepath.Join(dir, "foo", "bar"), 750)
		ExitOnErr(t, err)

		path := filepath.Join(dir, "foo", "bar", "tmp.txt")
		err = nwctl.WriteFileWithMkdir(path, buf)
		ExitOnErr(t, err)
		err = nwctl.WriteFileWithMkdir(path, buf)
		ExitOnErr(t, err)

		got, err := os.ReadFile(path)
		assert.Nil(t, err)
		assert.Equal(t, buf, got)
	})

}

func ExitOnErr(t *testing.T, err error) {
	if err != nil {
		t.Log(string(debug.Stack()))
		t.Fatal(err)
	}
}

func initRepo(t *testing.T, branch string) (*extgogit.Repository, string) {
	dir := t.TempDir()
	repo, err := extgogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	if err := addFile(repo, "README.md", "# test"); err != nil {
		t.Fatalf("add file on init: %v", err)
	}
	if _, err := commit(repo, time.Now()); err != nil {
		t.Fatalf("commit on init: %v", err)
	}
	if err := createBranch(repo, branch); err != nil {
		t.Fatalf("create init branch: %v", err)
	}
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
	if err != nil {
		t.Fatalf("git worktree: %v", err)
	}
	stmap, err := w.Status()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	return stmap
}

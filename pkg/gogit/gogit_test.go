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
			Path:        "./",
			TrunkBranch: "main",
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
		{
			"bad: branch is empty",
			func(g *gogit.GitOptions) {
				g.TrunkBranch = ""
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
			g := gogit.NewGitWithoutRepo(gogit.GitOptions{
				Token: tt.token,
			})
			assert.Equal(t, tt.want, g.BasicAuth())
		})
	}
}

func TestGit_Signature(t *testing.T) {
	t.Run("given user/email", func(t *testing.T) {
		g := gogit.NewGitWithoutRepo(gogit.GitOptions{
			User:  "test-user",
			Email: "test-email",
		})
		got := g.Signature()
		assert.Equal(t, "test-user", got.Name)
		assert.Equal(t, "test-email", got.Email)
	})
	t.Run("default", func(t *testing.T) {
		g := gogit.NewGitWithoutRepo(gogit.GitOptions{})
		got := g.Signature()
		assert.Equal(t, gogit.DefaultGitUser, got.Name)
		assert.Equal(t, gogit.DefaultGitEmail, got.Email)
	})
}

func TestGit_Head(t *testing.T) {
	repo, dir := initRepo(t, "main")
	ExitOnErr(t, addFile(repo, "test", "hash"))
	want, err := commit(repo, time.Now())
	ExitOnErr(t, err)

	g, err := gogit.NewGit(gogit.GitOptions{
		Path:        dir,
		TrunkBranch: "main",
	})
	ExitOnErr(t, err)

	c, err := g.Head()
	ExitOnErr(t, err)
	assert.Equal(t, want, c.Hash)
}

func TestGit_Checkout(t *testing.T) {
	t.Run("ok: checkout to main", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		g, err := gogit.NewGit(gogit.GitOptions{
			Path:        dir,
			TrunkBranch: "main",
		})
		ExitOnErr(t, err)

		_, err = g.Checkout()
		ExitOnErr(t, err)

		b, err := g.Branch()
		ExitOnErr(t, err)
		assert.Equal(t, "main", b)
	})

	t.Run("ok: checkout to new branch", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		g, err := gogit.NewGit(gogit.GitOptions{
			Path:        dir,
			TrunkBranch: "main",
		})
		ExitOnErr(t, err)

		_, err = g.Checkout(gogit.CheckoutOptsTo("test"), gogit.CheckoutOptsCreateNew())
		ExitOnErr(t, err)

		b, err := g.Branch()
		ExitOnErr(t, err)
		assert.Equal(t, "test", b)
	})

	t.Run("bad: checkout to existing branch with create opt", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		g, err := gogit.NewGit(gogit.GitOptions{
			Path:        dir,
			TrunkBranch: "main",
		})
		ExitOnErr(t, err)
		ExitOnErr(t, createBranch(repo, "test"))

		_, err = g.Checkout(gogit.CheckoutOptsTo("main"), gogit.CheckoutOptsCreateNew())
		assert.Error(t, err)
	})

	t.Run("bad: checkout to new branch without create opt", func(t *testing.T) {
		_, dir := initRepo(t, "main")
		g, err := gogit.NewGit(gogit.GitOptions{
			Path:        dir,
			TrunkBranch: "main",
		})
		ExitOnErr(t, err)

		_, err = g.Checkout(gogit.CheckoutOptsTo("test"))
		assert.Error(t, err)
	})
}

func TestGit_Commit(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		repo, dir := initRepo(t, "main")
		ExitOnErr(t, addFile(repo, "test", "dummy"))

		g, err := gogit.NewGit(gogit.GitOptions{
			Path:        dir,
			TrunkBranch: "main",
		})
		ExitOnErr(t, err)
		_, err = g.Commit("added: test")
		assert.Nil(t, err)
	})

	t.Run("ok: commit even when no change", func(t *testing.T) {
		_, dir := initRepo(t, "main")

		g, err := gogit.NewGit(gogit.GitOptions{
			Path:        dir,
			TrunkBranch: "main",
		})
		ExitOnErr(t, err)
		_, err = g.Commit("no change")
		assert.Nil(t, err)
	})
}

func TestGit_Push(t *testing.T) {
	remoteRepo, url := initBareRepo(t)

	repo, dir := initRepo(t, "main")
	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{url},
	})
	ExitOnErr(t, err)

	g, err := gogit.NewGit(gogit.GitOptions{
		Path:        dir,
		TrunkBranch: "main",
	})
	ExitOnErr(t, err)
	ExitOnErr(t, addFile(repo, "test", "push"))
	wantMsg := "git commit which should be pushed to remote"
	_, err = g.Commit(wantMsg)
	ExitOnErr(t, err)

	err = g.Push("main")
	ExitOnErr(t, err)

	ref, err := remoteRepo.Reference(plumbing.NewBranchReferenceName("main"), false)
	ExitOnErr(t, err)
	c, err := repo.CommitObject(ref.Hash())
	ExitOnErr(t, err)

	assert.Equal(t, wantMsg, c.Message)
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

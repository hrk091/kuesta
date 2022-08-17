package gogit_test

import (
	extgogit "github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hrk091/nwctl/pkg/gogit"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGit_Validate(t *testing.T) {
	newValidStruct := func(t func(git *gogit.Git)) *gogit.Git {
		g := &gogit.Git{
			Path:       "./",
			MainBranch: "main",
		}
		t(g)
		return g
	}

	tests := []struct {
		name      string
		transform func(g *gogit.Git)
		wantErr   bool
	}{
		{
			"ok",
			func(g *gogit.Git) {},
			false,
		},
		{
			"bad: path is empty",
			func(g *gogit.Git) {
				g.Path = ""
			},
			true,
		},
		{
			"bad: branch is empty",
			func(g *gogit.Git) {
				g.MainBranch = ""
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
			g := gogit.Git{Token: tt.token}
			assert.Equal(t, tt.want, g.BasicAuth())
		})
	}
}

func TestGit_Checkout(t *testing.T) {
	repo, dir := initRepo(t)
	_, err := commitFile(repo, "branch", "init", time.Now())
	ExitOnErr(t, err)
	ExitOnErr(t, createBranch(repo, "main"))

	g := gogit.Git{
		Path: dir,
	}
	_, err = g.Checkout("main")
	assert.Nil(t, err)
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

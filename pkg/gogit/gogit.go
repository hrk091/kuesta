package gogit

import (
	"fmt"
	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gogithttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/pkg/errors"
	"strings"
)

type Git struct {
	Token      string
	Path       string `validate:"required"`
	MainBranch string `validate:"required"`
}

const (
	DefaultUser = "anonymous"
)

// Validate validates exposed fields according to the `validate` tag.
func (g *Git) Validate() error {
	return common.Validate(g)
}

// BasicAuth returns the go-git BasicAuth if git token is provided, otherwise nil.
func (g *Git) BasicAuth() *gogithttp.BasicAuth {
	// TODO integrate with k8s secret
	// ref: https://github.com/fluxcd/source-controller/blob/main/pkg/git/options.go
	token := g.Token
	if token == "" {
		return nil
	}

	user := DefaultUser
	var password string
	if strings.Contains(token, ":") {
		slice := strings.Split(token, ":")
		user = slice[0]
		password = slice[1]
	} else {
		password = token
	}

	return &gogithttp.BasicAuth{
		Username: user,
		Password: password,
	}
}

// Checkout switches git branch to the given one and returns git worktree.
func (g *Git) Checkout(branch string) (*extgogit.Worktree, error) {
	repo, err := extgogit.PlainOpen(g.Path)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("open git repo: %w", err))
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("get worktree: %w", err))
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("resolve head: %w", err))
	}
	if ref.Name() == plumbing.NewBranchReferenceName(branch) {
		return w, nil
	}

	if err := w.Checkout(&extgogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
		Keep:   true,
	}); err != nil {
		return nil, errors.WithStack(fmt.Errorf("checkout to %s: %w", branch, err))
	}

	return w, nil
}

// IsTrackedAndChanged returns true if git file status is neither untracked nor unmodified.
func IsTrackedAndChanged(c extgogit.StatusCode) bool {
	return c != extgogit.Untracked && c != extgogit.Unmodified
}

// IsBothWorktreeAndStagingTrackedAndChanged returns true if both stating and worktree statuses
// of the given file are tracked and changed.
func IsBothWorktreeAndStagingTrackedAndChanged(st extgogit.FileStatus) bool {
	return IsTrackedAndChanged(st.Worktree) && IsTrackedAndChanged(st.Staging)
}

// IsEitherWorktreeOrStagingTrackedAndChanged returns true if either stating or worktree status
// of the given file is tracked and changed.
func IsEitherWorktreeOrStagingTrackedAndChanged(st extgogit.FileStatus) bool {
	return IsTrackedAndChanged(st.Worktree) || IsTrackedAndChanged(st.Staging)
}

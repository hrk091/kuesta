package gogit

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hrk091/nwctl/pkg/common"
	"strings"
)

type Git struct {
	Token  string
	Path   string `validate:"required"`
	Branch string `validate:"required"`
}

const (
	DefaultUser = "anonymous"
)

func (g *Git) Validate() error {
	return common.Validate(g)
}

func (g *Git) BasicAuth() *githttp.BasicAuth {
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

	return &githttp.BasicAuth{
		Username: user,
		Password: password,
	}
}

func (g *Git) Pull(singleBranch bool) error {
	repo, err := git.PlainOpen(g.Path)
	if err != nil {
		return fmt.Errorf("open git repo: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	if err := w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(g.Branch),
	}); err != nil {
		return fmt.Errorf("checkout to %s: %w", g.Branch, err)
	}

	ref, err := repo.Head()
	if err != nil {
		return fmt.Errorf("resolve head: %w", err)
	}
	if ref.Name() != plumbing.NewBranchReferenceName(g.Branch) {
		return fmt.Errorf("head is not main: %s", ref.Name())
	}

	pullOpts := git.PullOptions{
		SingleBranch: singleBranch,
		Auth:         g.BasicAuth(),
	}
	if err := w.Pull(&pullOpts); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("pull: %w", err)
		}
	}

	return nil
}

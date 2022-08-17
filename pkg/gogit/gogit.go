package gogit

import (
	"fmt"
	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gogithttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hrk091/nwctl/pkg/common"
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

func (g *Git) Validate() error {
	return common.Validate(g)
}

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

func (g *Git) Checkout(branch string) (*extgogit.Worktree, error) {
	repo, err := extgogit.PlainOpen(g.Path)
	if err != nil {
		return nil, fmt.Errorf("open git repo: %w", err)
	}
	fmt.Printf("%+v\n", repo)
	w, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("get worktree: %w", err)
	}
	fmt.Printf("%+v\n", w)

	if err := w.Checkout(&extgogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	}); err != nil {
		return nil, fmt.Errorf("checkout to %s: %w", branch, err)
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("resolve head: %w", err)
	}
	if ref.Name() != plumbing.NewBranchReferenceName(branch) {
		return nil, fmt.Errorf("head is not %s: %s", branch, ref.Name())
	}

	return w, nil
}

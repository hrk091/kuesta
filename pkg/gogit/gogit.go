/*
 * Copyright (c) 2022. Hiroki Okui
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package gogit

import (
	"fmt"
	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	gogithttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"
)

type GitOptions struct {
	Token       string
	Path        string `validate:"required"`
	TrunkBranch string
	RemoteName  string
	User        string
	Email       string
}

const (
	DefaultAuthUser    = "anonymous"
	DefaultTrunkBranch = "main"
	DefaultRemoteName  = "origin"
	DefaultGitUser     = "nwctl"
	DefaultGitEmail    = "nwctl@example.com"
)

// Validate validates exposed fields according to the `validate` tag.
func (g *GitOptions) Validate() error {
	return common.Validate(g)
}

type Git struct {
	opts *GitOptions
	repo *extgogit.Repository
}

// NewGit creates Git with a go-git repository.
func NewGit(o *GitOptions) (*Git, error) {
	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf("validate GitOptions struct: %w", err)
	}
	g := NewGitWithoutRepo(o)
	repo, err := extgogit.PlainOpen(g.opts.Path)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("open git repo: %w", err))
	}
	g.repo = repo
	return g, nil
}

// NewGitWithoutRepo creates Git without setting up a go-git repository.
func NewGitWithoutRepo(o *GitOptions) *Git {
	return &Git{
		opts: o,
	}
}

// Repo returns containing go-git repository.
func (g *Git) Repo() *extgogit.Repository {
	return g.repo
}

// BasicAuth returns the go-git BasicAuth if git token is provided, otherwise nil.
func (g *Git) BasicAuth() *gogithttp.BasicAuth {
	// TODO integrate with k8s secret
	// ref: https://github.com/fluxcd/source-controller/blob/main/pkg/git/options.go
	token := g.opts.Token
	if token == "" {
		return nil
	}

	user := DefaultAuthUser
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

// Signature returns the go-git Signature using user given User/Email or default one.
func (g *Git) Signature() *object.Signature {
	return &object.Signature{
		Name:  common.Or(g.opts.User, DefaultGitUser),
		Email: common.Or(g.opts.Email, DefaultGitEmail),
		When:  time.Now(),
	}
}

// Branch returns the current branch name.
func (g *Git) Branch() (string, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return "", errors.WithStack(fmt.Errorf("resolve head: %w", err))
	}
	return ref.Name().Short(), nil
}

// Branches returns the all branch names at the local repo.
func (g *Git) Branches() ([]string, error) {
	var branches []string
	it, err := g.repo.Branches()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = it.ForEach(func(ref *plumbing.Reference) error {
		branches = append(branches, ref.Name().Short())
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return branches, nil
}

// SetUpstream writes branch remote setting to .git/config.
func (g *Git) SetUpstream(branch string) error {
	b := config.Branch{
		Name:   branch,
		Remote: g.opts.RemoteName,
		Merge:  plumbing.NewBranchReferenceName(branch),
	}
	if err := g.repo.CreateBranch(&b); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Head returns the object.Commit of the current repository head.
func (g *Git) Head() (*object.Commit, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("resolve head: %w", err))
	}
	h := ref.Hash()
	c, err := g.repo.CommitObject(h)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("create commit object: %w", err))
	}
	return c, nil
}

// Checkout switches git branch to the given one and returns git worktree.
func (g *Git) Checkout(opts ...CheckoutOpts) (*extgogit.Worktree, error) {
	w, err := g.repo.Worktree()
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("get worktree: %w", err))
	}

	branch := common.Or(g.opts.TrunkBranch, DefaultTrunkBranch)
	o := &extgogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
		Keep:   true,
	}
	for _, tr := range opts {
		tr(o)
	}

	ref, _ := g.repo.Head()
	if ref != nil && ref.Name() == o.Branch {
		return w, nil
	}

	if err := w.Checkout(o); err != nil {
		return nil, errors.WithStack(fmt.Errorf("checkout to %s: %w", o.Branch.Short(), err))
	}

	return w, nil
}

// CheckoutOpts enables modification of the go-git CheckoutOptions.
type CheckoutOpts func(o *extgogit.CheckoutOptions)

func CheckoutOptsCreateNew() CheckoutOpts {
	return func(o *extgogit.CheckoutOptions) {
		o.Create = true
	}
}

func CheckoutOptsTo(branch string) CheckoutOpts {
	return func(o *extgogit.CheckoutOptions) {
		o.Branch = plumbing.NewBranchReferenceName(branch)
	}
}

// Commit execute `git commit -m` with given message.
func (g *Git) Commit(msg string, opts ...CommitOpts) (plumbing.Hash, error) {
	w, err := g.repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, errors.WithStack(fmt.Errorf("get worktree: %w", err))
	}

	o := &extgogit.CommitOptions{
		Author:    g.Signature(),
		Committer: g.Signature(),
	}
	for _, tr := range opts {
		tr(o)
	}
	h, err := w.Commit(msg, o)
	if err != nil {
		return plumbing.ZeroHash, errors.WithStack(fmt.Errorf("git commit: %w", err))
	}
	return h, nil
}

// CommitOpts enables modification of the go-git CommitOptions.
type CommitOpts func(o *extgogit.CommitOptions)

// Push pushes the specified git branch to remote.
func (g *Git) Push(branch string, opts ...PushOpts) error {
	o := &extgogit.PushOptions{
		RemoteName: common.Or(g.opts.RemoteName, DefaultRemoteName),
		Progress:   os.Stdout,
		RefSpecs: []config.RefSpec{
			config.RefSpec(plumbing.NewBranchReferenceName(branch) + ":" + plumbing.NewBranchReferenceName(branch)),
		},
		Auth: g.BasicAuth(),
	}
	for _, tr := range opts {
		tr(o)
	}
	if err := g.repo.Push(o); err != nil {
		return errors.WithStack(fmt.Errorf("git push: %w", err))
	}
	return nil
}

// PushOpts enables modification of the go-git PushOptions.
type PushOpts func(o *extgogit.PushOptions)

// Pull pulls the specified git branch from remote to local.
func (g *Git) Pull(opts ...PullOpts) error {
	o := &extgogit.PullOptions{
		RemoteName:   common.Or(g.opts.RemoteName, DefaultRemoteName),
		SingleBranch: false,
		Progress:     os.Stdout,
		Auth:         g.BasicAuth(),
	}
	// NOTE explicit head resolution is needed since go-git ReferenceName default does not work.
	if ref, err := g.repo.Head(); err == nil {
		o.ReferenceName = ref.Name()
	}
	for _, tr := range opts {
		tr(o)
	}

	w, err := g.repo.Worktree()
	if err != nil {
		return errors.WithStack(fmt.Errorf("get worktree: %w", err))
	}
	if err := w.Pull(o); err != nil {
		if err != extgogit.NoErrAlreadyUpToDate {
			return errors.WithStack(fmt.Errorf("git pull : %w", err))
		}
	}
	return nil
}

// PullOpts enables modification of the go-git PullOptions.
type PullOpts func(o *extgogit.PullOptions)

// ResetOpts enables modification of the go-git ResetOptions.
type ResetOpts func(o *extgogit.ResetOptions)

func ResetOptsHard() ResetOpts {
	return func(o *extgogit.ResetOptions) {
		o.Mode = extgogit.HardReset
	}
}

// Reset runs git-reset with supplied options.
func (g *Git) Reset(opts ...ResetOpts) error {
	o := &extgogit.ResetOptions{
		Mode: extgogit.HardReset,
	}
	for _, tr := range opts {
		tr(o)
	}

	w, err := g.repo.Worktree()
	if err != nil {
		return errors.WithStack(fmt.Errorf("get worktree: %w", err))
	}
	if err := w.Reset(o); err != nil {
		return errors.WithStack(fmt.Errorf("run git-reset: %w", err))
	}
	return nil
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

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

package nwctl

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/gogit"
	"github.com/hrk091/nwctl/pkg/logger"
	"strings"
)

type GitMergeDevicesCfg struct {
	RootCfg
}

// Validate validates exposed fields according to the `validate` tag.
func (c *GitMergeDevicesCfg) Validate() error {
	return common.Validate(c)
}

// RunGitMergeDevicesCfg runs the main process of the `git merge-devices` command.
func RunGitMergeDevicesCfg(ctx context.Context, cfg *GitMergeDevicesCfg) error {
	l := logger.FromContext(ctx)
	l.Debug("git merge-devices called")

	git, err := gogit.NewGit(cfg.GitOptions())
	if err != nil {
		return fmt.Errorf("init git: %w", err)
	}

	_, err = git.Checkout()
	if err != nil {
		return fmt.Errorf("git checkout: %w", err)
	}

	if err := git.Pull(); err != nil {
		return fmt.Errorf("git pull: %w", err)
	}

	remote, err := git.Remote("")
	if err != nil {
		return fmt.Errorf("get git remote: %w", err)
	}

	branches, err := remote.Branches()
	if err != nil {
		return fmt.Errorf("list remote references: %w", err)
	}

	var merged []plumbing.ReferenceName
	for _, br := range branches {
		rn := br.Name()
		if !strings.HasPrefix(rn.String(), "refs/heads/SYNC") {
			continue
		}
		l.Infof("pulling device update branch: %s", rn.String())
		if err := git.Pull(gogit.PullOptsReference(rn)); err != nil {
			return fmt.Errorf("git pull from %s: %w", rn.String(), err)
		}
		merged = append(merged, rn)
	}

	if err := git.Push(); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	for _, rn := range merged {
		if err := git.RemoveBranch(rn); err != nil {
			return fmt.Errorf("remove local branch: %w", err)
		}
		if err := remote.RemoveBranch(rn); err != nil {
			return fmt.Errorf("remove remote branch: %w", err)
		}
	}

	return nil
}

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

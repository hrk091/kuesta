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
	extgogit "github.com/go-git/go-git/v5"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/gogit"
	"github.com/hrk091/nwctl/pkg/logger"
	"go.uber.org/multierr"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type GitCommitCfg struct {
	RootCfg

	PushToMain bool
}

// Validate validates exposed fields according to the `validate` tag.
func (c *GitCommitCfg) Validate() error {
	return common.Validate(c)
}

// RunGitCommit runs the main process of the `git commit` command.
func RunGitCommit(ctx context.Context, cfg *GitCommitCfg) error {
	l := logger.FromContext(ctx)
	out := WriterFromContext(ctx)
	l.Debug("git commit called")

	git, err := gogit.NewGit(cfg.GitOptions())
	if err != nil {
		return fmt.Errorf("setup git: %w", err)
	}

	w, err := git.Checkout()
	if err != nil {
		return fmt.Errorf("git checkout: %w", err)
	}
	stmap, err := w.Status()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if len(stmap) == 0 {
		fmt.Fprintf(out, "Skipped: There are no update.")
		return nil
	}
	if err := CheckGitIsStagedOrUnmodified(stmap); err != nil {
		return fmt.Errorf("check files are either staged or unmodified: %w", err)
	}

	t := time.Now()
	branchName := "main"
	if !cfg.PushToMain {
		branchName = fmt.Sprintf("REV-%d", t.Unix())
		if w, err = git.Checkout(gogit.CheckoutOptsTo(branchName), gogit.CheckoutOptsCreateNew()); err != nil {
			return fmt.Errorf("create new branch: %w", err)
		}
	}

	commitMsg := MakeCommitMessage(stmap)
	if _, err := git.Commit(commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	if err := git.Push(gogit.PushOptBranch(branchName)); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}

// MakeCommitMessage returns the commit message that shows the summary of service and device updates.
func MakeCommitMessage(stmap extgogit.Status) string {
	var servicesAdded []string
	var servicesModified []string
	var servicesDeleted []string
	var devicesAdded []string
	var devicesModified []string
	var devicesDeleted []string

	for path, st := range stmap {
		dir, file := filepath.Split(path)
		dirElem := strings.Split(dir, string(filepath.Separator))
		if dirElem[0] == "services" && file == "input.cue" {
			serviceName := strings.TrimRight(dir, string(filepath.Separator))
			switch st.Staging {
			case extgogit.Added:
				servicesAdded = append(servicesAdded, serviceName)
			case extgogit.Modified:
				servicesModified = append(servicesModified, serviceName)
			case extgogit.Deleted:
				servicesDeleted = append(servicesDeleted, serviceName)
			}
		}
		if dirElem[0] == "devices" && file == "config.cue" {
			deviceName := dirElem[1]
			switch st.Staging {
			case extgogit.Added:
				devicesAdded = append(devicesAdded, deviceName)
			case extgogit.Modified:
				devicesModified = append(devicesModified, deviceName)
			case extgogit.Deleted:
				devicesDeleted = append(devicesDeleted, deviceName)
			}
		}
	}
	for _, v := range [][]string{servicesAdded, servicesModified, servicesDeleted, devicesAdded, devicesModified, devicesDeleted} {
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	}

	services := append(servicesAdded, servicesDeleted...)
	services = append(services, servicesModified...)

	title := fmt.Sprintf("Updated: %s", strings.Join(services, " "))
	var bodylines []string
	bodylines = append(bodylines, "", "Services:")
	for _, s := range servicesAdded {
		bodylines = append(bodylines, fmt.Sprintf("\tadded:     %s", s))
	}
	for _, s := range servicesDeleted {
		bodylines = append(bodylines, fmt.Sprintf("\tdeleted:   %s", s))
	}
	for _, s := range servicesModified {
		bodylines = append(bodylines, fmt.Sprintf("\tmodified:  %s", s))
	}

	bodylines = append(bodylines, "", "Devices:")
	for _, d := range devicesAdded {
		bodylines = append(bodylines, fmt.Sprintf("\tadded:     %s", d))
	}
	for _, d := range devicesDeleted {
		bodylines = append(bodylines, fmt.Sprintf("\tdeleted:   %s", d))
	}
	for _, d := range devicesModified {
		bodylines = append(bodylines, fmt.Sprintf("\tmodified:  %s", d))
	}

	return title + "\n" + strings.Join(bodylines, "\n")
}

// CheckGitIsStagedOrUnmodified checks all tracked files are modified and staged, or unmodified.
func CheckGitIsStagedOrUnmodified(stmap extgogit.Status) error {
	var err error
	for path, st := range stmap {
		err = multierr.Append(err, CheckGitFileIsStagedOrUnmodified(path, *st))
	}
	if err != nil {
		return common.JoinErr("check git status:", err)
	}
	return nil
}

// CheckGitFileIsStagedOrUnmodified checks the given file status is modified and staged, or unmodified.
func CheckGitFileIsStagedOrUnmodified(path string, st extgogit.FileStatus) error {
	if st.Worktree == extgogit.UpdatedButUnmerged {
		return fmt.Errorf("changes conflicted: you have to solve it in advance: %s", path)
	}
	if gogit.IsBothWorktreeAndStagingTrackedAndChanged(st) {
		return fmt.Errorf("both worktree and staging are modified: only changes in worktree or staging is allowed: %s", path)
	}
	return nil
}

/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package kuesta

import (
	"context"
	"fmt"
	extgogit "github.com/go-git/go-git/v5"
	"github.com/nttcom/kuesta/pkg/common"
	"github.com/nttcom/kuesta/pkg/gogit"
	"github.com/nttcom/kuesta/pkg/logger"
	"go.uber.org/multierr"
	"path/filepath"
	"strings"
)

type ServiceApplyCfg struct {
	RootCfg
}

// Validate validates exposed fields according to the `validate` tag.
func (c *ServiceApplyCfg) Validate() error {
	return common.Validate(c)
}

// RunServiceApply runs the main process of the `service apply` command.
func RunServiceApply(ctx context.Context, cfg *ServiceApplyCfg) error {
	l := logger.FromContext(ctx)
	l.Debug("service apply called")

	git, err := gogit.NewGit(cfg.ConfigGitOptions())
	if err != nil {
		return fmt.Errorf("init git: %w", err)
	}

	w, err := git.Checkout()
	if err != nil {
		return fmt.Errorf("git checkout: %w", err)
	}

	stmap, err := w.Status()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if err := CheckGitStatus(stmap); err != nil {
		return fmt.Errorf("check git status: %w", err)
	}

	scPlan := NewServiceCompilePlan(stmap, cfg.ConfigRootPath)
	if scPlan.IsEmpty() {
		fmt.Printf("No services updated.\n")
		return nil
	}
	err = scPlan.Do(ctx,
		func(ctx context.Context, sp ServicePath) error {
			fmt.Printf("Delete service config: service=%s keys=%v\n", sp.Service, sp.Keys)
			if _, err := w.Remove(sp.ServiceComputedDirPath(ExcludeRoot)); err != nil {
				return fmt.Errorf("git remove: %w", err)
			}
			return nil
		},
		func(ctx context.Context, sp ServicePath) error {
			fmt.Printf("Compile service config: service=%s keys=%v\n", sp.Service, sp.Keys)
			cfg := &ServiceCompileCfg{RootCfg: cfg.RootCfg, Service: sp.Service, Keys: sp.Keys}
			if err := RunServiceCompile(ctx, cfg); err != nil {
				return fmt.Errorf("service updating: %w", err)
			}
			if _, err := w.Add(sp.ServiceComputedDirPath(ExcludeRoot)); err != nil {
				return fmt.Errorf("git add: %w", err)
			}
			return nil
		})
	if err != nil {
		return err
	}

	stmap, err = w.Status()
	if err != nil {
		return fmt.Errorf("git status %w", err)
	}
	dcPlan := NewDeviceCompositePlan(stmap, cfg.ConfigRootPath)
	if dcPlan.IsEmpty() {
		fmt.Printf("No devices updated.\n")
		return nil
	}

	err = dcPlan.Do(ctx,
		func(ctx context.Context, dp DevicePath) error {
			fmt.Printf("Update device config: device=%s\n", dp.Device)
			cfg := &DeviceCompositeCfg{RootCfg: cfg.RootCfg, Device: dp.Device}
			if err := RunDeviceComposite(ctx, cfg); err != nil {
				return fmt.Errorf("device composite: %w", err)
			}
			if _, err := w.Add(dp.DeviceConfigPath(ExcludeRoot)); err != nil {
				return fmt.Errorf("git add: %w", err)
			}
			return nil
		})
	if err != nil {
		return err
	}

	return nil
}

// CheckGitStatus checks all git tracked files are in the proper status for service apply operation.
func CheckGitStatus(stmap extgogit.Status) error {
	var err error
	for path, st := range stmap {
		err = multierr.Append(err, CheckGitFileStatus(path, *st))
	}
	if err != nil {
		return common.JoinErr("check git status:", err)
	}
	return nil
}

// CheckGitFileStatus checks the given file status is in the proper status for service apply operation.
func CheckGitFileStatus(path string, st extgogit.FileStatus) error {
	dir, file := filepath.Split(path)
	dir = strings.TrimRight(dir, string(filepath.Separator))
	if strings.HasSuffix(dir, DirComputed) {
		if gogit.IsEitherWorktreeOrStagingTrackedAndChanged(st) {
			return fmt.Errorf("changes in compilation result is not allowd, you need to reset it: %s", path)
		}
	}
	if strings.HasPrefix(dir, DirDevices) && file == FileConfigCue {
		if gogit.IsEitherWorktreeOrStagingTrackedAndChanged(st) {
			return fmt.Errorf("changes in device config is not allowd, you need to reset it: %s", path)
		}
	}
	if gogit.IsBothWorktreeAndStagingTrackedAndChanged(st) {
		return fmt.Errorf("both worktree and staging are modified, only change in worktree or staging is allowed: %s", path)
	}
	if st.Worktree == extgogit.UpdatedButUnmerged {
		return fmt.Errorf("updated but unmerged changes remain. you have to solve it in advance: %s", path)
	}
	return nil
}

type ServiceFunc func(ctx context.Context, sp ServicePath) error
type DeviceFunc func(ctx context.Context, sp DevicePath) error

type ServiceCompilePlan struct {
	update []ServicePath
	delete []ServicePath
}

// NewServiceCompilePlan creates new ServiceCompilePlan from the given git file statuses.
func NewServiceCompilePlan(stmap extgogit.Status, root string) *ServiceCompilePlan {
	plan := &ServiceCompilePlan{}

	for path, st := range stmap {
		if !gogit.IsTrackedAndChanged(st.Staging) {
			continue
		}
		service, keys, err := ParseServiceInputPath(path)
		if err != nil {
			continue
		}

		sp := ServicePath{RootDir: root, Service: service, Keys: keys}
		if st.Staging == extgogit.Deleted {
			plan.delete = append(plan.delete, sp)
		} else {
			plan.update = append(plan.update, sp)
		}
	}
	return plan
}

// Do executes given delete ServiceFunc and update ServiceFunc according to its execution plan.
func (p *ServiceCompilePlan) Do(ctx context.Context, deleteFunc ServiceFunc, updateFunc ServiceFunc) error {
	for _, sp := range p.delete {
		if err := deleteFunc(ctx, sp); err != nil {
			return err
		}
	}
	for _, sp := range p.update {
		if err := updateFunc(ctx, sp); err != nil {
			return err
		}
	}
	return nil
}

// IsEmpty returns True when there are no planned targets.
func (p *ServiceCompilePlan) IsEmpty() bool {
	return len(p.update)+len(p.delete) == 0
}

type DeviceCompositePlan struct {
	composite []DevicePath
}

// NewDeviceCompositePlan creates new DeviceCompositePlan from the given git file statuses.
func NewDeviceCompositePlan(stmap extgogit.Status, root string) *DeviceCompositePlan {
	updated := common.NewSet[DevicePath]()
	for path, st := range stmap {
		if st.Staging == extgogit.Unmodified {
			continue
		}
		device, err := ParseServiceComputedFilePath(path)
		if err != nil {
			continue
		}
		updated.Add(DevicePath{RootDir: root, Device: device})
	}
	plan := &DeviceCompositePlan{composite: updated.List()}
	return plan
}

// Do executes given composite DeviceFunc according to its execution plan.
func (p *DeviceCompositePlan) Do(ctx context.Context, compositeFunc DeviceFunc) error {
	for _, dp := range p.composite {
		if err := compositeFunc(ctx, dp); err != nil {
			return err
		}
	}
	return nil
}

// IsEmpty returns True when there are no planned targets.
func (p *DeviceCompositePlan) IsEmpty() bool {
	return len(p.composite) == 0
}

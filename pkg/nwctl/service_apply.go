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

	git := gogit.Git{
		Path:       cfg.RootPath,
		MainBranch: cfg.GitBranch,
		Token:      cfg.GitToken,
	}
	if err := git.Validate(); err != nil {
		return fmt.Errorf("validate git struct: %w", err)
	}

	w, err := git.Checkout(git.MainBranch)
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

	scPlan := NewServiceCompilePlan(stmap, cfg)
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
	dcPlan := NewDeviceCompositePlan(stmap, cfg)
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
		msg := []string{"check git status:"}
		for _, err := range multierr.Errors(err) {
			msg = append(msg, err.Error())
		}
		return fmt.Errorf("%s", strings.Join(msg, "\n "))
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

type ServiceCompilePlan struct {
	update []ServicePath
	delete []ServicePath
}

func NewServiceCompilePlan(stmap extgogit.Status, cfg *ServiceApplyCfg) *ServiceCompilePlan {
	plan := &ServiceCompilePlan{}

	for path, st := range stmap {
		if !gogit.IsTrackedAndChanged(st.Staging) {
			continue
		}
		service, keys, err := ParseServiceInputPath(path)
		if err != nil {
			continue
		}

		sp := ServicePath{RootDir: cfg.RootPath, Service: service, Keys: keys}
		if st.Staging == extgogit.Deleted {
			plan.delete = append(plan.delete, sp)
		} else {
			plan.update = append(plan.update, sp)
		}
	}
	return plan
}

type ServiceFunc func(ctx context.Context, sp ServicePath) error

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

func (p *ServiceCompilePlan) IsEmpty() bool {
	return len(p.update)+len(p.delete) == 0
}

type DeviceCompositePlan struct {
	composite []DevicePath
}

func NewDeviceCompositePlan(stmap extgogit.Status, cfg *ServiceApplyCfg) *DeviceCompositePlan {
	updated := common.NewSet[DevicePath]()
	for path, st := range stmap {
		if st.Staging == extgogit.Unmodified {
			continue
		}
		device, err := ParseServiceComputedFilePath(path)
		if err != nil {
			continue
		}
		updated.Add(DevicePath{RootDir: cfg.RootPath, Device: device})
	}
	plan := &DeviceCompositePlan{composite: updated.List()}
	return plan
}

type DeviceFunc func(ctx context.Context, sp DevicePath) error

func (p *DeviceCompositePlan) Do(ctx context.Context, compositeFunc DeviceFunc) error {
	for _, dp := range p.composite {
		if err := compositeFunc(ctx, dp); err != nil {
			return err
		}
	}
	return nil
}

func (p *DeviceCompositePlan) IsEmpty() bool {
	return len(p.composite) == 0
}

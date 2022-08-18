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

	var modified []string
	for path, st := range stmap {
		if !gogit.IsTrackedAndChanged(st.Staging) {
			continue
		}

		service, keys, err := ParseServiceInputPath(path)
		if err != nil {
			continue
		}
		sp := ServicePath{RootDir: cfg.RootPath, Service: service, Keys: keys}

		modified = append(modified, path)
		if st.Staging == extgogit.Deleted {
			fmt.Printf("Service deleted: %s\n", path)
			if _, err := w.Remove(sp.ServiceComputedDirPath(ExcludeRoot)); err != nil {
				return fmt.Errorf("git remove: %w", err)
			}
		} else {
			fmt.Printf("Service updated: %s\n", path)
			scCfg := &ServiceCompileCfg{RootCfg: cfg.RootCfg, Service: service, Keys: keys}
			if err := RunServiceCompile(ctx, scCfg); err != nil {
				return fmt.Errorf("service updating: %w", err)
			}
			if _, err := w.Add(sp.ServiceComputedDirPath(ExcludeRoot)); err != nil {
				return fmt.Errorf("git add: %w", err)
			}
		}
	}
	if len(modified) == 0 {
		fmt.Printf("No services updated.\n")
		return nil
	}

	stmap, err = w.Status()
	if err != nil {
		return fmt.Errorf("git status %w", err)
	}
	updated := common.NewSet[string]()
	for path, st := range stmap {
		if st.Staging == extgogit.Unmodified {
			continue
		}
		device, err := ParseServiceComputedFilePath(path)
		if err != nil {
			continue
		}
		updated.Add(device)
	}
	if len(updated.List()) == 0 {
		fmt.Printf("No devices updated.\n")
		return nil
	}

	for _, name := range updated.List() {
		dp := DevicePath{RootDir: cfg.RootPath, Device: name}
		dcCfg := &DeviceCompositeCfg{RootCfg: cfg.RootCfg, Device: name}
		if err := RunDeviceComposite(ctx, dcCfg); err != nil {
			return fmt.Errorf("device composite: %w", err)
		}
		if _, err := w.Add(dp.DeviceConfigPath(ExcludeRoot)); err != nil {
			return fmt.Errorf("git add: %w", err)
		}
		fmt.Printf("Config updated: device=%s\n", name)
	}
	return nil
}

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

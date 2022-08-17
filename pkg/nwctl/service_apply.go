package nwctl

import (
	"context"
	"fmt"
	"github.com/hrk091/nwctl/pkg/gogit"
	"github.com/hrk091/nwctl/pkg/logger"
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

	_, err = w.Status()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}

	return nil
}

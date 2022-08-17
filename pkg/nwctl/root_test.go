package nwctl_test

import (
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRootCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(*nwctl.RootCfg)) *nwctl.RootCfg {
		cfg := &nwctl.RootCfg{
			Verbose:   0,
			Devel:     false,
			RootPath:  "./",
			GitBranch: "main",
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *nwctl.RootCfg)
		wantError bool
	}{
		{
			"ok",
			func(cfg *nwctl.RootCfg) {},
			false,
		},
		{
			"bad: Verbose is over range",
			func(cfg *nwctl.RootCfg) {
				cfg.Verbose = 4
			},
			true,
		},
		{
			"bad: RootPath is empty",
			func(cfg *nwctl.RootCfg) {
				cfg.RootPath = ""
			},
			true,
		},
		{
			"bad: GitBranch is empty",
			func(cfg *nwctl.RootCfg) {
				cfg.GitBranch = ""
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidStruct(tt.transform)
			err := cfg.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

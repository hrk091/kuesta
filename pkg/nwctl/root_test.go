package nwctl_test

import (
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRootCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(*nwctl.RootCfg)) *nwctl.RootCfg {
		cfg := &nwctl.RootCfg{
			Verbose:  0,
			Devel:    false,
			RootPath: "./",
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
			"valid",
			func(cfg *nwctl.RootCfg) {},
			false,
		},
		{
			"invalid: verbose is over range",
			func(cfg *nwctl.RootCfg) {
				cfg.Verbose = 4
			},
			true,
		},
		{
			"invalid: rootpath is empty",
			func(cfg *nwctl.RootCfg) {
				cfg.RootPath = ""
			},
			true,
		},
	}

	for _, tt := range tests {
		cfg := newValidStruct(tt.transform)
		err := cfg.Validate()
		if tt.wantError {
			assert.Error(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

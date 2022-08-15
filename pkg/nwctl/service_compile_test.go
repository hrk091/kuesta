package nwctl_test

import (
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestServiceCompileCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(cfg *nwctl.ServiceCompileCfg)) *nwctl.ServiceCompileCfg {
		cfg := &nwctl.ServiceCompileCfg{
			RootCfg: nwctl.RootCfg{
				Verbose:  0,
				Devel:    false,
				RootPath: "./",
			},
			Service: "foo",
			Keys:    []string{"one", "two"},
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *nwctl.ServiceCompileCfg)
		wantError bool
	}{
		{
			"valid",
			func(cfg *nwctl.ServiceCompileCfg) {},
			false,
		},
		{
			"invalid: service is empty",
			func(cfg *nwctl.ServiceCompileCfg) {
				cfg.Service = ""
			},
			true,
		},
		{
			"invalid: keys length is 0",
			func(cfg *nwctl.ServiceCompileCfg) {
				cfg.Keys = nil
			},
			true,
		},
		{
			"invalid: one of keys is empty",
			func(cfg *nwctl.ServiceCompileCfg) {
				cfg.Keys = []string{"one", ""}
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

package nwctl_test

import (
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeviceCompositeCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(cfg *nwctl.DeviceCompositeCfg)) *nwctl.DeviceCompositeCfg {
		cfg := &nwctl.DeviceCompositeCfg{
			RootCfg: nwctl.RootCfg{
				RootPath: "./",
			},
			Device: "device1",
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *nwctl.DeviceCompositeCfg)
		wantError bool
	}{
		{
			"ok",
			func(cfg *nwctl.DeviceCompositeCfg) {},
			false,
		},
		{
			"bad: service is empty",
			func(cfg *nwctl.DeviceCompositeCfg) {
				cfg.Device = ""
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

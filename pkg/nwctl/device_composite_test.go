package nwctl_test

import (
	"context"
	"cuelang.org/go/cue/cuecontext"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestDeviceCompositeCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(cfg *nwctl.DeviceCompositeCfg)) *nwctl.DeviceCompositeCfg {
		cfg := &nwctl.DeviceCompositeCfg{
			RootCfg: nwctl.RootCfg{
				RootPath: "./",
				GitTrunk: "main",
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
			"bad: device is empty",
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

func TestRunDeviceComposite(t *testing.T) {
	want := []byte(`{
	Interface: {
		Ethernet1: {
			Name:        "Ethernet1" @go(,*string)
			Description: "foo"       @go(,*string)
			Enabled:     true        @go(,*bool)
			AdminStatus: 1
			OperStatus:  1
			Type:        80
			Mtu:         9000 @go(,*uint16)
			Subinterface: {} @go(,map[uint32]*Interface_Subinterface)
		}
		Ethernet2: {
			Name:        "Ethernet2" @go(,*string)
			Description: "bar"       @go(,*string)
			Enabled:     false       @go(,*bool)
			AdminStatus: 1
			OperStatus:  1
			Type:        80
			Mtu:         9000 @go(,*uint16)
			Subinterface: {} @go(,map[uint32]*Interface_Subinterface)
		}
	} @go(,map[string]*Interface)
	Vlan: {} @go(,map[uint16]*Vlan)
}
`)
	err := nwctl.RunDeviceComposite(context.Background(), &nwctl.DeviceCompositeCfg{
		RootCfg: nwctl.RootCfg{RootPath: filepath.Join("./testdata")},
		Device:  "oc01",
	})
	ExitOnErr(t, err)
	got, err := os.ReadFile(filepath.Join("./testdata", "devices", "oc01", "config.cue"))
	ExitOnErr(t, err)

	cctx := cuecontext.New()
	wantVal := cctx.CompileBytes(want)
	gotVal := cctx.CompileBytes(got)
	assert.True(t, wantVal.Equals(gotVal))
}

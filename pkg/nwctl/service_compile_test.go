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

func TestServiceCompileCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(cfg *nwctl.ServiceCompileCfg)) *nwctl.ServiceCompileCfg {
		cfg := &nwctl.ServiceCompileCfg{
			RootCfg: nwctl.RootCfg{
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
			"ok",
			func(cfg *nwctl.ServiceCompileCfg) {},
			false,
		},
		{
			"bad: service is empty",
			func(cfg *nwctl.ServiceCompileCfg) {
				cfg.Service = ""
			},
			true,
		},
		{
			"bad: keys length is 0",
			func(cfg *nwctl.ServiceCompileCfg) {
				cfg.Keys = nil
			},
			true,
		},
		{
			"bad: one of keys is empty",
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

func TestRunServiceCompile(t *testing.T) {
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
	} @go(,map[string]*Interface)
	Vlan: {} @go(,map[uint16]*Vlan)
}
`)
	err := nwctl.RunServiceCompile(context.Background(), &nwctl.ServiceCompileCfg{
		RootCfg: nwctl.RootCfg{RootPath: filepath.Join("./testdata")},
		Service: "oc_interface",
		Keys:    []string{"oc01", "1"},
	})
	ExitOnErr(t, err)
	got, err := os.ReadFile(filepath.Join("./testdata", "services", "oc_interface", "oc01", "1", "computed", "oc01.cue"))
	ExitOnErr(t, err)

	cctx := cuecontext.New()
	wantVal := cctx.CompileBytes(want)
	gotVal := cctx.CompileBytes(got)
	assert.True(t, wantVal.Equals(gotVal))
}

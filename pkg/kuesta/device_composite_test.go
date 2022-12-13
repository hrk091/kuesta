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

package kuesta_test

import (
	"context"
	"cuelang.org/go/cue/cuecontext"
	"github.com/nttcom/kuesta/pkg/common"
	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestDeviceCompositeCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(cfg *kuesta.DeviceCompositeCfg)) *kuesta.DeviceCompositeCfg {
		cfg := &kuesta.DeviceCompositeCfg{
			RootCfg: kuesta.RootCfg{
				ConfigRootPath: "./",
			},
			Device: "device1",
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *kuesta.DeviceCompositeCfg)
		wantError bool
	}{
		{
			"ok",
			func(cfg *kuesta.DeviceCompositeCfg) {},
			false,
		},
		{
			"err: device is empty",
			func(cfg *kuesta.DeviceCompositeCfg) {
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
	err := kuesta.RunDeviceComposite(context.Background(), &kuesta.DeviceCompositeCfg{
		RootCfg: kuesta.RootCfg{ConfigRootPath: filepath.Join("./testdata")},
		Device:  "oc01",
	})
	common.ExitOnErr(t, err)
	got, err := os.ReadFile(filepath.Join("./testdata", "devices", "oc01", "config.cue"))
	common.ExitOnErr(t, err)

	cctx := cuecontext.New()
	wantVal := cctx.CompileBytes(want)
	gotVal := cctx.CompileBytes(got)
	assert.True(t, wantVal.Equals(gotVal))
}

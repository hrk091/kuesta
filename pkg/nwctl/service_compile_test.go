/*
 * Copyright (c) 2022. Hiroki Okui
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

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

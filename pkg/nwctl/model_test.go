/*
 Copyright 2022 NTT Communications Corporation.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package nwctl_test

import (
	"cuelang.org/go/cue/cuecontext"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

// testdata: transform
var (
	transform = []byte(`
package foo

#Input: {
	port:   uint16
	noShut: bool
	mtu:    uint16 | *9000
}

#Template: {
	input: #Input

	let _portName = "Ethernet\(input.port)"

	output: devices: {
		"device1": config: {
			Interface: "\(_portName)": {
				Name:        _portName
				Enabled:     input.noShut
				Mtu:         input.mtu
			}
		}
		"device2": config: {
			Interface: "\(_portName)": {
				Name:        _portName
				Enabled:     input.noShut
				Mtu:         input.mtu
			}
		}
	}
}`)
)

func TestReadServiceMeta(t *testing.T) {

	tests := []struct {
		name    string
		given   []byte
		want    *nwctl.ServiceMeta
		wantErr bool
	}{
		{
			"ok",
			[]byte(`{"name": "foo", "keys": ["device", "port"]}`),
			&nwctl.ServiceMeta{
				Name: "foo",
				Keys: []string{"device", "port"},
			},
			false,
		},
		{
			"err: invalid format",
			[]byte(`{"keys": ["device", "port"]`),
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "metadata.json")
			err := nwctl.WriteFileWithMkdir(path, tt.given)
			exitOnErr(t, err)
			got, err := nwctl.ReadServiceMeta(path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}

		})
	}
}

func TestNewServiceTransformer(t *testing.T) {

	tests := []struct {
		name    string
		given   []byte
		wantErr bool
	}{
		{
			"ok",
			transform,
			false,
		},
		{
			"err: invalid cue file",
			[]byte("#Input: {"),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			err := nwctl.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), tt.given)
			exitOnErr(t, err)

			cctx := cuecontext.New()
			tr, err := nwctl.NewServiceTransformer(cctx, []string{"transform.cue"}, dir)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, tr.Value())
				assert.Nil(t, tr.Value().Err())
			}
		})
	}

}

func TestServerTransformer_Apply(t *testing.T) {
	dir := t.TempDir()
	err := nwctl.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), transform)
	exitOnErr(t, err)

	cctx := cuecontext.New()
	tr, err := nwctl.NewServiceTransformer(cctx, []string{"transform.cue"}, dir)
	exitOnErr(t, err)

	t.Run("ok", func(t *testing.T) {
		in := cctx.CompileBytes(input)
		exitOnErr(t, in.Err())

		it, err := tr.Apply(in)
		exitOnErr(t, err)

		assert.True(t, it.Next())
		assert.Equal(t, "device1", it.Label())
		assert.True(t, it.Next())
		assert.Equal(t, "device2", it.Label())
		assert.False(t, it.Next())
	})

	t.Run("ok: missing optional fields", func(t *testing.T) {
		in := cctx.CompileBytes(missingOptinoal)
		exitOnErr(t, in.Err())

		_, err := tr.Apply(in)
		assert.Nil(t, err)
	})

	t.Run("bad: missing required fields", func(t *testing.T) {
		in := cctx.CompileBytes(missingRequired)
		exitOnErr(t, in.Err())

		_, err := tr.Apply(in)
		assert.Error(t, err)
	})
}

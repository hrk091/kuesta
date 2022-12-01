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
	"cuelang.org/go/cue/cuecontext"
	"github.com/nttcom/kuesta/pkg/kuesta"
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
		want    *kuesta.ServiceMeta
		wantErr bool
	}{
		{
			"ok",
			[]byte(`
kind: "foo"
keys: ["device", "port"]`),
			&kuesta.ServiceMeta{
				Kind: "foo",
				Keys: []string{"device", "port"},
			},
			false,
		},
		{
			"err: invalid format",
			[]byte(`keys: ["device", "port"`),
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "metadata.yaml")
			err := kuesta.WriteFileWithMkdir(path, tt.given)
			exitOnErr(t, err)
			got, err := kuesta.ReadServiceMeta(path)
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
			err := kuesta.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), tt.given)
			exitOnErr(t, err)

			cctx := cuecontext.New()
			tr, err := kuesta.ReadServiceTransformer(cctx, []string{"transform.cue"}, dir)
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
	err := kuesta.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), transform)
	exitOnErr(t, err)

	cctx := cuecontext.New()
	tr, err := kuesta.ReadServiceTransformer(cctx, []string{"transform.cue"}, dir)
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

	t.Run("err: missing required fields", func(t *testing.T) {
		in := cctx.CompileBytes(missingRequired)
		exitOnErr(t, in.Err())

		_, err := tr.Apply(in)
		assert.Error(t, err)
	})
}

func TestServiceTransformer_ConvertInputType(t *testing.T) {
	transformCue := []byte(`#Input: {
	strVal:   string
	intVal:   uint16
	boolVal:  bool
	floatVal: float64
	nullVal:  null
}`)
	dir := t.TempDir()
	err := kuesta.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), transformCue)
	exitOnErr(t, err)

	cctx := cuecontext.New()
	transformer, err := kuesta.ReadServiceTransformer(cctx, []string{"transform.cue"}, dir)
	exitOnErr(t, err)

	tests := []struct {
		name    string
		given   map[string]string
		want    map[string]any
		wantErr bool
	}{
		{
			"ok",
			map[string]string{
				"strVal":   "foo",
				"intVal":   "1",
				"floatVal": "2.0",
				"boolVal":  "true",
				"nullVal":  "anyValue",
			},
			map[string]any{
				"strVal":   "foo",
				"intVal":   1,
				"floatVal": 2.0,
				"boolVal":  true,
				"nullVal":  nil,
			},
			false,
		},
		{
			"err: not exist",
			map[string]string{
				"notExist": "foo",
			},
			nil,
			true,
		},
		{
			"err: cannot convert int",
			map[string]string{
				"intVal": "foo",
			},
			nil,
			true,
		},
		{
			"err: cannot convert float",
			map[string]string{
				"floatVal": "foo",
			},
			nil,
			true,
		},
		{
			"err: cannot convert bool",
			map[string]string{
				"boolVal": "foo",
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformer.ConvertInputType(tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNewDeviceFromBytes(t *testing.T) {

	tests := []struct {
		name    string
		given   []byte
		wantErr bool
	}{
		{
			"ok",
			[]byte(`config: {
	Interface: Ethernet1: {
		Name:    1
		Enabled: true
		Mtu:     9000
	}
}`),
			false,
		},
		{
			"err: invalid format",
			[]byte(`config: {`),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cctx := cuecontext.New()
			device, err := kuesta.NewDeviceFromBytes(cctx, tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, device)
			}
		})
	}
}

func TestDevice_Config(t *testing.T) {

	t.Run("ok", func(t *testing.T) {
		cctx := cuecontext.New()
		given := []byte(`
config: {
	Interface: Ethernet1: {
		Name:    1
		Enabled: true
		Mtu:     9000
	}
}`)
		want := cctx.CompileBytes([]byte(`{
	Interface: "Ethernet1": {
		Name:    1
		Enabled: true
		Mtu:     9000
	}
}`))
		exitOnErr(t, want.Err())

		device, err := kuesta.NewDeviceFromBytes(cctx, given)
		exitOnErr(t, err)
		got, err := device.Config()
		assert.Nil(t, err)
		assert.True(t, want.Equals(cctx.CompileBytes(got)))
	})

	t.Run("err: config missing", func(t *testing.T) {
		cctx := cuecontext.New()
		given := []byte(`something: {foo: "bar"}`)

		device, err := kuesta.NewDeviceFromBytes(cctx, given)
		exitOnErr(t, err)
		got, err := device.Config()
		assert.Nil(t, got)
		assert.Error(t, err)
	})

}

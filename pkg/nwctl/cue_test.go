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
	"cuelang.org/go/cue/load"
	"fmt"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

// testdata: input
var (
	input = []byte(`{
	port:   1
	noShut: true
	mtu:    9000
}`)
	invalidInput    = []byte(`{port: 1`)
	missingRequired = []byte(`{
	port:   1
    mtu: 9000
}`)
	missingOptinoal = []byte(`{
	port:   1
	noShut: true
}`)
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

// testdata: device
var (
	device = []byte(`
config: {
	Interface: Ethernet1: {
		Name:    1
		Enabled: true
		Mtu:     9000
	}
}`)
	keyMissing = []byte(`something: {foo: "bar"}`)
)

func TestNewValueFromBuf(t *testing.T) {
	cctx := cuecontext.New()
	tests := []struct {
		name    string
		given   []byte
		want    string
		wantErr bool
	}{
		{
			"ok",
			input,
			string(input),
			false,
		},
		{
			"bad: cue format",
			invalidInput,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := nwctl.NewValueFromBuf(cctx, tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want, fmt.Sprint(v))
			}
		})
	}
}

func TestNewValueFromJson(t *testing.T) {
	cctx := cuecontext.New()
	tests := []struct {
		name    string
		given   string
		want    string
		wantErr bool
	}{
		{
			"ok",
			`{"foo": "bar"}`,
			`{foo: "bar"}`,
			false,
		},
		{
			"err: invalid json",
			`{"foo": "bar"`,
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nwctl.NewValueFromJson(cctx, []byte(tt.given))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				w, err := nwctl.NewValueFromBuf(cctx, []byte(tt.want))
				exitOnErr(t, err)
				assert.Equal(t, fmt.Sprint(w), fmt.Sprint(got))
			}
		})
	}
}

func TestNewValueWithInstance(t *testing.T) {
	dir := t.TempDir()
	err := nwctl.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), transform)
	exitOnErr(t, err)

	tests := []struct {
		name    string
		files   []string
		wantErr bool
	}{
		{
			"ok",
			[]string{"transform.cue"},
			false,
		},
		{
			"bad: not exist",
			[]string{"notExist.cue"},
			true,
		},
		{
			"bad: no file given",
			[]string{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := nwctl.NewValueWithInstance(cuecontext.New(), tt.files, &load.Config{Dir: dir})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestApplyTemplate(t *testing.T) {
	dir := t.TempDir()
	err := nwctl.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), transform)
	exitOnErr(t, err)

	cctx := cuecontext.New()

	tr, err := nwctl.NewValueWithInstance(cctx, []string{"transform.cue"}, &load.Config{Dir: dir})
	exitOnErr(t, err)

	t.Run("ok", func(t *testing.T) {
		in := cctx.CompileBytes(input)
		exitOnErr(t, in.Err())

		it, err := nwctl.ApplyTransform(cctx, in, tr)
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

		_, err = nwctl.ApplyTransform(cctx, in, tr)
		assert.Nil(t, err)
	})

	t.Run("bad: missing required fields", func(t *testing.T) {
		in := cctx.CompileBytes(missingRequired)
		exitOnErr(t, in.Err())

		_, err = nwctl.ApplyTransform(cctx, in, tr)
		assert.Error(t, err)
	})
}

func TestExtractDeviceConfig(t *testing.T) {
	cctx := cuecontext.New()

	t.Run("ok", func(t *testing.T) {
		want := cctx.CompileBytes([]byte(`{
	Interface: "Ethernet1": {
		Name:    1
		Enabled: true
		Mtu:     9000
	}
}`))
		exitOnErr(t, want.Err())

		v := cctx.CompileBytes(device)
		exitOnErr(t, v.Err())

		got, err := nwctl.ExtractDeviceConfig(v)
		assert.Nil(t, err)
		assert.True(t, want.Equals(cctx.CompileBytes(got)))
	})

	t.Run("bad: config missing", func(t *testing.T) {
		v := cctx.CompileBytes(keyMissing)
		exitOnErr(t, v.Err())

		got, err := nwctl.ExtractDeviceConfig(v)
		assert.Nil(t, got)
		assert.Error(t, err)
	})

}

func TestFormatCue(t *testing.T) {
	cctx := cuecontext.New()
	want := cctx.CompileBytes([]byte(`{
	Interface: "Ethernet1": {
		Name:    1
		Enabled: true
		Mtu:     9000
	}
}`))
	exitOnErr(t, want.Err())

	got, err := nwctl.FormatCue(want)
	assert.Nil(t, err)
	assert.True(t, want.Equals(cctx.CompileBytes(got)))
}

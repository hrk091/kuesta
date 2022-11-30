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
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"fmt"
	"github.com/nttcom/kuesta/pkg/kuesta"
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

func TestNewValueFromBytes(t *testing.T) {
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
			"err: cue format",
			invalidInput,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := kuesta.NewValueFromBytes(cctx, tt.given)
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
			got, err := kuesta.NewValueFromJson(cctx, []byte(tt.given))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				w, err := kuesta.NewValueFromBytes(cctx, []byte(tt.want))
				exitOnErr(t, err)
				assert.True(t, w.Equals(got))
			}
		})
	}
}

func TestNewValueWithInstance(t *testing.T) {
	dir := t.TempDir()
	err := kuesta.WriteFileWithMkdir(filepath.Join(dir, "transform.cue"), transform)
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
			"err: not exist",
			[]string{"notExist.cue"},
			true,
		},
		{
			"err: no file given",
			[]string{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := kuesta.NewValueWithInstance(cuecontext.New(), tt.files, &load.Config{Dir: dir})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
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

	got, err := kuesta.FormatCue(want)
	assert.Nil(t, err)
	assert.True(t, want.Equals(cctx.CompileBytes(got)))
}

func TestNewAstExpr(t *testing.T) {
	given := map[string]any{
		"intVal":   1,
		"floatVal": 1.1,
		"boolVal":  false,
		"strVal":   "foo",
		"nilVal":   nil,
		"listVal":  []any{1, "foo", true},
		"map": map[string]any{
			"intVal":   1,
			"floatVal": 1.0,
			"boolVal":  true,
			"strVal":   "foo",
			"nilVal":   nil,
			"listVal":  []any{1, "foo", true},
		},
	}
	expr := kuesta.NewAstExpr(given)
	cctx := cuecontext.New()
	v := cctx.BuildExpr(expr)
	assert.Nil(t, v.Err())

	tests := []struct {
		path string
		want any
	}{
		{"intVal", 1},
		{"floatVal", 1.1},
		{"boolVal", false},
		{"strVal", `"foo"`},
		{"nilVal", "null"},
		{"listVal", `[1, "foo", true]`},
		{"map.intVal", 1},
		{"map.floatVal", 1.0},
		{"map.boolVal", true},
		{"map.strVal", `"foo"`},
		{"map.nilVal", "null"},
		{"map.listVal", `[1, "foo", true]`},
	}
	for _, tt := range tests {
		got := v.LookupPath(cue.ParsePath(tt.path))
		assert.Equal(t, fmt.Sprint(tt.want), fmt.Sprint(got))
	}
}

func TestCueKindOf(t *testing.T) {
	given := []byte(`#Input: {
	strVal:   string
	intVal:   uint16
	boolVal:  bool
	floatVal: float64
	nullVal:  null
}
`)
	cctx := cuecontext.New()
	val, err := kuesta.NewValueFromBytes(cctx, given)
	exitOnErr(t, err)

	assert.Equal(t, cue.StructKind, kuesta.CueKindOf(val, ""))
	assert.Equal(t, cue.StructKind, kuesta.CueKindOf(val, "#Input"))
	assert.Equal(t, cue.StringKind, kuesta.CueKindOf(val, "#Input.strVal"))
	assert.Equal(t, cue.IntKind, kuesta.CueKindOf(val, "#Input.intVal"))
	assert.Equal(t, cue.BoolKind, kuesta.CueKindOf(val, "#Input.boolVal"))
	assert.Equal(t, cue.NumberKind, kuesta.CueKindOf(val, "#Input.floatVal"))
	assert.Equal(t, cue.NullKind, kuesta.CueKindOf(val, "#Input.nullVal"))
}

func TestStringConverter(t *testing.T) {
	tests := []struct {
		name    string
		kind    cue.Kind
		val     string
		want    any
		wantErr bool
	}{
		{
			"ok: string",
			cue.StringKind,
			"foo",
			"foo",
			false,
		},
		{
			"ok: int",
			cue.IntKind,
			"2",
			2,
			false,
		},
		{
			"ok: float",
			cue.FloatKind,
			"1.0",
			1.0,
			false,
		},
		{
			"ok: float",
			cue.FloatKind,
			"1.1",
			1.1,
			false,
		},
		{
			"ok: number",
			cue.NumberKind,
			"1.0",
			1.0,
			false,
		},
		{
			"ok: number",
			cue.NumberKind,
			"1.1",
			1.1,
			false,
		},
		{
			"err: struct",
			cue.StructKind,
			`{"foo": "bar"}`,
			`{"foo": "bar"}`,
			true,
		},
		{
			"err: list",
			cue.ListKind,
			`["foo", "bar"]`,
			`["foo", "bar"]`,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convert, err := kuesta.NewStrConvFunc(tt.kind)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
			got, _ := convert(tt.val)
			assert.Equal(t, tt.want, got)
		})
	}
}
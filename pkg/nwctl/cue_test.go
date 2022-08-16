package nwctl_test

import (
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"fmt"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

var (
	input = []byte(`{
	device: "oc01"
	noShut: true
	port:   1
	mtu:    9000
}`)

	invalid = []byte(`{
	device: "oc01"
`)

	transform = []byte(`
package foo

#Input: {
	device: string
	port:   uint16
	noShut: bool
	mtu:    uint16 | *9000
}

#Template: {
	input: #Input

	let _portName = "Ethernet\(input.port)"

	output: devices: "\(input.device)": config: {
		Interface: "\(_portName)": {
			Name:        _portName
			Enabled:     input.noShut
			Mtu:         input.mtu
		}
	}
}`)
)

func TestNewValueFromBuf(t *testing.T) {
	tests := []struct {
		name    string
		given   []byte
		want    string
		wantErr bool
	}{
		{
			"valid",
			input,
			string(input),
			false,
		},
		{
			"invalid: cue format",
			invalid,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := nwctl.NewValueFromBuf(cuecontext.New(), tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want, fmt.Sprint(v))
			}
		})
	}
}

func TestNewValueWithInstance(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "transform.cue"), transform, 0644)

	tests := []struct {
		name    string
		files   []string
		wantErr bool
	}{
		{
			"valid",
			[]string{"transform.cue"},
			false,
		},
		{
			"invalid: not exist",
			[]string{"notExist.cue"},
			true,
		},
		{
			"invalid: no file given",
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

package nwctl_test

import (
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func newValidServicePath() *nwctl.ServicePath {
	return &nwctl.ServicePath{
		RootDir: "./tmproot",
		Service: "foo",
		Keys:    []string{"one", "two"},
	}
}

func TestServicePath_Validate(t *testing.T) {
	newValidStruct := func(t func(cfg *nwctl.ServicePath)) *nwctl.ServicePath {
		cfg := newValidServicePath()
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *nwctl.ServicePath)
		wantError bool
	}{
		{
			"valid",
			func(cfg *nwctl.ServicePath) {},
			false,
		},
		{
			"invalid: rootpath is empty",
			func(cfg *nwctl.ServicePath) {
				cfg.RootDir = ""
			},
			true,
		},
		{
			"invalid: service is empty",
			func(cfg *nwctl.ServicePath) {
				cfg.Service = ""
			},
			true,
		},
		{
			"invalid: keys length is 0",
			func(cfg *nwctl.ServicePath) {
				cfg.Keys = nil
			},
			true,
		},
		{
			"invalid: one of keys is empty",
			func(cfg *nwctl.ServicePath) {
				cfg.Keys = []string{"one", ""}
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newValidStruct(tt.transform)
			err := v.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestServicePath_RootPath(t *testing.T) {
	p := newValidServicePath()
	want := filepath.Join("path", "to", "root")
	p.RootDir = "path/to/root"
	assert.Equal(t, want, p.RootPath())
}

func TestServicePath_ServiceInputPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services/foo/one/two/input.cue", p.ServiceInputPath(nwctl.ExcludeRoot))
	assert.Equal(t, "tmproot/services/foo/one/two/input.cue", p.ServiceInputPath(nwctl.IncludeRoot))
}

func TestServicePath_ReadServiceInput(t *testing.T) {
	dir, err := os.MkdirTemp("", "repo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	t.Run("file exists", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		want := []byte("foobar")
		os.MkdirAll(filepath.Join(dir, "services", "foo", "one", "two"), 0750)
		os.WriteFile(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), want, 0644)

		r, err := p.ReadServiceInput()
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, want, r)
		}
	})

	t.Run("file not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		os.MkdirAll(filepath.Join(dir, "services", "bar", "one", "two"), 0750)

		_, err := p.ReadServiceInput()
		assert.Error(t, err)
	})

	t.Run("dir not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		p.Keys = []string{"not", "exist"}

		_, err := p.ReadServiceInput()
		assert.Error(t, err)
	})
}

func TestServicePath_ServiceTransformPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services/foo/transform.cue", p.ServiceTransformPath(nwctl.ExcludeRoot))
	assert.Equal(t, "tmproot/services/foo/transform.cue", p.ServiceTransformPath(nwctl.IncludeRoot))
}

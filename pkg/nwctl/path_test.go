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
			"ok",
			func(cfg *nwctl.ServicePath) {},
			false,
		},
		{
			"ok: service is empty",
			func(cfg *nwctl.ServicePath) {
				cfg.Service = ""
			},
			false,
		},
		{
			"ok: keys length is 0",
			func(cfg *nwctl.ServicePath) {
				cfg.Keys = nil
			},
			false,
		},
		{
			"bad: rootpath is empty",
			func(cfg *nwctl.ServicePath) {
				cfg.RootDir = ""
			},
			true,
		},
		{
			"bad: one of keys is empty",
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

func TestServicePath_ServiceDirPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services", p.ServiceDirPath(nwctl.ExcludeRoot))
	assert.Equal(t, "tmproot/services", p.ServiceDirPath(nwctl.IncludeRoot))
}

func TestServicePath_ServiceInputPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services/foo/one/two/input.cue", p.ServiceInputPath(nwctl.ExcludeRoot))
	assert.Equal(t, "tmproot/services/foo/one/two/input.cue", p.ServiceInputPath(nwctl.IncludeRoot))
}

func TestServicePath_ReadServiceInput(t *testing.T) {
	dir := t.TempDir()

	t.Run("ok: file exists", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		want := []byte("foobar")
		err := nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), want)
		ExitOnErr(t, err)

		r, err := p.ReadServiceInput()
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, want, r)
		}
	})

	t.Run("bad: file not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		err := os.MkdirAll(filepath.Join(dir, "services", "bar", "one", "two"), 0750)
		ExitOnErr(t, err)

		_, err = p.ReadServiceInput()
		assert.Error(t, err)
	})

	t.Run("bad: dir not exist", func(t *testing.T) {
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

func TestServicePath_ReadServiceTransform(t *testing.T) {
	dir := t.TempDir()

	t.Run("ok: file exists", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		want := []byte("foobar")
		err := nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "transform.cue"), want)
		ExitOnErr(t, err)

		r, err := p.ReadServiceTransform()
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, want, r)
		}
	})

	t.Run("bad: file not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		err := os.MkdirAll(filepath.Join(dir, "services", "bar"), 0750)
		ExitOnErr(t, err)

		_, err = p.ReadServiceTransform()
		assert.Error(t, err)
	})

	t.Run("bad: dir not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		p.Keys = []string{"not", "exist"}

		_, err := p.ReadServiceTransform()
		assert.Error(t, err)
	})
}

func TestServicePath_ServiceComputedDirPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services/foo/one/two/computed", p.ServiceComputedDirPath(nwctl.ExcludeRoot))
	assert.Equal(t, "tmproot/services/foo/one/two/computed", p.ServiceComputedDirPath(nwctl.IncludeRoot))
}

func TestServicePath_ServiceComputedPath(t *testing.T) {
	p := newValidServicePath()
	assert.Equal(t, "services/foo/one/two/computed/device1.cue", p.ServiceComputedPath("device1", nwctl.ExcludeRoot))
	assert.Equal(t, "tmproot/services/foo/one/two/computed/device1.cue", p.ServiceComputedPath("device1", nwctl.IncludeRoot))
}

func TestServicePath_ReadServiceComputedFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("ok: file exists", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		want := []byte("foobar")
		err := nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "computed", "device1.cue"), want)
		ExitOnErr(t, err)

		r, err := p.ReadServiceComputedFile("device1")
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, want, r)
		}
	})

	t.Run("bad: file not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		err := os.MkdirAll(filepath.Join(dir, "services", "bar", "one", "two", "computed"), 0750)
		ExitOnErr(t, err)

		_, err = p.ReadServiceTransform()
		assert.Error(t, err)
	})

	t.Run("bad: dir not exist", func(t *testing.T) {
		p := newValidServicePath()
		p.RootDir = dir
		p.Service = "bar"
		p.Keys = []string{"not", "exist"}

		_, err := p.ReadServiceTransform()
		assert.Error(t, err)
	})
}

func TestServicePath_WriteServiceComputedFile(t *testing.T) {
	dir := t.TempDir()
	buf := []byte("foobar")

	p := newValidServicePath()
	p.RootDir = dir

	err := p.WriteServiceComputedFile("device1", buf)
	ExitOnErr(t, err)

	got, err := os.ReadFile(filepath.Join(dir, "services", "foo", "one", "two", "computed", "device1.cue"))
	assert.Nil(t, err)
	assert.Equal(t, buf, got)
}

func newValidDevicePath() *nwctl.DevicePath {
	return &nwctl.DevicePath{
		RootDir: "./tmproot",
		Device:  "device1",
	}
}

func TestDevicePath_Validate(t *testing.T) {
	newValidStruct := func(t func(cfg *nwctl.DevicePath)) *nwctl.DevicePath {
		cfg := newValidDevicePath()
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *nwctl.DevicePath)
		wantError bool
	}{
		{
			"ok",
			func(cfg *nwctl.DevicePath) {},
			false,
		},
		{
			"ok: service is empty",
			func(cfg *nwctl.DevicePath) {
				cfg.Device = ""
			},
			false,
		},
		{
			"bad: rootpath is empty",
			func(cfg *nwctl.DevicePath) {
				cfg.RootDir = ""
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

func TestDevicePath_DeviceConfigPath(t *testing.T) {
	p := newValidDevicePath()
	assert.Equal(t, "devices/device1/config.cue", p.DeviceConfigPath(nwctl.ExcludeRoot))
	assert.Equal(t, "tmproot/devices/device1/config.cue", p.DeviceConfigPath(nwctl.IncludeRoot))
}

func TestDevicePath_ReadDeviceConfigFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("ok: file exists", func(t *testing.T) {
		p := newValidDevicePath()
		p.RootDir = dir
		want := []byte("foobar")
		err := nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), want)
		ExitOnErr(t, err)

		r, err := p.ReadDeviceConfigFile()
		if err != nil {
			t.Error(err)
		} else {
			assert.Equal(t, want, r)
		}
	})

	t.Run("bad: file not exist", func(t *testing.T) {
		p := newValidDevicePath()
		p.RootDir = dir
		p.Device = "device2"
		err := os.MkdirAll(filepath.Join(dir, "devices", "device2"), 0750)
		ExitOnErr(t, err)

		_, err = p.ReadDeviceConfigFile()
		assert.Error(t, err)
	})

	t.Run("bad: dir not exist", func(t *testing.T) {
		p := newValidDevicePath()
		p.Device = "notExist"
		p.RootDir = dir

		_, err := p.ReadDeviceConfigFile()
		assert.Error(t, err)
	})
}

func TestDevicePath_WriteDeviceConfigFile(t *testing.T) {
	dir := t.TempDir()
	buf := []byte("foobar")

	p := newValidDevicePath()
	p.RootDir = dir

	err := p.WriteDeviceConfigFile(buf)
	ExitOnErr(t, err)

	got, err := os.ReadFile(filepath.Join(dir, "devices", "device1", "config.cue"))
	assert.Nil(t, err)
	assert.Equal(t, buf, got)
}

func TestParseServiceInputPath(t *testing.T) {
	tests := []struct {
		name     string
		given    string
		wantSvc  string
		wantKeys []string
		wantErr  bool
	}{
		{
			"ok",
			"services/foo/one/input.cue",
			"foo",
			[]string{"one"},
			false,
		},
		{
			"ok",
			"services/foo/one/two/three/four/input.cue",
			"foo",
			[]string{"one", "two", "three", "four"},
			false,
		},
		{
			"bad: not start from services",
			"devices/device1/config.cue",
			"",
			[]string{},
			true,
		},
		{
			"bad: file is not input.cue",
			"services/foo/one/computed/device1.cue",
			"",
			[]string{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSvc, gotKeys, err := nwctl.ParseServiceInputPath(tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.wantSvc, gotSvc)
				assert.Equal(t, tt.wantKeys, gotKeys)
				assert.Nil(t, err)
			}
		})
	}
}

func TestParseServiceComputedFilePath(t *testing.T) {
	tests := []struct {
		name    string
		given   string
		want    string
		wantErr bool
	}{
		{
			"ok",
			"services/foo/one/computed/device1.cue",
			"device1",
			false,
		},
		{
			"ok",
			"services/foo/one/two/three/four/computed/device2.cue",
			"device2",
			false,
		},
		{
			"bad: not start from services",
			"devices/device1/config.cue",
			"",
			true,
		},
		{
			"bad: file is not in computed dir",
			"services/foo/one/input.cue",
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nwctl.ParseServiceComputedFilePath(tt.given)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want, got)
				assert.Nil(t, err)
			}
		})
	}
}

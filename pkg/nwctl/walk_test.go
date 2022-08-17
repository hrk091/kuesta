package nwctl_test

import (
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"io/fs"
	"path/filepath"
	"testing"
)

func TestCollectPartialDeviceConfig(t *testing.T) {
	dir := t.TempDir()
	dummy := []byte("dummy")
	ExitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "transform.cue"), dummy))
	ExitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "one", "input.cue"), dummy))
	ExitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "one", "computed", "device1.cue"), dummy))
	ExitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "one", "computed", "device2.cue"), dummy))
	ExitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "two", "input.cue"), dummy))
	ExitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "two", "computed", "device1.cue"), dummy))
	ExitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "bar", "transform.cue"), dummy))
	ExitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "bar", "one", "input.cue"), dummy))
	ExitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "bar", "one", "computed", "device1.cue"), dummy))

	t.Run("ok", func(t *testing.T) {
		files, err := nwctl.CollectPartialDeviceConfig(dir, "device1")
		assert.Nil(t, err)
		assert.Equal(t, 3, len(files))
		assert.Contains(t, files, filepath.Join(dir, "foo/one/computed/device1.cue"))
		assert.Contains(t, files, filepath.Join(dir, "foo/two/computed/device1.cue"))
		assert.Contains(t, files, filepath.Join(dir, "bar/one/computed/device1.cue"))
	})

	t.Run("ok: not found", func(t *testing.T) {
		files, err := nwctl.CollectPartialDeviceConfig(dir, "device3")
		assert.Nil(t, err)
		assert.Equal(t, 0, len(files))
	})

	t.Run("bad: directory not exist", func(t *testing.T) {
		_, err := nwctl.CollectPartialDeviceConfig("notexist", "device1")
		if assert.Error(t, err) {
			var pathError *fs.PathError
			assert.ErrorAs(t, err, &pathError)
		}
	})
}

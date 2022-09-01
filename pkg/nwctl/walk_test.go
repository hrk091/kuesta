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
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"io/fs"
	"path/filepath"
	"testing"
)

func TestCollectPartialDeviceConfig(t *testing.T) {
	dir := t.TempDir()
	dummy := []byte("dummy")
	exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "transform.cue"), dummy))
	exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "one", "input.cue"), dummy))
	exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "one", "computed", "device1.cue"), dummy))
	exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "one", "computed", "device2.cue"), dummy))
	exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "two", "input.cue"), dummy))
	exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "foo", "two", "computed", "device1.cue"), dummy))
	exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "bar", "transform.cue"), dummy))
	exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "bar", "one", "input.cue"), dummy))
	exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "bar", "one", "computed", "device1.cue"), dummy))

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

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

package nwctl

import (
	"errors"
	"fmt"
	errs "github.com/pkg/errors"
	"io/fs"
	"os"
	"path/filepath"
)

// CollectPartialDeviceConfig returns list of partial device configs for the given device.
func CollectPartialDeviceConfig(dir, device string) ([]string, error) {
	var files []string
	walkDirFunc := func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return errs.WithStack(fmt.Errorf("walk dir: %w", err))
		}
		if !info.IsDir() {
			return nil
		}
		if info.Name() != DirComputed {
			return nil
		}

		p := filepath.Join(path, fmt.Sprintf("%s.cue", device))
		if _, err := os.Stat(p); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return filepath.SkipDir
			}
			return errs.WithStack(fmt.Errorf("check if file exists: %w", err))
		}
		files = append(files, p)
		return filepath.SkipDir
	}

	if err := filepath.WalkDir(dir, walkDirFunc); err != nil {
		return nil, err
	}
	return files, nil
}

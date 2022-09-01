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
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

// WriteFileWithMkdir writes data to the named file, along with any necessary parent directories.
func WriteFileWithMkdir(path string, buf []byte) error {
	dir, _ := filepath.Split(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0750); err != nil {
			return errors.WithStack(err)
		}
	}
	if err := os.WriteFile(path, buf, 0644); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

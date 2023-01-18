/*
 Copyright (c) 2022-2023 NTT Communications Corporation

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

package testhelper

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/pkg/errors"
)

func ExitOnErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Error(string(debug.Stack()))
		t.Fatal(err)
	}
}

func MustNil(err error) {
	if err != nil {
		panic(err)
	}
}

func Chdir(t *testing.T, path string) {
	t.Helper()
	cd, err := os.Getwd()
	t.Log(cd)
	MustNil(err)
	ExitOnErr(t, os.Chdir(path))
	t.Cleanup(func() {
		ExitOnErr(t, os.Chdir(cd))
	})
}

// WriteFileWithMkdir writes data to the named file, along with any necessary parent directories.
func WriteFileWithMkdir(path string, buf []byte) error {
	dir, _ := filepath.Split(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0o750); err != nil {
			return errors.WithStack(err)
		}
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil { // nolint: gosec
		return errors.WithStack(err)
	}
	return nil
}

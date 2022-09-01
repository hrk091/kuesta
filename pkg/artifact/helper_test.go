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

package artifact_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"runtime/debug"
	"testing"
)

func exitOnErr(t *testing.T, err error) {
	if err != nil {
		t.Log(string(debug.Stack()))
		t.Fatal(err)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func mustGenTgzArchive(path, content string) (string, io.Reader) {
	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{Name: path, Mode: 0600, Size: int64(len(content))}); err != nil {
		panic(err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		panic(err)
	}
	must(tw.Close())
	must(gw.Close())

	hasher := sha256.New()
	var out bytes.Buffer
	if _, err := io.Copy(io.MultiWriter(hasher, &out), &buf); err != nil {
		panic(err)
	}
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	return checksum, &out
}

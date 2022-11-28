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

package controllers_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	nwctlv1alpha1 "github.com/nttcom/kuesta/provisioner/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
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

func newTestDataFromFixture(name string, o metav1.Object) error {
	buf, err := ioutil.ReadFile(fmt.Sprintf("./fixtures/%s.yaml", name))
	if err != nil {
		return err
	}
	// TODO GVK validation

	if err := yaml.Unmarshal(buf, o); err != nil {
		return err
	}
	return nil
}

func mustGenTgzArchiveDir(dir string) (string, io.Reader) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	walkDirFunc := func(path string, info os.FileInfo, err error) error {
		relPath := strings.TrimPrefix(path, dir+string(filepath.Separator))

		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if err := tw.WriteHeader(&tar.Header{
			Name:    relPath,
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		}); err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
		return nil
	}
	if err := filepath.Walk(dir, walkDirFunc); err != nil {
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

func hash(buf []byte) string {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, bytes.NewBuffer(buf)); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func TestNewTestDataFromFixture(t *testing.T) {

	t.Run("ok", func(t *testing.T) {
		var dr nwctlv1alpha1.DeviceRollout
		err := newTestDataFromFixture("devicerollout", &dr)
		assert.Nil(t, err)
		assert.Equal(t, dr.Name, "test-devicerollout")
		assert.Equal(t, dr.Namespace, "test-ns")
	})

	t.Run("err: file not found", func(t *testing.T) {
		var dr nwctlv1alpha1.DeviceRollout
		err := newTestDataFromFixture("not-found", &dr)
		assert.Error(t, err)
	})
}

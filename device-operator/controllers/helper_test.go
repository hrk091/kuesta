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

package controllers_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	nwctlv1alpha1 "github.com/hrk091/nwctl/provisioner/api/v1alpha1"
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

func mustGenTgzArchive(dir string) (string, io.Reader) {
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

func newGitRepoArtifact(fn func(dir string)) (string, io.Reader) {
	dir, err := ioutil.TempDir("", "git-watcher-test-*")
	must(err)
	fn(dir)
	return mustGenTgzArchive(dir)
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
		assert.Equal(t, dr.Name, "test-configrepo")
		assert.Equal(t, dr.Namespace, "test-ns")
	})

	t.Run("bad: file not found", func(t *testing.T) {
		var dr nwctlv1alpha1.DeviceRollout
		err := newTestDataFromFixture("not-found", &dr)
		assert.Error(t, err)
	})
}

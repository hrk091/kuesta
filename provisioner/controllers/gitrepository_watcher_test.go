//go:build bdd

/*
 * Copyright (c) 2022. Hiroki Okui
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package controllers_test

import (
	"context"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/hrk091/nwctl/pkg/nwctl"
	nwctlv1alpha1 "github.com/hrk091/nwctl/provisioner/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("GitRepository watcher", func() {
	ctx := context.Background()

	var testGr sourcev1.GitRepository
	Must(newTestDataFromFixture("gitrepository", &testGr))

	config1 := []byte("foo")
	config2 := []byte("bar")
	dir, err := ioutil.TempDir("", "git-watcher-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	Must(nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config1))
	Must(nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device2", "config.cue"), config2))

	checksum, buf := mustGenTgzArchiveDir(dir)
	revision := "test-revision"

	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(w, buf); err != nil {
			panic(err)
		}
	}))

	_ = logf.Log

	BeforeEach(func() {
		gr := testGr.DeepCopy()
		err := k8sClient.Create(ctx, gr)
		Expect(err).NotTo(HaveOccurred())

		gr.Status.Artifact = &sourcev1.Artifact{
			URL:      h.URL,
			Checksum: checksum,
			Revision: revision,
		}
		err = k8sClient.Status().Update(ctx, gr)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := k8sClient.DeleteAllOf(ctx, &nwctlv1alpha1.DeviceRollout{}, client.InNamespace(namespace))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should update DeviceRollout's status to running", func() {
		var dr nwctlv1alpha1.DeviceRollout
		key := client.ObjectKey{
			Namespace: testGr.Namespace,
			Name:      testGr.Name,
		}

		Eventually(func() error {
			if err := k8sClient.Get(ctx, key, &dr); err != nil {
				return err
			}
			return nil
		}, timeout, interval).Should(Succeed())

		cmap := nwctlv1alpha1.DeviceConfigMap{
			"device1": nwctlv1alpha1.DeviceConfig{
				Checksum:    hash(config1),
				GitRevision: revision,
			},
			"device2": nwctlv1alpha1.DeviceConfig{
				Checksum:    hash(config2),
				GitRevision: revision,
			},
		}
		Expect(dr.Spec.DeviceConfigMap).To(Equal(cmap))
	})

})

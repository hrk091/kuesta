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
	"context"
	"fmt"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/nttcom/kuesta/pkg/nwctl"
	nwctlv1alpha1 "github.com/nttcom/kuesta/provisioner/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("GitRepository watcher", func() {
	ctx := context.Background()

	var testGr sourcev1.GitRepository
	must(newTestDataFromFixture("gitrepository", &testGr))

	config1 := []byte("foo")
	config2 := []byte("bar")
	revision := "test-rev"

	var dir string

	BeforeEach(func() {
		var err error
		dir, err = ioutil.TempDir("", "git-watcher-test-*")
		Expect(err).NotTo(HaveOccurred())
		must(nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config1))
		must(nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device2", "config.cue"), config2))

		gr := testGr.DeepCopy()
		Expect(k8sClient.Create(ctx, gr)).NotTo(HaveOccurred())

		checksum, buf := mustGenTgzArchiveDir(dir)
		h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.Copy(w, buf); err != nil {
				panic(err)
			}
		}))

		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKeyFromObject(&testGr), gr)
		}, timeout, interval).Should(Succeed())
		gr.Status.Artifact = &sourcev1.Artifact{
			URL:      h.URL,
			Checksum: checksum,
			Revision: revision,
		}
		Eventually(func() error {
			return k8sClient.Status().Update(ctx, gr)
		}, timeout, interval).Should(Succeed())
	})

	AfterEach(func() {
		err := k8sClient.DeleteAllOf(ctx, &nwctlv1alpha1.DeviceRollout{}, client.InNamespace(namespace))
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.DeleteAllOf(ctx, &sourcev1.GitRepository{}, client.InNamespace(namespace))
		Expect(err).NotTo(HaveOccurred())
		os.RemoveAll(dir)
	})

	It("should create DeviceRollout", func() {
		var dr nwctlv1alpha1.DeviceRollout
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: testGr.Namespace, Name: testGr.Name}, &dr)
		}, timeout, interval).Should(Succeed())

		want := nwctlv1alpha1.DeviceConfigMap{
			"device1": nwctlv1alpha1.DeviceConfig{
				Checksum:    hash(config1),
				GitRevision: revision,
			},
			"device2": nwctlv1alpha1.DeviceConfig{
				Checksum:    hash(config2),
				GitRevision: revision,
			},
		}
		Expect(dr.Spec.DeviceConfigMap).To(Equal(want))
	})

	Context("when device config updated", func() {

		config1 := []byte("foo-updated")
		config2 := []byte("bar-updated")
		revision := "test-rev-updated"

		var version string

		BeforeEach(func() {
			must(nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config1))
			must(nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device2", "config.cue"), config2))

			var dr nwctlv1alpha1.DeviceRollout
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: testGr.Namespace, Name: testGr.Name}, &dr)
			}, timeout, interval).Should(Succeed())
			version = dr.ResourceVersion

			checksum, buf := mustGenTgzArchiveDir(dir)
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := io.Copy(w, buf); err != nil {
					panic(err)
				}
			}))

			var gr sourcev1.GitRepository
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(&testGr), &gr)
			}, timeout, interval).Should(Succeed())
			gr.Status.Artifact = &sourcev1.Artifact{
				URL:      h.URL,
				Checksum: checksum,
				Revision: revision,
			}
			Eventually(func() error {
				return k8sClient.Status().Update(ctx, &gr)
			}, timeout, interval).Should(Succeed())
		})

		It("should update DeviceRollout", func() {
			var dr nwctlv1alpha1.DeviceRollout
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: testGr.Namespace, Name: testGr.Name}, &dr); err != nil {
					return err
				}
				if dr.ResourceVersion == version {
					return fmt.Errorf("not updated yet")
				}
				return nil
			}, timeout, interval).Should(Succeed())

			want := nwctlv1alpha1.DeviceConfigMap{
				"device1": nwctlv1alpha1.DeviceConfig{
					Checksum:    hash(config1),
					GitRevision: revision,
				},
				"device2": nwctlv1alpha1.DeviceConfig{
					Checksum:    hash(config2),
					GitRevision: revision,
				},
			}
			Expect(dr.Spec.DeviceConfigMap).To(Equal(want))
		})

	})

})

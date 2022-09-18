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
	"context"
	"fmt"
	source "github.com/fluxcd/source-controller/api/v1beta2"
	deviceoperator "github.com/hrk091/nwctl/device-operator/api/v1alpha1"
	"github.com/hrk091/nwctl/pkg/nwctl"
	provisioner "github.com/hrk091/nwctl/provisioner/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DeviceOperator controller", func() {
	ctx := context.Background()

	config1 := []byte("foo")
	config2 := []byte("bar")
	rev1st := "rev1"

	var testOpe deviceoperator.OcDemo
	must(newTestDataFromFixture("device1.deviceoperator", &testOpe))
	var testDr provisioner.DeviceRollout
	must(newTestDataFromFixture("devicerollout", &testDr))
	var testGr source.GitRepository
	must(newTestDataFromFixture("gitrepository", &testGr))

	BeforeEach(func() {
		Expect(k8sClient.Create(ctx, testOpe.DeepCopy())).NotTo(HaveOccurred())
		Expect(k8sClient.Create(ctx, testDr.DeepCopy())).NotTo(HaveOccurred())
		Expect(k8sClient.Create(ctx, testGr.DeepCopy())).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &deviceoperator.OcDemo{}, client.InNamespace(namespace))).NotTo(HaveOccurred())
		Expect(k8sClient.DeleteAllOf(ctx, &provisioner.DeviceRollout{}, client.InNamespace(namespace))).NotTo(HaveOccurred())
		Expect(k8sClient.DeleteAllOf(ctx, &source.GitRepository{}, client.InNamespace(namespace))).NotTo(HaveOccurred())
	})

	It("should create subscriber pod", func() {
		var pod corev1.Pod
		Eventually(func() error {
			key := types.NamespacedName{
				Name:      fmt.Sprintf("subscriber-%s", testOpe.Name),
				Namespace: testOpe.Namespace,
			}
			if err := k8sClient.Get(ctx, key, &pod); err != nil {
				return err
			}
			return nil
		}, timeout, interval).Should(Succeed())
	})

	Context("when not initialized", func() {

		var dir string

		BeforeEach(func() {
			var err error
			dir, err = ioutil.TempDir("", "git-watcher-test-*")
			Expect(err).NotTo(HaveOccurred())
			must(nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config1))
			must(nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device2", "config.cue"), config2))

			checksum, buf := mustGenTgzArchiveDir(dir)
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := io.Copy(w, buf); err != nil {
					panic(err)
				}
			}))

			gr := testGr.DeepCopy()
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(&testGr), gr)
			}, timeout, interval).Should(Succeed())
			gr.Status.Artifact = &source.Artifact{
				URL:      h.URL,
				Checksum: checksum,
				Revision: rev1st,
			}
			Eventually(func() error {
				return k8sClient.Status().Update(ctx, gr)
			}, timeout, interval).Should(Succeed())
		})

		It("should initialize device resource with the specified base revision", func() {
			var ope deviceoperator.OcDemo
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testOpe), &ope); err != nil {
					return err
				}
				ope.Spec.BaseRevision = rev1st
				if err := k8sClient.Update(ctx, &ope); err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())

			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testOpe), &ope); err != nil {
					return err
				}
				if ope.Status.BaseRevision != rev1st {
					return fmt.Errorf("revision not updated yet: rev=%s", ope.Status.BaseRevision)
				}
				return nil
			}, timeout, interval).Should(Succeed())

			Expect(ope.Status.BaseRevision).To(Equal(rev1st))
			Expect(ope.Status.LastApplied).To(Equal(config1))
			Expect(ope.Status.Checksum).To(Equal(hash(config1)))
		})
	})

})

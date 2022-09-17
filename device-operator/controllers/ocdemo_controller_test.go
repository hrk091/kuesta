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
	provisioner "github.com/hrk091/nwctl/provisioner/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DeviceOperator controller", func() {
	ctx := context.Background()

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
})

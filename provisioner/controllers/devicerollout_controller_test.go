/*
Copyright 2022 Hiroki Okui.

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
	nwctlv1alpha1 "github.com/hrk091/nwctl/provisioner/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	timeout   = time.Second * 5
	interval  = time.Millisecond * 500
	namespace = "test-ns"
)

var _ = Describe("DeviceRollout controller", func() {
	ctx := context.Background()

	testDr := NewDeviceRolloutTestData("devicerollout")
	desired := nwctlv1alpha1.DeviceConfigMap{
		"device1": {Checksum: "desired", GitRevision: "desired"},
		"device2": {Checksum: "desired", GitRevision: "desired"},
	}

	BeforeEach(func() {
		err := k8sClient.Create(ctx, testDr.DeepCopy())
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			var dr nwctlv1alpha1.DeviceRollout
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
			if dr.Status.Status == "" {
				return fmt.Errorf("not updated yet")
			}
			return nil
		}, timeout, interval).Should(Succeed())
	})

	AfterEach(func() {
		err := k8sClient.DeleteAllOf(ctx, &nwctlv1alpha1.DeviceRollout{}, client.InNamespace(namespace))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should update DeviceRollout's status to running", func() {
		var dr nwctlv1alpha1.DeviceRollout
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
		Expect(dr.Status.Phase).Should(Equal(nwctlv1alpha1.RolloutPhaseHealthy))
		Expect(dr.Status.Status).Should(Equal(nwctlv1alpha1.RolloutStatusRunning))
		Expect(dr.Status.DesiredDeviceConfigMap).Should(Equal(dr.Spec.DeviceConfigMap))
		for _, v := range dr.Status.DeviceStatusMap {
			Expect(v).Should(Equal(nwctlv1alpha1.DeviceStatusRunning))
		}
	})

	Context("when devices update succeeded", func() {

		BeforeEach(func() {
			var dr nwctlv1alpha1.DeviceRollout
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
			for k, _ := range dr.Status.DeviceStatusMap {
				dr.Status.DeviceStatusMap[k] = nwctlv1alpha1.DeviceStatusCompleted
			}
			Expect(k8sClient.Status().Update(ctx, &dr)).NotTo(HaveOccurred())

			Eventually(func() error {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
				if dr.Status.Status == nwctlv1alpha1.RolloutStatusRunning {
					return fmt.Errorf("not updated yet")
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})

		It("should update DeviceRollout's status to completed", func() {
			var dr nwctlv1alpha1.DeviceRollout
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
			Expect(dr.Status.Phase).Should(Equal(nwctlv1alpha1.RolloutPhaseHealthy))
			Expect(dr.Status.Status).Should(Equal(nwctlv1alpha1.RolloutStatusCompleted))
		})

		Context("when new config provisioned", func() {

			BeforeEach(func() {
				var dr nwctlv1alpha1.DeviceRollout
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
				dr.Spec.DeviceConfigMap = desired
				Expect(k8sClient.Update(ctx, &dr)).NotTo(HaveOccurred())

				Eventually(func() error {
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					if dr.Status.Status == nwctlv1alpha1.RolloutStatusCompleted {
						return fmt.Errorf("not updated yet")
					}
					return nil
				}, timeout, interval).Should(Succeed())
			})

			It("should update DeviceRollout's status to running", func() {
				var dr nwctlv1alpha1.DeviceRollout
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
				Expect(dr.Status.Phase).Should(Equal(nwctlv1alpha1.RolloutPhaseHealthy))
				Expect(dr.Status.Status).Should(Equal(nwctlv1alpha1.RolloutStatusRunning))
				Expect(dr.Status.DesiredDeviceConfigMap).Should(Equal(desired))
				Expect(dr.Status.PrevDeviceConfigMap).Should(Equal(testDr.Spec.DeviceConfigMap))
				for _, v := range dr.Status.DeviceStatusMap {
					Expect(v).Should(Equal(nwctlv1alpha1.DeviceStatusRunning))
				}
			})

			Context("when device update failed", func() {

				BeforeEach(func() {
					var dr nwctlv1alpha1.DeviceRollout
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					for k, _ := range dr.Status.DeviceStatusMap {
						dr.Status.DeviceStatusMap[k] = nwctlv1alpha1.DeviceStatusFailed
						break
					}
					Expect(k8sClient.Status().Update(ctx, &dr)).NotTo(HaveOccurred())

					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						if dr.Status.Phase == nwctlv1alpha1.RolloutPhaseHealthy {
							return fmt.Errorf("not updated yet")
						}
						return nil
					}, timeout, interval).Should(Succeed())
				})

				It("should update DeviceRollout's phase to rollback and status to running", func() {
					var dr nwctlv1alpha1.DeviceRollout
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					Expect(dr.Status.Phase).Should(Equal(nwctlv1alpha1.RolloutPhaseRollback))
					Expect(dr.Status.Status).Should(Equal(nwctlv1alpha1.RolloutStatusRunning))
					Expect(dr.Status.DesiredDeviceConfigMap).Should(Equal(desired))
					Expect(dr.Status.PrevDeviceConfigMap).Should(Equal(testDr.Spec.DeviceConfigMap))
					for _, v := range dr.Status.DeviceStatusMap {
						Expect(v).Should(Equal(nwctlv1alpha1.DeviceStatusRunning))
					}
				})

				It("should update DeviceRollout to rollback/completed when rollback succeeded", func() {
					var dr nwctlv1alpha1.DeviceRollout
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					for k, _ := range dr.Status.DeviceStatusMap {
						dr.Status.DeviceStatusMap[k] = nwctlv1alpha1.DeviceStatusCompleted
					}
					Expect(k8sClient.Status().Update(ctx, &dr)).NotTo(HaveOccurred())
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						if dr.Status.Status == nwctlv1alpha1.RolloutStatusRunning {
							return fmt.Errorf("not updated yet")
						}
						return nil
					}, timeout, interval).Should(Succeed())

					Expect(dr.Status.Phase).Should(Equal(nwctlv1alpha1.RolloutPhaseRollback))
					Expect(dr.Status.Status).Should(Equal(nwctlv1alpha1.RolloutStatusCompleted))
				})

				It("should update DeviceRollout to rollback/failed when rollback failed", func() {
					var dr nwctlv1alpha1.DeviceRollout
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					for k, _ := range dr.Status.DeviceStatusMap {
						dr.Status.DeviceStatusMap[k] = nwctlv1alpha1.DeviceStatusFailed
						break
					}
					Expect(k8sClient.Status().Update(ctx, &dr)).NotTo(HaveOccurred())
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						if dr.Status.Status == nwctlv1alpha1.RolloutStatusRunning {
							return fmt.Errorf("not updated yet")
						}
						return nil
					}, timeout, interval).Should(Succeed())

					Expect(dr.Status.Phase).Should(Equal(nwctlv1alpha1.RolloutPhaseRollback))
					Expect(dr.Status.Status).Should(Equal(nwctlv1alpha1.RolloutStatusFailed))
				})
			})
		})
	})
})

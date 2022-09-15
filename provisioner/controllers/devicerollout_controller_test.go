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
	provisioner "github.com/hrk091/nwctl/provisioner/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DeviceRollout controller", func() {
	ctx := context.Background()

	var testDr provisioner.DeviceRollout
	must(newTestDataFromFixture("devicerollout", &testDr))
	desired := provisioner.DeviceConfigMap{
		"device1": {Checksum: "desired", GitRevision: "desired"},
		"device2": {Checksum: "desired", GitRevision: "desired"},
	}

	BeforeEach(func() {
		err := k8sClient.Create(ctx, testDr.DeepCopy())
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			var dr provisioner.DeviceRollout
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr); err != nil {
				return err
			}
			if dr.Status.Phase == "" {
				return fmt.Errorf("not updated yet")
			}
			return nil
		}, timeout, interval).Should(Succeed())
	})

	AfterEach(func() {
		err := k8sClient.DeleteAllOf(ctx, &provisioner.DeviceRollout{}, client.InNamespace(namespace))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should update DeviceRollout's status to running", func() {
		var dr provisioner.DeviceRollout
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
		Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseHealthy))
		Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusRunning))
		Expect(dr.Status.DesiredDeviceConfigMap).Should(Equal(dr.Spec.DeviceConfigMap))
		for _, v := range dr.Status.DeviceStatusMap {
			Expect(v).Should(Equal(provisioner.DeviceStatusRunning))
		}
	})

	Context("when devices update succeeded", func() {

		BeforeEach(func() {
			var dr provisioner.DeviceRollout
			Eventually(func() error {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
				for k, _ := range dr.Status.DeviceStatusMap {
					dr.Status.DeviceStatusMap[k] = provisioner.DeviceStatusCompleted
				}
				return k8sClient.Status().Update(ctx, &dr)
			}, timeout, interval).Should(Succeed())

			Eventually(func() error {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
				if dr.Status.Status == provisioner.RolloutStatusRunning {
					return fmt.Errorf("not updated yet")
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})

		It("should update DeviceRollout's status to completed", func() {
			var dr provisioner.DeviceRollout
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
			Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseHealthy))
			Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusCompleted))
		})

		Context("when new config provisioned", func() {

			BeforeEach(func() {
				var dr provisioner.DeviceRollout
				Eventually(func() error {
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					dr.Spec.DeviceConfigMap = desired
					return k8sClient.Update(ctx, &dr)
				}, timeout, interval).Should(Succeed())

				Eventually(func() error {
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					if dr.Status.Status == provisioner.RolloutStatusCompleted {
						return fmt.Errorf("not updated yet")
					}
					return nil
				}, timeout, interval).Should(Succeed())
			})

			It("should update DeviceRollout's status to running", func() {
				var dr provisioner.DeviceRollout
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
				Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseHealthy))
				Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusRunning))
				Expect(dr.Status.DesiredDeviceConfigMap).Should(Equal(desired))
				Expect(dr.Status.PrevDeviceConfigMap).Should(Equal(testDr.Spec.DeviceConfigMap))
				for _, v := range dr.Status.DeviceStatusMap {
					Expect(v).Should(Equal(provisioner.DeviceStatusRunning))
				}
			})

			Context("when device update failed", func() {

				BeforeEach(func() {
					var dr provisioner.DeviceRollout
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						for k, _ := range dr.Status.DeviceStatusMap {
							dr.Status.DeviceStatusMap[k] = provisioner.DeviceStatusFailed
							break
						}
						return k8sClient.Status().Update(ctx, &dr)
					}, timeout, interval).Should(Succeed())

					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						if dr.Status.Phase == provisioner.RolloutPhaseHealthy {
							return fmt.Errorf("not updated yet")
						}
						return nil
					}, timeout, interval).Should(Succeed())
				})

				It("should update DeviceRollout's phase to rollback and status to running", func() {
					var dr provisioner.DeviceRollout
					Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
					Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseRollback))
					Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusRunning))
					Expect(dr.Status.DesiredDeviceConfigMap).Should(Equal(desired))
					Expect(dr.Status.PrevDeviceConfigMap).Should(Equal(testDr.Spec.DeviceConfigMap))
					for _, v := range dr.Status.DeviceStatusMap {
						Expect(v).Should(Equal(provisioner.DeviceStatusRunning))
					}
				})

				It("should update DeviceRollout to rollback/completed when rollback succeeded", func() {
					var dr provisioner.DeviceRollout
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						for k, _ := range dr.Status.DeviceStatusMap {
							dr.Status.DeviceStatusMap[k] = provisioner.DeviceStatusCompleted
						}
						return k8sClient.Status().Update(ctx, &dr)
					}, timeout, interval).Should(Succeed())
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						if dr.Status.Status == provisioner.RolloutStatusRunning {
							return fmt.Errorf("not updated yet")
						}
						return nil
					}, timeout, interval).Should(Succeed())

					Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseRollback))
					Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusCompleted))
				})

				It("should update DeviceRollout to rollback/failed when rollback failed", func() {
					var dr provisioner.DeviceRollout
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						for k, _ := range dr.Status.DeviceStatusMap {
							dr.Status.DeviceStatusMap[k] = provisioner.DeviceStatusFailed
							break
						}
						return k8sClient.Status().Update(ctx, &dr)
					}, timeout, interval).Should(Succeed())
					Eventually(func() error {
						Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr)).NotTo(HaveOccurred())
						if dr.Status.Status == provisioner.RolloutStatusRunning {
							return fmt.Errorf("not updated yet")
						}
						return nil
					}, timeout, interval).Should(Succeed())

					Expect(dr.Status.Phase).Should(Equal(provisioner.RolloutPhaseRollback))
					Expect(dr.Status.Status).Should(Equal(provisioner.RolloutStatusFailed))
				})
			})
		})
	})
})

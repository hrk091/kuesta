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
	"github.com/hrk091/nwctl/pkg/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"net"

	//"github.com/hrk091/nwctl/pkg/gnmi"
	//pb "github.com/openconfig/gnmi/proto/gnmi"

	//"github.com/hrk091/nwctl/pkg/gnmi"
	"github.com/hrk091/nwctl/pkg/nwctl"
	provisioner "github.com/hrk091/nwctl/provisioner/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//pb "github.com/openconfig/gnmi/proto/gnmi"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DeviceOperator controller", func() {
	ctx := context.Background()

	config1 := []byte(`{
	Interface: {
		Ethernet1: {
			Name:        "Ethernet1"
			Description: "foo"
		}
	}
}`)
	config2 := []byte(`{
		Interface: {
			Ethernet1: {
				Name:        "Ethernet1"
				Description: "bar"
			}
		}
	}`)
	rev1st := "rev1"
	rev2nd := "rev2"

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

		BeforeEach(func() {
			checksum, buf := newGitRepoArtifact(func(dir string) {
				must(nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config1))
			})
			h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := io.Copy(w, buf)
				must(err)
			}))

			Eventually(func() error {
				var pod corev1.Pod
				key := types.NamespacedName{
					Name:      fmt.Sprintf("subscriber-%s", testOpe.Name),
					Namespace: testOpe.Namespace,
				}
				if err := k8sClient.Get(ctx, key, &pod); err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())

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

			// set base revision
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
		})

		It("should initialize device resource with the specified base revision", func() {
			var ope deviceoperator.OcDemo
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&testOpe), &ope)).NotTo(HaveOccurred())
			Expect(ope.Status.BaseRevision).To(Equal(rev1st))
			Expect(ope.Status.LastApplied).To(Equal(config1))
			Expect(ope.Status.Checksum).To(Equal(hash(config1)))
		})

		It("should change rollout status to Completed when checksum is the same", func() {
			var dr provisioner.DeviceRollout
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr); err != nil {
					return err
				}
				dr.Status.Phase = provisioner.RolloutPhaseHealthy
				dr.Status.Status = provisioner.RolloutStatusRunning
				dr.Status.SetDeviceStatus(testOpe.Name, provisioner.DeviceStatusRunning)
				if dr.Status.DesiredDeviceConfigMap == nil {
					dr.Status.DesiredDeviceConfigMap = map[string]provisioner.DeviceConfig{}
				}
				dr.Status.DesiredDeviceConfigMap[testOpe.Name] = provisioner.DeviceConfig{
					Checksum:    hash(config1),
					GitRevision: rev2nd,
				}
				if err := k8sClient.Status().Update(ctx, &dr); err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())

			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr); err != nil {
					return err
				}
				if dr.Status.GetDeviceStatus(testOpe.Name) == provisioner.DeviceStatusRunning {
					return fmt.Errorf("status not changed yet")
				}
				return nil
			}, timeout, interval).Should(Succeed())

			Expect(dr.Status.GetDeviceStatus(testOpe.Name)).To(Equal(provisioner.DeviceStatusCompleted))

		})

		Context("when device config updated", func() {

			BeforeEach(func() {
				checksum, buf := newGitRepoArtifact(func(dir string) {
					must(nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), config2))
				})
				h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, err := io.Copy(w, buf)
					must(err)
				}))

				// TODO check followings are really needed
				gr := testGr.DeepCopy()
				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKeyFromObject(&testGr), gr)
				}, timeout, interval).Should(Succeed())
				gr.Status.Artifact = &source.Artifact{
					URL:      h.URL,
					Checksum: checksum,
					Revision: rev2nd,
				}
				Eventually(func() error {
					return k8sClient.Status().Update(ctx, gr)
				}, timeout, interval).Should(Succeed())
			})

			updateConfig := func() error {
				var dr provisioner.DeviceRollout
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr); err != nil {
					return err
				}
				dr.Status.Phase = provisioner.RolloutPhaseHealthy
				dr.Status.Status = provisioner.RolloutStatusRunning
				dr.Status.SetDeviceStatus(testOpe.Name, provisioner.DeviceStatusRunning)
				if dr.Status.DesiredDeviceConfigMap == nil {
					dr.Status.DesiredDeviceConfigMap = map[string]provisioner.DeviceConfig{}
				}
				dr.Status.DesiredDeviceConfigMap[testOpe.Name] = provisioner.DeviceConfig{
					Checksum:    hash(config2),
					GitRevision: rev2nd,
				}
				fmt.Fprintf(GinkgoWriter, "device rollout status!!!, %+v", dr.Status)
				if err := k8sClient.Status().Update(ctx, &dr); err != nil {
					return err
				}
				return nil
			}

			It("should send gNMI SetRequest and change to completed when request succeeded", func() {
				setCalled := false
				m := &gnmi.GnmiMock{
					SetHandler: func(ctx context.Context, request *pb.SetRequest) (*pb.SetResponse, error) {
						setCalled = true
						return &pb.SetResponse{}, nil
					},
				}
				lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", testOpe.Spec.Address, testOpe.Spec.Port))
				must(err)
				gs := gnmi.NewServerWithListener(m, lis)
				defer gs.Stop()

				Eventually(updateConfig, timeout, interval).Should(Succeed())

				var dr provisioner.DeviceRollout
				Eventually(func() error {
					if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr); err != nil {
						return err
					}
					if dr.Status.GetDeviceStatus(testOpe.Name) == provisioner.DeviceStatusRunning {
						return fmt.Errorf("status not changed yet")
					}
					return nil
				}, timeout, interval).Should(Succeed())

				Expect(setCalled).To(BeTrue())
				Expect(dr.Status.GetDeviceStatus(testOpe.Name)).To(Equal(provisioner.DeviceStatusCompleted))
			})

			It("should send gNMI SetRequest and change to failed when request failed", func() {
				setCalled := false
				m := &gnmi.GnmiMock{
					SetHandler: func(ctx context.Context, request *pb.SetRequest) (*pb.SetResponse, error) {
						setCalled = true
						return &pb.SetResponse{}, fmt.Errorf("failed")
					},
				}
				lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", testOpe.Spec.Address, testOpe.Spec.Port))
				must(err)
				gs := gnmi.NewServerWithListener(m, lis)
				defer gs.Stop()

				Eventually(updateConfig, timeout, interval).Should(Succeed())

				var dr provisioner.DeviceRollout
				Eventually(func() error {
					if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&testDr), &dr); err != nil {
						return err
					}
					if dr.Status.GetDeviceStatus(testOpe.Name) == provisioner.DeviceStatusRunning {
						return fmt.Errorf("status not changed yet")
					}
					return nil
				}, timeout, interval).Should(Succeed())

				Expect(setCalled).To(BeTrue())
				Expect(dr.Status.GetDeviceStatus(testOpe.Name)).To(Equal(provisioner.DeviceStatusFailed))
			})

		})

	})

})

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

package controllers

import (
	"context"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	deviceoperator "github.com/hrk091/nwctl/device-operator/api/v1alpha1"
	provisioner "github.com/hrk091/nwctl/provisioner/api/v1alpha1"
)

// OcDemoReconciler reconciles a OcDemo object
type OcDemoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	impl   *DeviceReconciler
}

//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=ocdemoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=ocdemoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=ocdemoes/finalizers,verbs=update
//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=devicerollouts,verbs=get;list;watch
//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=devicerollouts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OcDemoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	return r.impl.DoReconcile(ctx, req)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OcDemoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.SetupReconciler()
	return ctrl.NewControllerManagedBy(mgr).
		For(&deviceoperator.OcDemo{}).
		Watches(
			&source.Kind{Type: &provisioner.DeviceRollout{}},
			handler.EnqueueRequestsFromMapFunc(r.impl.findObjectForDeviceRollout),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *OcDemoReconciler) SetupReconciler() {
	r.impl = &DeviceReconciler{r}
}

// DeviceReconciler reconciles a OcDemo object
type DeviceReconciler struct {
	*OcDemoReconciler
}

func (r *DeviceReconciler) getDevice(ctx context.Context, nsName types.NamespacedName) (*deviceoperator.OcDemo, error) {
	device := deviceoperator.NewDevice()
	if err := r.Get(ctx, nsName, device); err != nil {
		return nil, errors.WithStack(err)
	}
	return device, nil
}

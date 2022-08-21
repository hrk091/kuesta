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

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nwctlv1alpha1 "github.com/hrk091/nwctl/provisioner/api/v1alpha1"
)

// DeviceRolloutReconciler reconciles a DeviceRollout object
type DeviceRolloutReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=devicerollouts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=devicerollouts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=devicerollouts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DeviceRolloutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	var dr nwctlv1alpha1.DeviceRollout
	if err := r.Get(ctx, req.NamespacedName, &dr); err != nil {
		l.Error(err, "failed to fetch DeviceRollout")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// NOTE just for test
	if dr.Annotations[nwctlv1alpha1.AnnKeyResetStatus] != "" {
		if _, err := r.reconcileOnIdle(ctx, &dr); err != nil {
			return ctrl.Result{}, err
		}
		dr.Annotations[nwctlv1alpha1.AnnKeyResetStatus] = ""
		if err := r.Update(ctx, &dr); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if dr.Status.IsRunning() {
		return r.reconcileOnRunning(ctx, &dr)
	} else {
		return r.reconcileOnIdle(ctx, &dr)
	}
}

func (r *DeviceRolloutReconciler) reconcileOnRunning(ctx context.Context, dr *nwctlv1alpha1.DeviceRollout) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// check deviceStatusMap
	changed := false
	switch {
	case dr.Status.IsTxCompleted():
		dr.Status.Status = nwctlv1alpha1.RolloutStatusCompleted
		changed = true

	case dr.Status.IsTxRunning():
		// noop

	case dr.Status.IsTxFailed():
		if dr.Status.Phase == nwctlv1alpha1.RolloutPhaseHealthy {
			dr.Status.Phase = nwctlv1alpha1.RolloutPhaseRollback
			dr.Status.Status = nwctlv1alpha1.RolloutStatusRunning
			dr.Status.StartTx()
		} else {
			dr.Status.Status = nwctlv1alpha1.RolloutStatusFailed
		}
		changed = true
	}

	if changed {
		if err := r.Status().Update(ctx, dr); err != nil {
			l.Error(err, "failed to update DeviceRollout Status")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *DeviceRolloutReconciler) reconcileOnIdle(ctx context.Context, dr *nwctlv1alpha1.DeviceRollout) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	if dr.Spec.DeviceConfigMap.Equal(dr.Status.DesiredDeviceConfigMap) {
		return ctrl.Result{}, nil
	}

	// copy desired config to prev config if healthy
	if dr.Status.Phase == nwctlv1alpha1.RolloutPhaseHealthy {
		dr.Status.PrevDeviceConfigMap = dr.Status.DesiredDeviceConfigMap.DeepCopy()
	}
	// copy new config to desired config
	dr.Status.DesiredDeviceConfigMap = dr.Spec.DeviceConfigMap.DeepCopy()

	// update status
	dr.Status.Phase = nwctlv1alpha1.RolloutPhaseHealthy
	dr.Status.Status = nwctlv1alpha1.RolloutStatusRunning
	dr.Status.StartTx()

	if err := r.Status().Update(ctx, dr); err != nil {
		l.Error(err, "failed to update DeviceRollout Status")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceRolloutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nwctlv1alpha1.DeviceRollout{}).
		Complete(r)
}

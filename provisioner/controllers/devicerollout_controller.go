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
	"github.com/hrk091/nwctl/pkg/logger"
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
	l.Info("start reconciliation")

	var dr nwctlv1alpha1.DeviceRollout
	if err := r.Get(ctx, req.NamespacedName, &dr); err != nil {
		r.Error(ctx, err, "get DeviceRollout")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	changed := dr.UpdateStatus()
	if changed {
		l.Info("changed", "phase", dr.Status.Phase, "status", dr.Status.Status)
		if err := r.Status().Update(ctx, &dr); err != nil {
			r.Error(ctx, err, "update DeviceRollout")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	} else {
		l.Info("not updated")
	}
	return ctrl.Result{}, nil
}

func (r *DeviceRolloutReconciler) Error(ctx context.Context, err error, msg string, kvs ...interface{}) {
	l := log.FromContext(ctx).WithCallDepth(1)
	if st := logger.GetStackTrace(err); st != "" {
		l = l.WithValues("stacktrace", st)
	}
	l.Error(err, msg, kvs...)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceRolloutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nwctlv1alpha1.DeviceRollout{}).
		Complete(r)
}

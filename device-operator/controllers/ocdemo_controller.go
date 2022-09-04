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
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"fmt"
	"github.com/hrk091/nwctl/device-operator/pkg/model"
	"github.com/hrk091/nwctl/pkg/artifact"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/nwctl"
	gclient "github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	gnmiproto "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/prototext"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	fluxcd "github.com/fluxcd/source-controller/api/v1beta2"
	deviceoperator "github.com/hrk091/nwctl/device-operator/api/v1alpha1"
	provisioner "github.com/hrk091/nwctl/provisioner/api/v1alpha1"
)

// OcDemoReconciler reconciles a OcDemo object
type OcDemoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=ocdemoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=ocdemoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=ocdemoes/finalizers,verbs=update
//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=devicerollouts,verbs=get;list;watch
//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=devicerollouts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OcDemoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("start reconciliation")

	var device deviceoperator.OcDemo
	if err := r.Get(ctx, req.NamespacedName, &device); err != nil {
		r.Error(ctx, err, "get DeviceOperator")
		return ctrl.Result{}, errors.WithStack(client.IgnoreNotFound(err))
	}

	var dr provisioner.DeviceRollout
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: device.Namespace,
		Name:      device.Spec.RolloutRef,
	}, &dr); err != nil {
		r.Error(ctx, err, "get DeviceRollout")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if dr.Status.GetDeviceStatus(device.Name) != provisioner.DeviceStatusRunning {
		l.Info("reconcile skipped: device status is not running")
		return ctrl.Result{}, nil
	}

	next := dr.Status.ResolveNextDeviceConfig(device.Name)
	if next == nil || next.Checksum == "" {
		l.Info("device data is not stored at git repository, re-check after 10 seconds")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	if next.Checksum == device.Status.Checksum {
		l.Info(fmt.Sprintf("already provisioned: revision=%s", next.GitRevision))
		return ctrl.Result{}, nil
	}
	l.Info(fmt.Sprintf("next: revision=%s", next.GitRevision))

	var gr fluxcd.GitRepository
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: dr.Namespace,
		Name:      dr.Name,
	}, &gr); err != nil {
		r.Error(ctx, err, "get GitRepository")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	tmpDir, err := ioutil.TempDir("", gr.Name)
	if err != nil {
		r.Error(ctx, err, "create temp dir")
		return ctrl.Result{}, err
	}
	defer os.RemoveAll(tmpDir)

	if _, err = artifact.FetchArtifact(ctx, gr, tmpDir); err != nil {
		r.Error(ctx, err, "fetch artifact")
		return ctrl.Result{}, err
	}

	dp := nwctl.DevicePath{RootDir: tmpDir, Device: device.Name}
	checksum, err := dp.CheckSum()
	if err != nil {
		r.Error(ctx, err, "calc checksum")
		return ctrl.Result{}, err
	}
	if checksum != next.Checksum {
		err = fmt.Errorf("checksum is different: want=%s, got=%s", next.Checksum, checksum)
		r.Error(ctx, err, "check checksum")
		return ctrl.Result{}, err
	}

	cctx := cuecontext.New()

	newBuf, err := dp.ReadDeviceConfigFile()
	if err != nil {
		r.Error(ctx, err, "read device config")
		return ctrl.Result{}, err
	}
	newObj, err := decodeCueBuf(cctx, newBuf)
	if err != nil {
		r.Error(ctx, err, "load device config")
		return ctrl.Result{}, err
	}

	// TODO change to out-of-sync
	curBuf, err := dp.ReadActualDeviceConfigFile()
	if err != nil {
		r.Error(ctx, err, "read actual device config")
		return ctrl.Result{}, err
	}
	curObj, err := decodeCueBuf(cctx, curBuf)
	if err != nil {
		r.Error(ctx, err, "load actual device config")
		return ctrl.Result{}, err
	}

	// TODO enhance performance
	n, err := ygot.Diff(curObj, newObj, &ygot.DiffPathOpt{
		MapToSinglePath: true,
	})
	if err != nil {
		r.Error(ctx, err, "get config diff")
		return ctrl.Result{}, err
	}
	l.V(1).Info("gNMI notification", "updated", n.GetUpdate(), "deleted", n.GetDelete())

	sr := gnmiproto.SetRequest{
		Prefix: n.Prefix,
		Delete: n.Delete,
		Update: n.Update,
	}
	c, err := gnmiclient.New(ctx, gclient.Destination{
		Addrs:       []string{fmt.Sprintf("%s:%d", device.Spec.Address, device.Spec.Port)},
		Target:      "",
		Timeout:     10 * time.Second,
		Credentials: device.Spec.GnmiCredentials(),
	})
	if err != nil {
		r.Error(ctx, err, "create gNMI client")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	resp, err := c.(*gnmiclient.Client).Set(ctx, &sr)
	if err != nil {
		// TODO handle connection error
		r.Error(ctx, err, "apply Set")
		dr.Status.SetDeviceStatus(device.Name, provisioner.DeviceStatusFailed)
	} else {
		l.Info(fmt.Sprintf("succeeded SetRequest: response=%s", prototext.Format(resp)))
		dr.Status.SetDeviceStatus(device.Name, provisioner.DeviceStatusCompleted)
		device.Status.Checksum = next.Checksum
	}

	// TODO use patch to avoid optimistic locking
	if err := r.Status().Update(ctx, &dr); err != nil {
		r.Error(ctx, err, "update DeviceRollout")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// TODO get latest config and calc checksum
	// TODO change status Synced if both actual config checksum and given checksum are the same

	return ctrl.Result{}, nil
}

func (r *OcDemoReconciler) Error(ctx context.Context, err error, msg string, kvs ...interface{}) {
	l := log.FromContext(ctx).WithCallDepth(1)
	if st := common.GetStackTrace(err); st != "" {
		l = l.WithValues("stacktrace", st)
	}
	l.Error(err, msg, kvs...)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OcDemoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&deviceoperator.OcDemo{}).
		Watches(
			&source.Kind{Type: &provisioner.DeviceRollout{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectForDeviceRollout),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func decodeCueBuf(cctx *cue.Context, buf []byte) (*model.Device, error) {
	val, err := nwctl.NewValueFromBuf(cctx, buf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var o model.Device
	if err := val.Decode(&o); err != nil {
		return nil, errors.WithStack(err)
	}
	return &o, nil
}

func (r *OcDemoReconciler) findObjectForDeviceRollout(deviceRollout client.Object) []reconcile.Request {
	attachedDevices := &deviceoperator.OcDemoList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(deviceoperator.RefField, deviceRollout.GetName()),
		Namespace:     deviceRollout.GetNamespace(),
	}
	if err := r.List(context.TODO(), attachedDevices, listOps); err != nil {
		fmt.Printf("unable to list effected devices: %v\n", err)
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedDevices.Items))
	for i, v := range attachedDevices.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      v.GetName(),
				Namespace: v.GetNamespace(),
			},
		}
	}
	return requests
}

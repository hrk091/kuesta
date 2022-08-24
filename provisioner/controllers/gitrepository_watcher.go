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
	"fmt"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/hrk091/nwctl/provisioner/api/v1alpha1"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

const (
	EnvSourceHost = "SOURCE_HOST"
)

// GitRepositoryWatcher watches GitRepository objects for revision changes
type GitRepositoryWatcher struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories,verbs=get;list;watch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories/status,verbs=get
//+kubebuilder:rbac:groups=nwctl.hrk091.dev,resources=devicerollouts,verbs=get;list;create;update;patch;delete

func (r *GitRepositoryWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// get source object
	var repository sourcev1.GitRepository
	if err := r.Get(ctx, req.NamespacedName, &repository); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	l.Info("start reconciliation", "revision", repository.Status.Artifact.Revision)

	tmpDir, err := ioutil.TempDir("", repository.Name)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create temp dir, error: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	summary, err := FetchArtifact(ctx, repository, tmpDir)
	if err != nil {
		l.Error(err, "unable to fetch artifact")
		return ctrl.Result{}, err
	}
	l.Info(summary)

	dps, err := nwctl.NewDevicePathList(tmpDir)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("list devices: %w", err)
	}

	cmap := v1alpha1.DeviceConfigMap{}
	for _, dp := range dps {
		checksum, err := dp.CheckSum()
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("get checksum: %w", err)
		}
		cmap[dp.Device] = v1alpha1.DeviceConfig{
			Checksum:    checksum,
			GitRevision: repository.GetArtifact().Revision,
		}
	}

	dr := v1alpha1.DeviceRollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	}
	if _, err = ctrl.CreateOrUpdate(ctx, r.Client, &dr, func() error {
		dr.Spec.DeviceConfigMap = cmap
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GitRepositoryWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sourcev1.GitRepository{}, builder.WithPredicates(GitRepositoryRevisionChangePredicate{})).
		Complete(r)
}

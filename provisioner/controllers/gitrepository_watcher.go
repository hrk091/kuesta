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
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/hrk091/nwctl/provisioner/api/v1alpha1"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
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

	l.Info("New revision detected", "revision", repository.Status.Artifact.Revision)

	// create tmp dir
	tmpDir, err := ioutil.TempDir("", repository.Name)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create temp dir, error: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// download and extract artifact
	summary, err := v1alpha1.FetchArtifact(ctx, repository, tmpDir)
	if err != nil {
		l.Error(err, "unable to fetch artifact")
		return ctrl.Result{}, err
	}
	l.Info(summary)

	// list devices
	deviceDirPath := filepath.Join(tmpDir, "devices")
	devices, err := os.ReadDir(deviceDirPath)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list devices, error: %w", err)
	}

	dr := v1alpha1.DeviceRollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: v1alpha1.DeviceRolloutSpec{
			DeviceConfigMap: v1alpha1.DeviceConfigMap{},
		},
	}
	if _, err = ctrl.CreateOrUpdate(ctx, r.Client, &dr, func() error {
		// make deviceConfigMap
		for _, d := range devices {
			if !d.IsDir() {
				continue
			}
			dn := d.Name()
			l.Info("Processing " + dn)
			configFilePath := filepath.Join(deviceDirPath, dn, "config.cue")
			l.Info("Opening device config", "path", configFilePath)
			f, err := os.Open(configFilePath)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					l.Error(err, "unable to open file")
				}
				continue
			}
			hasher := sha256.New()
			_, err = io.Copy(hasher, f)
			if err != nil {
				l.Error(err, "unable to make hash")
				continue
			}
			next := v1alpha1.DeviceConfig{
				Checksum: fmt.Sprintf("%x", hasher.Sum(nil)),
				// TODO
				GitRevision: repository.GetArtifact().Revision,
			}
			curr, ok := dr.Spec.DeviceConfigMap[dn]
			l.Info("Diff", "curr", curr, "next", next)
			if !ok || curr.Checksum != next.Checksum {
				dr.Spec.DeviceConfigMap[dn] = next
			}
		}
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

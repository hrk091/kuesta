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
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/fluxcd/pkg/untar"
	"github.com/hrk091/nwctl/provisioner/api/v1alpha1"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
	"path/filepath"
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

	l.Info("New revision detected", "revision", repository.Status.Artifact.Revision)

	// create tmp dir
	tmpDir, err := ioutil.TempDir("", repository.Name)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create temp dir, error: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// download and extract artifact
	summary, err := FetchArtifact(ctx, repository, tmpDir)
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

func FetchArtifact(ctx context.Context, repository sourcev1.GitRepository, dir string) (string, error) {
	if repository.Status.Artifact == nil {
		return "", fmt.Errorf("respository %s does not containt an artifact", repository.Name)
	}

	url := repository.Status.Artifact.URL

	// for local run:
	// kubectl -n flux-system port-forward svc/source-controller 8080:80
	// export SOURCE_HOST=localhost:8080
	if hostname := os.Getenv(EnvSourceHost); hostname != "" {
		url = fmt.Sprintf("http://%s/gitrepository/%s/%s/latest.tar.gz", hostname, repository.Namespace, repository.Name)
	}

	// download the tarball
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request, error: %w", err)
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to download artifact from %s, error: %w", url, err)
	}
	defer resp.Body.Close()

	// check response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download artifact, status: %s", resp.Status)
	}

	var buf bytes.Buffer

	// verify checksum matches origin
	if err := verifyArtifact(repository.GetArtifact(), &buf, resp.Body); err != nil {
		return "", err
	}

	// extract
	summary, err := untar.Untar(&buf, dir)
	if err != nil {
		return "", fmt.Errorf("failed to untar artifact, error: %w", err)
	}

	return summary, nil
}

func verifyArtifact(artifact *sourcev1.Artifact, buf *bytes.Buffer, reader io.Reader) error {
	hasher := sha256.New()

	// for backwards compatibility with source-controller v0.17.2 and older
	if len(artifact.Checksum) == 40 {
		hasher = sha1.New()
	}

	// compute checksum
	mw := io.MultiWriter(hasher, buf)
	if _, err := io.Copy(mw, reader); err != nil {
		return err
	}

	if checksum := fmt.Sprintf("%x", hasher.Sum(nil)); checksum != artifact.Checksum {
		return fmt.Errorf("failed to verify artifact: computed checksum '%s' doesn't match advertised '%s'",
			checksum, artifact.Checksum)
	}

	return nil
}

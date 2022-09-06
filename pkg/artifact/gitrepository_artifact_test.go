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

package artifact_test

import (
	"context"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/hrk091/nwctl/pkg/artifact"
	"github.com/stretchr/testify/assert"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchArtifact(t *testing.T) {
	dir := t.TempDir()
	want := []byte("dummy")
	checksum, buf := mustGenTgzArchive("test.txt", string(want))

	tests := []struct {
		name     string
		handler  http.HandlerFunc
		checksum string
		wantErr  bool
	}{
		{
			"ok",
			func(w http.ResponseWriter, r *http.Request) {
				if _, err := io.Copy(w, buf); err != nil {
					panic(err)
				}
			},
			checksum,
			false,
		},
		{
			"bad: wrong checksum",
			func(w http.ResponseWriter, r *http.Request) {
				if _, err := io.Copy(w, buf); err != nil {
					panic(err)
				}
			},
			"wrong checksum",
			true,
		},
		{
			"bad: wrong contents",
			func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte("wrong content")); err != nil {
					panic(err)
				}
			},
			checksum,
			true,
		},
		{
			"bad: error from server",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(500)
			},
			checksum,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := httptest.NewServer(tt.handler)
			repo := sourcev1.GitRepository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-ns",
				},
				Status: sourcev1.GitRepositoryStatus{
					Artifact: &sourcev1.Artifact{
						URL:      h.URL,
						Checksum: tt.checksum,
					},
				},
			}

			_, err := artifact.FetchArtifact(context.Background(), repo, dir)
			if tt.wantErr {
				t.Log(err)
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				got, err := os.ReadFile(filepath.Join(dir, "test.txt"))
				exitOnErr(t, err)
				assert.Equal(t, want, got)
			}

		})
	}

	t.Run("bad: url not set", func(t *testing.T) {
		repo := sourcev1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test-ns",
			},
			Status: sourcev1.GitRepositoryStatus{
				Artifact: &sourcev1.Artifact{
					Checksum: checksum,
				},
			},
		}

		_, err := artifact.FetchArtifact(context.Background(), repo, dir)
		assert.Error(t, err)
	})

}

func TestReplaceRevision(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		revision string
		want     string
	}{
		{
			"ok: local",
			"http://source-controller.flux-system.svc.cluster.local./gitrepository/namespace/name/latest.tar.gz",
			"abc123",
			"http://source-controller.flux-system.svc.cluster.local./gitrepository/namespace/name/abc123.tar.gz",
		},
		{
			"ok: https",
			"https://source-controller.flux-system.svc.cluster.local./gitrepository/namespace/name/93e99aa7.tar.gz",
			"abc123",
			"https://source-controller.flux-system.svc.cluster.local./gitrepository/namespace/name/abc123.tar.gz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replaced := artifact.ReplaceRevision(tt.url, tt.revision)
			assert.Equal(t, tt.want, replaced)
		})
	}
}

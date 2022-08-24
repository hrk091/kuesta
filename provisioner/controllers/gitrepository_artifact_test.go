/*
 * Copyright (c) 2022. Hiroki Okui
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package controllers_test

import (
	"context"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/hrk091/nwctl/provisioner/controllers"
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

			_, err := controllers.FetchArtifact(context.Background(), repo, dir)
			if tt.wantErr {
				t.Log(err)
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				got, err := os.ReadFile(filepath.Join(dir, "test.txt"))
				ExitOnErr(t, err)
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

		_, err := controllers.FetchArtifact(context.Background(), repo, dir)
		assert.Error(t, err)
	})

}

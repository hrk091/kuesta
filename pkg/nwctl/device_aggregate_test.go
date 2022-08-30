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

package nwctl_test

import (
	"bytes"
	"context"
	"encoding/json"
	extgogit "github.com/go-git/go-git/v5"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecodeSaveConfigRequest(t *testing.T) {
	config := "foobar"

	tests := []struct {
		name    string
		given   string
		want    *nwctl.SaveConfigRequest
		wantErr bool
	}{
		{
			"ok",
			`{"device": "device1", "config": "foobar"}`,
			&nwctl.SaveConfigRequest{
				Device: "device1",
				Config: &config,
			},
			false,
		},
		{
			"err: no device",
			`{"config": "foobar"}`,
			nil,
			true,
		},
		{
			"err: no config",
			`{"device": "device1"}`,
			nil,
			true,
		},
		{
			"err: invalid format",
			`{"device": "device1", "config": "foobar"`,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.given)
			got, err := nwctl.DecodeSaveConfigRequest(r)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMakeSyncCommitMessage(t *testing.T) {
	stmap := extgogit.Status{
		"devices/dvc1/actual_config.cue": &extgogit.FileStatus{Staging: extgogit.Added},
		"devices/dvc2/actual_config.cue": &extgogit.FileStatus{Staging: extgogit.Deleted},
		"devices/dvc3/actual_config.cue": &extgogit.FileStatus{Staging: extgogit.Modified},
	}
	want := `Updated: dvc1 dvc2 dvc3

Devices:
	added:     dvc1
	deleted:   dvc2
	modified:  dvc3`

	assert.Equal(t, want, nwctl.MakeSyncCommitMessage(stmap))
}

func TestDeviceAggregateServer_SaveConfig(t *testing.T) {
	dir := t.TempDir()
	config := "foobar"
	given := &nwctl.SaveConfigRequest{
		Device: "device1",
		Config: &config,
	}

	s := nwctl.NewDeviceAggregateServer(&nwctl.DeviceAggregateCfg{
		RootCfg: nwctl.RootCfg{RootPath: dir},
	})
	err := s.SaveConfig(context.Background(), given)
	assert.Nil(t, err)

	got, err := os.ReadFile(filepath.Join(dir, "devices", "device1", "actual_config.cue"))
	exitOnErr(t, err)
	assert.Equal(t, []byte(config), got)
}

func TestDeviceAggregateServer_GitPushSyncBranch(t *testing.T) {
	testRemote := "test-remote"

	t.Run("ok", func(t *testing.T) {
		repo, dir := setupGitRepoWithRemote(t, testRemote)
		s := nwctl.NewDeviceAggregateServer(&nwctl.DeviceAggregateCfg{
			RootCfg: nwctl.RootCfg{
				RootPath:  dir,
				GitRemote: testRemote,
			},
		})
		err := s.GitPushSyncBranch(context.Background())
		exitOnErr(t, err)

		exists := false
		for _, b := range getRemoteBranches(t, repo, testRemote) {
			if strings.HasPrefix(b.Name().Short(), "SYNC-") {
				exists = true
			}
		}
		assert.True(t, exists)
	})
}

func TestDeviceAggregateServer_Run(t *testing.T) {
	testRemote := "test-remote"
	repo, dir := setupGitRepoWithRemote(t, testRemote)
	config := "foobar"
	req := nwctl.SaveConfigRequest{
		Device: "device1",
		Config: &config,
	}

	buf, err := json.Marshal(req)
	exitOnErr(t, err)
	request := httptest.NewRequest(http.MethodPost, "/commit", bytes.NewBuffer(buf))
	response := httptest.NewRecorder()

	s := nwctl.NewDeviceAggregateServer(&nwctl.DeviceAggregateCfg{
		RootCfg: nwctl.RootCfg{
			RootPath:  dir,
			GitRemote: testRemote,
		},
	})
	nwctl.UpdateCheckDuration = 100 * time.Millisecond
	s.Run(context.Background())

	s.HandleFunc(response, request)
	res := response.Result()
	assert.Equal(t, 200, res.StatusCode)

	var got []byte
	assert.Eventually(t, func() bool {
		got, err = os.ReadFile(filepath.Join(dir, "devices", "device1", "actual_config.cue"))
		return err == nil
	}, time.Second, 100*time.Millisecond)
	assert.Equal(t, []byte(config), got)

	assert.Eventually(t, func() bool {
		exists := false
		for _, b := range getRemoteBranches(t, repo, testRemote) {
			if strings.HasPrefix(b.Name().Short(), "SYNC-") {
				exists = true
			}
		}
		return exists
	}, time.Second, time.Millisecond)

}

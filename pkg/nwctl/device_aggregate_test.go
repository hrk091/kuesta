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
		RootCfg: nwctl.RootCfg{StatusRootPath: dir},
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
		repo, dir, _ := setupGitRepoWithRemote(t, testRemote)
		oldRef, _ := repo.Head()
		assert.Greater(t, len(getStatus(t, repo)), 0)

		s := nwctl.NewDeviceAggregateServer(&nwctl.DeviceAggregateCfg{
			RootCfg: nwctl.RootCfg{
				StatusRootPath: dir,
				GitRemote:      testRemote,
			},
		})
		err := s.GitPushDeviceConfig(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, len(getStatus(t, repo)), 0)

		localRef, _ := repo.Head()
		remoteRef := getRemoteBranch(t, repo, testRemote, "main")
		assert.NotEqual(t, localRef.Hash().String(), oldRef.Hash().String())
		assert.Equal(t, localRef.Hash().String(), remoteRef.Hash().String())
	})
}

func TestDeviceAggregateServer_Run(t *testing.T) {
	testRemote := "test-remote"
	repo, dir, _ := setupGitRepoWithRemote(t, testRemote)
	config := "foobar"
	req := nwctl.SaveConfigRequest{
		Device: "device1",
		Config: &config,
	}

	s := nwctl.NewDeviceAggregateServer(&nwctl.DeviceAggregateCfg{
		RootCfg: nwctl.RootCfg{
			StatusRootPath: dir,
			GitRemote:      testRemote,
		},
	})
	nwctl.UpdateCheckDuration = 100 * time.Millisecond
	s.Run(context.Background())

	buf, err := json.Marshal(req)
	exitOnErr(t, err)
	request := httptest.NewRequest(http.MethodPost, "/commit", bytes.NewBuffer(buf))
	response := httptest.NewRecorder()
	s.HandleFunc(response, request)
	res := response.Result()
	assert.Equal(t, 200, res.StatusCode)

	var got []byte
	assert.Eventually(t, func() bool {
		got, err = os.ReadFile(filepath.Join(dir, "devices", "device1", "actual_config.cue"))
		return err == nil
	}, time.Second, 100*time.Millisecond)
	assert.Equal(t, []byte(config), got)

	assert.Greater(t, len(getStatus(t, repo)), 0)
	assert.Eventually(t, func() bool {
		localRef, _ := repo.Head()
		remoteRef := getRemoteBranch(t, repo, testRemote, "main")
		return localRef.Hash().String() == remoteRef.Hash().String()
	}, time.Second, 100*time.Millisecond)

	assert.Equal(t, len(getStatus(t, repo)), 0)
}

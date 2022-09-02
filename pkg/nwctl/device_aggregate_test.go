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
	"github.com/go-git/go-git/v5/plumbing"
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
	trunkBranch := "main"

	t.Run("ok: new branch", func(t *testing.T) {
		repo, dir, _ := setupGitRepoWithRemote(t, testRemote)

		s := nwctl.NewDeviceAggregateServer(&nwctl.DeviceAggregateCfg{
			RootCfg: nwctl.RootCfg{
				RootPath:  dir,
				GitRemote: testRemote,
			},
		})
		err := s.GitPushSyncBranch(context.Background())
		exitOnErr(t, err)

		count := 0
		for _, b := range getRemoteBranches(t, repo, testRemote) {
			if strings.HasPrefix(b.Name().Short(), "SYNC-") {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("ok: existing branch", func(t *testing.T) {
		repo, dir, _ := setupGitRepoWithRemote(t, testRemote)

		syncBranch := "SYNC-1662097098"
		exitOnErr(t, createBranch(repo, syncBranch))
		exitOnErr(t, checkout(repo, trunkBranch))

		s := nwctl.NewDeviceAggregateServer(&nwctl.DeviceAggregateCfg{
			RootCfg: nwctl.RootCfg{
				RootPath:  dir,
				GitRemote: testRemote,
				GitTrunk:  trunkBranch,
			},
		})
		err := s.GitPushSyncBranch(context.Background())
		exitOnErr(t, err)

		count := 0
		for _, b := range getRemoteBranches(t, repo, testRemote) {
			if strings.HasPrefix(b.Name().Short(), "SYNC-") {
				count++
				assert.Equal(t, syncBranch, b.Name().Short())
			}
		}
		assert.Equal(t, 1, count)
	})
}

func TestLatestSyncBranch(t *testing.T) {
	syncBrOld := "SYNC-1662097098"
	syncBrNew := "SYNC-1700000000"
	dummyBr := "DUMMY-123"

	tests := []struct {
		name  string
		given []*plumbing.Reference
		want  string
	}{
		{
			"ok: select one from multi",
			[]*plumbing.Reference{
				plumbing.NewReferenceFromStrings(plumbing.NewBranchReferenceName(syncBrOld).String(), "test"),
				plumbing.NewReferenceFromStrings(plumbing.NewBranchReferenceName(syncBrNew).String(), "test"),
				plumbing.NewReferenceFromStrings(plumbing.NewBranchReferenceName(dummyBr).String(), "test"),
			},
			"SYNC-1700000000",
		},
		{
			"ok: single",
			[]*plumbing.Reference{
				plumbing.NewReferenceFromStrings(plumbing.NewBranchReferenceName(syncBrNew).String(), "test"),
				plumbing.NewReferenceFromStrings(plumbing.NewBranchReferenceName(dummyBr).String(), "test"),
			},
			"SYNC-1700000000",
		},
		{
			"ok: not found",
			[]*plumbing.Reference{
				plumbing.NewReferenceFromStrings(plumbing.NewBranchReferenceName(dummyBr).String(), "test"),
			},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nwctl.LatestSyncBranch(tt.given)
			assert.Equal(t, tt.want, got)
		})
	}

}

func TestDeviceAggregateServer_Run(t *testing.T) {
	testRemote := "test-remote"
	repo, dir, _ := setupGitRepoWithRemote(t, testRemote)
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

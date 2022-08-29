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
	"context"
	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
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
	setup := func(t *testing.T) (*extgogit.Repository, string) {
		_, url := initBareRepo(t)

		repo, dir := initRepo(t, "main")
		_, err := repo.CreateRemote(&config.RemoteConfig{
			Name: testRemote,
			URLs: []string{url},
		})
		exitOnErr(t, err)

		exitOnErr(t, addFile(repo, "devices/device1/actual_config.cue", "{will_deleted: _}"))
		exitOnErr(t, addFile(repo, "devices/device2/actual_config.cue", "{will_updated: _}"))
		_, err = commit(repo, time.Now())
		exitOnErr(t, err)

		exitOnErr(t, repo.CreateBranch(&config.Branch{
			Name:   "main",
			Remote: testRemote,
			Merge:  plumbing.NewBranchReferenceName("main"),
		}))

		exitOnErr(t, push(repo, "main", testRemote))

		exitOnErr(t, deleteFile(repo, "devices/device1/actual_config.cue"))
		exitOnErr(t, addFile(repo, "devices/device2/actual_config.cue", "{updated: _}"))
		exitOnErr(t, addFile(repo, "devices/device3/actual_config.cue", "{added: _}"))

		return repo, dir
	}

	t.Run("ok", func(t *testing.T) {
		repo, dir := setup(t)
		s := nwctl.NewDeviceAggregateServer(&nwctl.DeviceAggregateCfg{
			RootCfg: nwctl.RootCfg{
				RootPath:  dir,
				GitRemote: testRemote,
			},
		})
		err := s.GitPushSyncBranch(context.Background())
		exitOnErr(t, err)

		remote, err := repo.Remote(testRemote)
		exitOnErr(t, err)
		branches, err := remote.List(&extgogit.ListOptions{})
		exitOnErr(t, err)

		exists := false
		for _, b := range branches {
			t.Log(b.Name().Short())
			if strings.HasPrefix(b.Name().Short(), "SYNC-") {
				exists = true
			}
		}
		assert.True(t, exists)
	})
}

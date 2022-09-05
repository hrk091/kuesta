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
	extgogit "github.com/go-git/go-git/v5"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"regexp"
	"testing"
	"time"
)

func TestServiceApplyCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(cfg *nwctl.ServiceApplyCfg)) *nwctl.ServiceApplyCfg {
		cfg := &nwctl.ServiceApplyCfg{
			RootCfg: nwctl.RootCfg{
				ConfigRootPath: "./",
			},
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *nwctl.ServiceApplyCfg)
		wantError bool
	}{
		{
			"ok",
			func(cfg *nwctl.ServiceApplyCfg) {},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidStruct(tt.transform)
			err := cfg.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCheckGitStatus(t *testing.T) {
	tests := []struct {
		st      extgogit.Status
		wantErr int
	}{
		{
			extgogit.Status{
				"computed/device1.cue":           &extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified},
				"devices/device1/config.cue":     &extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified},
				"services/foo/one/two/input.cue": &extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Unmodified},
			},
			0,
		},
		{
			extgogit.Status{
				"computed/device1.cue":           &extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
				"devices/device1/config.cue":     &extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
				"services/foo/one/two/input.cue": &extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Modified},
			},
			3,
		},
	}

	for _, tt := range tests {
		err := nwctl.CheckGitStatus(tt.st)
		if tt.wantErr > 0 {
			reg := regexp.MustCompile(".cue")
			assert.Equal(t, tt.wantErr, len(reg.FindAllString(err.Error(), -1)))
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestCheckGitFileStatus(t *testing.T) {
	tests := []struct {
		path    string
		st      extgogit.FileStatus
		wantErr bool
	}{
		{
			"computed/device1.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
			true,
		},
		{
			"computed/device1.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified},
			false,
		},
		{
			"devices/device1/config.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
			true,
		},
		{
			"devices/device1/config.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Unmodified},
			false,
		},
		{
			"services/foo/one/two/input.cue",
			extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Modified},
			true,
		},
		{
			"services/foo/one/two/input.cue",
			extgogit.FileStatus{Staging: extgogit.Modified, Worktree: extgogit.Unmodified},
			false,
		},
		{
			"services/foo/one/two/input.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.Modified},
			false,
		},
		{
			"services/foo/one/two/input.cue",
			extgogit.FileStatus{Staging: extgogit.Unmodified, Worktree: extgogit.UpdatedButUnmerged},
			true,
		},
	}

	for _, tt := range tests {
		err := nwctl.CheckGitFileStatus(tt.path, tt.st)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestServiceCompilePlan_Do(t *testing.T) {
	var err error
	repo, dir := initRepo(t, "main")
	exitOnErr(t, addFile(repo, "services/foo/one/input.cue", "{}"))
	_, err = commit(repo, time.Now())
	exitOnErr(t, err)

	exitOnErr(t, deleteFile(repo, "services/foo/one/input.cue"))
	exitOnErr(t, addFile(repo, "services/foo/two/input.cue", "{}"))
	exitOnErr(t, addFile(repo, "services/foo/three/input.cue", "{}"))

	stmap := getStatus(t, repo)
	plan := nwctl.NewServiceCompilePlan(stmap, dir)
	updated := 0
	deleted := 0
	err = plan.Do(context.Background(),
		func(ctx context.Context, sp nwctl.ServicePath) error {
			deleted++
			assert.Equal(t, "foo", sp.Service)
			assert.Contains(t, []string{"one"}, sp.Keys[0])
			return nil
		},
		func(ctx context.Context, sp nwctl.ServicePath) error {
			updated++
			assert.Equal(t, "foo", sp.Service)
			assert.Contains(t, []string{"two", "three"}, sp.Keys[0])
			return nil
		})
	assert.Equal(t, 1, deleted)
	assert.Equal(t, 2, updated)
	assert.Nil(t, err)
}

func TestServiceCompilePlan_IsEmpty(t *testing.T) {
	var err error
	repo, dir := initRepo(t, "main")
	exitOnErr(t, addFile(repo, "services/foo/one/input.cue", "{}"))
	_, err = commit(repo, time.Now())
	exitOnErr(t, err)

	stmap := getStatus(t, repo)
	plan := nwctl.NewServiceCompilePlan(stmap, dir)
	assert.True(t, plan.IsEmpty())
}

func TestDeviceCompositePlan_Do(t *testing.T) {
	var err error
	repo, dir := initRepo(t, "main")
	exitOnErr(t, addFile(repo, "services/foo/one/computed/device1.cue", "{}"))
	_, err = commit(repo, time.Now())
	exitOnErr(t, err)

	exitOnErr(t, deleteFile(repo, "services/foo/one/computed/device1.cue"))
	exitOnErr(t, addFile(repo, "services/foo/two/computed/device2.cue", "{}"))
	exitOnErr(t, addFile(repo, "services/foo/three/computed/device3.cue", "{}"))

	stmap := getStatus(t, repo)
	plan := nwctl.NewDeviceCompositePlan(stmap, dir)
	executed := 0
	err = plan.Do(context.Background(),
		func(ctx context.Context, dp nwctl.DevicePath) error {
			executed++
			assert.Contains(t, []string{"device1", "device2", "device3"}, dp.Device)
			return nil
		})
	assert.Equal(t, 3, executed)
	assert.Nil(t, err)
}

func TestDeviceCompositePlan_IsEmpty(t *testing.T) {
	var err error
	repo, dir := initRepo(t, "main")
	exitOnErr(t, addFile(repo, "services/foo/one/computed/device1.cue", "{}"))
	_, err = commit(repo, time.Now())
	exitOnErr(t, err)

	stmap := getStatus(t, repo)
	plan := nwctl.NewDeviceCompositePlan(stmap, dir)
	assert.True(t, plan.IsEmpty())
}

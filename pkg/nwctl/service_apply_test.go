package nwctl_test

import (
	extgogit "github.com/go-git/go-git/v5"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"testing"
)

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

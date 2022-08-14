package nwctl_test

import (
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRootCfgBuilder_Verbose(t *testing.T) {
	tests := []struct {
		name      string
		given     uint8
		want      *nwctl.RootCfg
		wantError bool
	}{
		{"warn level", 0, &nwctl.RootCfg{Verbose: 0}, false},
		{"debug level", 3, &nwctl.RootCfg{Verbose: 3}, false},
		{"invalid: over range", 4, nil, true},
	}

	for _, tt := range tests {
		cfg, err := nwctl.NewRootCfg().Verbose(tt.given).Build()
		assert.Equal(t, tt.want, cfg)
		if tt.wantError {
			var e *nwctl.ErrConfigValue
			assert.ErrorAs(t, err, &e)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestRootCfgBuilder_Devel(t *testing.T) {
	cfg, _ := nwctl.NewRootCfg().Devel(true).Build()
	want := &nwctl.RootCfg{Devel: true}
	assert.Equal(t, want, cfg)
}

func TestRootCfgBuilder_RootPath(t *testing.T) {
	tests := []struct {
		name      string
		given     string
		want      *nwctl.RootCfg
		wantError bool
	}{
		{"filled", "foo/bar", &nwctl.RootCfg{RootPath: "foo/bar"}, false},
		{"invalid: empty", "", nil, true},
	}
	for _, tt := range tests {
		cfg, err := nwctl.NewRootCfg().RootPath(tt.given).Build()
		assert.Equal(t, tt.want, cfg)
		if tt.wantError {
			var e *nwctl.ErrConfigValue
			assert.ErrorAs(t, err, &e)
		} else {
			assert.Nil(t, err)
		}
	}
}

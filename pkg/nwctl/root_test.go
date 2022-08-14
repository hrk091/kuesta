package nwctl_test

import (
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVerbose(t *testing.T) {
	tests := []struct {
		name      string
		given     uint8
		want      *nwctl.RootCfg
		wantError bool
	}{
		{"warn level", 0, &nwctl.RootCfg{Verbose: 0}, false},
		{"debug level", 3, &nwctl.RootCfg{Verbose: 3}, false},
		{"over range", 4, nil, true},
	}

	for _, tt := range tests {
		cfg, err := nwctl.NewRootCfg(nwctl.Verbose(tt.given))
		assert.Equal(t, tt.want, cfg)
		if tt.wantError {
			var e *nwctl.ErrConfigValue
			assert.ErrorAs(t, err, &e)
		}
	}
}

func TestDevel(t *testing.T) {
	cfg, _ := nwctl.NewRootCfg(nwctl.Devel(true))
	want := &nwctl.RootCfg{Devel: true}
	assert.Equal(t, want, cfg)
}

func TestRootPath(t *testing.T) {
	cfg, _ := nwctl.NewRootCfg(nwctl.RootPath("foo/bar"))
	want := &nwctl.RootCfg{RootPath: "foo/bar"}
	assert.Equal(t, want, cfg)
}

package nwctl_test

import (
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestServiceCompileCfgBuilder_Service(t *testing.T) {
	tests := []struct {
		name      string
		given     string
		want      *nwctl.ServiceCompileCfg
		wantError bool
	}{
		{"filled", "test", &nwctl.ServiceCompileCfg{Service: "test"}, false},
		{"invalid: empty", "", nil, true},
	}
	for _, tt := range tests {
		cfg, err := nwctl.NewServiceCompileCfg().Service(tt.given).Build()
		assert.Equal(t, tt.want, cfg)
		if tt.wantError {
			var e *nwctl.ErrConfigValue
			assert.ErrorAs(t, err, &e)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestServiceCompileCfgBuilder_Keys(t *testing.T) {
	tests := []struct {
		name      string
		given     []string
		want      *nwctl.ServiceCompileCfg
		wantError bool
	}{
		{"filled", []string{"foo", "bar"}, &nwctl.ServiceCompileCfg{Keys: []string{"foo", "bar"}}, false},
		{"invalid: empty", nil, nil, true},
	}
	for _, tt := range tests {
		cfg, err := nwctl.NewServiceCompileCfg().Keys(tt.given).Build()
		assert.Equal(t, tt.want, cfg)
		if tt.wantError {
			var e *nwctl.ErrConfigValue
			assert.ErrorAs(t, err, &e)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestServiceCompileCfgBuilder_Build_multiTimes(t *testing.T) {
	b := nwctl.NewServiceCompileCfg()
	cfg, _ := b.Service("before").Keys([]string{"before"}).Build()

	cfg2, _ := b.Keys([]string{"after"}).Build()
	assert.Equal(t, &nwctl.ServiceCompileCfg{Service: "before", Keys: []string{"before"}}, cfg)
	assert.Equal(t, &nwctl.ServiceCompileCfg{Service: "before", Keys: []string{"after"}}, cfg2)
}

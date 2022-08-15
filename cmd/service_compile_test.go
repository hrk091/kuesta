package cmd_test

import (
	"github.com/hrk091/nwctl/cmd"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRootCmd_ServiceCompile(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			"service is not set",
			[]string{"service", "compile", "-r=./"},
			true,
		},
		{
			"keys is not set",
			[]string{"service", "compile", "abc", "-r=./"},
			true,
		},
	}

	for _, tt := range tests {
		c := cmd.NewRootCmd()
		c.SetArgs(tt.args)
		err := c.Execute()
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

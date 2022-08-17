package gogit_test

import (
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hrk091/nwctl/pkg/gogit"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGit_Validate(t *testing.T) {
	newValidStruct := func(t func(git *gogit.Git)) *gogit.Git {
		g := &gogit.Git{
			Path:   "./",
			Branch: "main",
		}
		t(g)
		return g
	}

	tests := []struct {
		name      string
		transform func(g *gogit.Git)
		wantErr   bool
	}{
		{
			"ok",
			func(g *gogit.Git) {},
			false,
		},
		{
			"bad: path is empty",
			func(g *gogit.Git) {
				g.Path = ""
			},
			true,
		},
		{
			"bad: branch is empty",
			func(g *gogit.Git) {
				g.Branch = ""
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newValidStruct(tt.transform)
			err := g.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestGit_BasicAuth(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  *githttp.BasicAuth
	}{
		{
			"both username and password",
			"user:pass",
			&githttp.BasicAuth{
				Username: "user",
				Password: "pass",
			},
		},
		{
			"only password",
			"pass",
			&githttp.BasicAuth{
				Username: "anonymous",
				Password: "pass",
			},
		},
		{
			"not set",
			"",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gogit.Git{Token: tt.token}
			assert.Equal(t, tt.want, g.BasicAuth())
		})
	}
}

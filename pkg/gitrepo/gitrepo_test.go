/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package gitrepo_test

import (
	"github.com/nttcom/kuesta/pkg/gitrepo"
	"github.com/stretchr/testify/assert"
	"testing"
)

//func NewMockClientFunc(t *testing.T) gitrepo.NewGitClientFunc {
//	mockCtrl := gomock.NewController(t)
//	return func(repoURL string, token string) gitrepo.GitRepoClient {
//		return NewMockGitRepoClient(mockCtrl)
//	}
//}
//
func TestNewGitRepoClient(t *testing.T) {
	token := ""

	var tests = []struct {
		name     string
		given    string
		wantKind string
		wantErr  bool
	}{
		{
			"ok",
			"https://github.com/hrk091/kuesta-testdata",
			"github",
			false,
		},
		{
			"err: incorrect git repo",
			"https://not.exist.com/hrk091/kuesta-testdata",
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := gitrepo.NewGitRepoClient(tt.given, token)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.wantKind, c.Kind())
			}
		})
	}
}

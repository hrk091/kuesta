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

package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidate(t *testing.T) {

	type WithValidateTag struct {
		Required string `validate:"required"`
		Limited  uint8  `validate:"max=1"`
	}

	tests := []struct {
		name    string
		given   WithValidateTag
		wantErr bool
	}{
		{
			"ok",
			WithValidateTag{
				Required: "foo",
				Limited:  1,
			},
			false,
		},
		{
			"bad",
			WithValidateTag{
				Required: "",
				Limited:  3,
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.given)
			t.Log(err)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

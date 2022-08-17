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

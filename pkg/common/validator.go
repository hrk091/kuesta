package common

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"strings"
)

var (
	_validator = validator.New()
)

func Validate(v any) error {
	return errors.WithStack(handleError(_validator.Struct(v)))
}

func handleError(err error) error {
	switch e := err.(type) {
	case validator.ValidationErrors:
		var errMsg []string
		for _, fe := range e {
			errMsg = append(errMsg, fe.Error())
		}
		return fmt.Errorf(strings.Join(errMsg, "\n"))
	default:
		return e
	}
}

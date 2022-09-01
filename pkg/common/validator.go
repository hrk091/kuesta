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

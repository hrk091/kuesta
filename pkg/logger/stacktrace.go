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

package logger

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
)

// ShowStackTrace shows the stacktrace of the original error only.
func ShowStackTrace(w io.Writer, err error) {
	if st := GetStackTrace(err); st != "" {
		fmt.Fprintf(w, "StackTrace: %s\n\n", st)
	}
}

// GetStackTrace returns the stacktrace of the original error only.
func GetStackTrace(err error) string {
	st := bottomStackTrace(err)
	if st != nil {
		return fmt.Sprintf("%+v", st.StackTrace())
	}
	return ""
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func bottomStackTrace(err error) stackTracer {
	var st stackTracer
	if errors.Unwrap(err) != nil {
		st = bottomStackTrace(errors.Unwrap(err))
		if st != nil {
			return st
		}
	}
	if e, ok := err.(stackTracer); ok {
		return e
	}
	return nil
}

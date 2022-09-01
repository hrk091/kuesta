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

package nwctl

import (
	"context"
	"io"
	"os"
)

type _keyWriter struct{}

// WithWriter sets io.Writer to context for outputting message from command.
func WithWriter(parent context.Context, w io.Writer) context.Context {
	return context.WithValue(parent, _keyWriter{}, w)
}

// WriterFromContext extract io.Writer from context.
func WriterFromContext(ctx context.Context) io.Writer {
	v, ok := ctx.Value(_keyWriter{}).(io.Writer)
	if !ok {
		return os.Stdout
	} else {
		return v
	}
}

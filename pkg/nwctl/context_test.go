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

package nwctl_test

import (
	"bytes"
	"context"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestWriterFromContext(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, os.Stdout, nwctl.WriterFromContext(ctx))
}

func TestWriterFromContext_WithWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := context.Background()
	ctx = nwctl.WithWriter(ctx, buf)
	assert.Equal(t, buf, nwctl.WriterFromContext(ctx))
}

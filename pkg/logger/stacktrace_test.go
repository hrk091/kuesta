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

package logger_test

import (
	"bytes"
	"github.com/hrk091/nwctl/pkg/logger"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

func TestShowStackTrace(t *testing.T) {
	t.Run("nested error", func(t *testing.T) {
		err1 := errors.New("foo")
		err2 := errors.Wrap(err1, "bar")
		err3 := errors.Wrap(err2, "baz")

		buf := &bytes.Buffer{}
		logger.ShowStackTrace(buf, err3)

		found := regexp.MustCompile("testing.tRunner").FindAllIndex(buf.Bytes(), -1)
		assert.Equal(t, 1, len(found))
	})

	t.Run("single error", func(t *testing.T) {
		err := errors.New("foo")

		buf := &bytes.Buffer{}
		logger.ShowStackTrace(buf, err)

		found := regexp.MustCompile("testing.tRunner").FindAllIndex(buf.Bytes(), -1)
		assert.Equal(t, 1, len(found))
	})

	t.Run("nil", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger.ShowStackTrace(buf, nil)
		t.Log(buf)

		assert.Equal(t, 0, len(buf.Bytes()))
	})
}

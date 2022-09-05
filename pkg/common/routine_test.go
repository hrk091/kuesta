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

package common_test

import (
	"context"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSetInterval(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		count := 0
		common.SetInterval(context.Background(), func() {
			count++
		}, time.Millisecond)

		assert.Eventually(t, func() bool {
			return count > 2
		}, time.Second, 5*time.Millisecond)
	})

	t.Run("ok: not called after cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		count := 0
		common.SetInterval(ctx, func() {
			count++
			cancel()
		}, time.Millisecond)
		time.Sleep(5 * time.Millisecond)
		assert.Equal(t, 1, count)
	})
}

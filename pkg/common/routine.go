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
	"context"
	"fmt"
	"github.com/hrk091/nwctl/pkg/logger"
	"strings"
	"time"
)

func SetInterval(ctx context.Context, fn func(), dur time.Duration, msgs ...string) {
	l := logger.FromContext(ctx)
	msg := strings.Join(msgs, " ")

	go func() {
		for {
			select {
			case <-time.After(dur):
				l.Debugf("run func (every %s): %s", dur.String(), msg)
				fn()
			case <-ctx.Done():
				return
			}
		}
	}()
	l.Infof(fmt.Sprintf("start interval loop: %s", msg))
}

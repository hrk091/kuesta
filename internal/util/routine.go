/*
 Copyright (c) 2022-2023 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package util

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nttcom/kuesta/internal/logger"
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
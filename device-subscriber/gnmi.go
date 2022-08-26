/*
 * Copyright (c) 2022. Hiroki Okui
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package main

import (
	"context"
	"fmt"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
)

// ExtractJsonIetfVal extracts the JSON IETF field of the supplied TypedValue.
func ExtractJsonIetfVal(tv *gnmi.TypedValue) ([]byte, error) {
	v, ok := tv.GetValue().(*gnmi.TypedValue_JsonIetfVal)
	if !ok {
		return nil, errors.WithStack(fmt.Errorf("value did not contain IETF JSON"))
	}
	return v.JsonIetfVal, nil
}

// GetEntireConfig requests gNMI GetRequest and returns entire device config as
func GetEntireConfig(ctx context.Context, client *gnmiclient.Client) ([]byte, error) {
	req := gnmi.GetRequest{
		Path: []*gnmi.Path{
			{}, // TODO consider to specify target and path
		},
		Encoding: gnmi.Encoding_JSON_IETF,
	}

	resp, err := client.Get(ctx, &req)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// resolve gnmi TypedValue
	var tv *gnmi.TypedValue
	for _, v := range resp.GetNotification() {
		for _, u := range v.GetUpdate() {
			tv = u.GetVal()
			break
		}
	}
	if tv == nil {
		return nil, errors.WithStack(fmt.Errorf("no content from gNMI server"))
	}

	return ExtractJsonIetfVal(tv)
}

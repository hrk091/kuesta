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
	"bytes"
	"context"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"encoding/json"
	"fmt"
	"github.com/hrk091/nwctl/device-subscriber/pkg/model"
	"github.com/hrk091/nwctl/pkg/logger"
	"github.com/hrk091/nwctl/pkg/nwctl"
	gclient "github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

func Run(cfg Config) error {
	ctx := context.Background()

	c, err := gnmiclient.New(ctx, gclient.Destination{
		Addrs:   []string{cfg.Addr},
		Target:  "",
		Timeout: 60 * time.Second,
		Credentials: &gclient.Credentials{
			Username: cfg.Username,
			Password: cfg.Password,
		},
	})
	if err != nil {
		return fmt.Errorf("create gNMI client: %w", errors.WithStack(err))
	}

	cb := func() error {
		return Sync(ctx, cfg, c.(*gnmiclient.Client))
	}
	if err := Subscribe(ctx, c, cb); err != nil {
		return err
	}
	return nil
}

func Subscribe(ctx context.Context, c gclient.Impl, cb func() error) error {
	l := logger.FromContext(ctx)

	query := gclient.Query{
		NotificationHandler: func(noti gclient.Notification) error {
			if err, ok := noti.(error); ok {
				return fmt.Errorf("error received: %w", err)
			}
			l.Infow("notification received", "item", noti)
			return nil
		},
	}

	if err := c.Subscribe(ctx, query); err != nil {
		return fmt.Errorf("open subscribe channel: %w", errors.WithStack(err))
	}
	defer func() {
		if err := c.Close(); err != nil {
			l.Errorf("close gNMI subscription: %w", err)
		}
	}()

	for {
		err := c.Recv()
		if err == io.EOF {
			l.Infow("EOF")
			return nil
		} else if err != nil {
			return fmt.Errorf("error received on gNMI subscribe channel: %w", err)
		}
		if err := cb(); err != nil {
			l.Errorf("sync on received: %v", err)
		}
	}

}

func Sync(ctx context.Context, cfg Config, client *gnmiclient.Client) error {
	buf, err := GetEntireConfig(ctx, client)
	if err != nil {
		return fmt.Errorf("get device config: %w", err)
	}

	// unmarshal
	var obj model.Device
	schema, err := model.Schema()
	if err != nil {
		return err
	}
	if err := schema.Unmarshal(buf, &obj); err != nil {
		return fmt.Errorf("decode JSON IETF val of gNMI update: %w", err)
	}

	// convert to CUE
	cctx := cuecontext.New()
	v := cctx.Encode(obj)
	b, err := nwctl.FormatCue(v, cue.Final())
	if err != nil {
		return fmt.Errorf("encode cue.Value to bytes: %w", err)
	}

	if err := PostDeviceConfig(cfg, b); err != nil {
		return err
	}

	return nil
}

type SaveConfigRequest struct {
	Device string `json:"device"`
	Config string `json:"config"`
}

// PostDeviceConfig sends HTTP POST with supplied device config.
func PostDeviceConfig(cfg Config, data []byte) error {
	u, err := url.Parse(cfg.AggregatorURL)
	if err != nil {
		return fmt.Errorf("url parse error: %w", errors.WithStack(err))
	}
	u.Path = path.Join(u.Path, "commit")
	body := SaveConfigRequest{
		Device: cfg.Device,
		Config: string(data),
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&body); err != nil {
		return fmt.Errorf("json encode error: %w", errors.WithStack(err))
	}

	resp, err := http.Post(u.String(), "application/json", &buf)
	if err != nil {
		return fmt.Errorf("post: %w", errors.WithStack(err))
	}
	if resp.StatusCode != 200 {
		return errors.WithStack(fmt.Errorf("error response: %d", resp.StatusCode))
	}
	defer resp.Body.Close()
	return nil
}

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

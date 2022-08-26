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
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

func Subscribe(cfg Config) error {
	ctx := context.Background()
	l := logger.FromContext(ctx)

	// send gNMI Subscribe Query
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
	l.Infof("start subscribe: %s (%s)", cfg.Device, cfg.Addr)

	for {
		err = c.Recv()
		if err == io.EOF {
			l.Infow("EOF")
			return nil
		} else if err != nil {
			return fmt.Errorf("error received on gNMI subscribe channel: %w", err)
		}
		if err := Sync(ctx, cfg, c.(*gnmiclient.Client)); err != nil {
			// ignore error not to stop server even when raised
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

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

package main

import (
	"bytes"
	"context"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"encoding/json"
	"fmt"
	"github.com/hrk091/nwctl/device-subscriber/pkg/model"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/logger"
	"github.com/hrk091/nwctl/pkg/nwctl"
	gclient "github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/url"
	"os"
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

	fn := func() error {
		return Sync(ctx, cfg, c.(*gnmiclient.Client))
	}
	if err := Subscribe(ctx, c, fn); err != nil {
		return err
	}
	return nil
}

func Subscribe(ctx context.Context, c gclient.Impl, fn func() error) error {
	l := logger.FromContext(ctx)

	query := gclient.Query{
		Type: gclient.Stream,
		NotificationHandler: func(noti gclient.Notification) error {
			if err, ok := noti.(error); ok {
				return fmt.Errorf("error received: %w", err)
			}
			// NOTE Run something if needed
			return nil
		},
	}

	l.Infow("starting subscribe...")
	if err := c.Subscribe(ctx, query); err != nil {
		return fmt.Errorf("open subscribe channel: %w", errors.WithStack(err))
	}
	l.Infow("started subscribe")

	defer func() {
		if err := c.Close(); err != nil {
			l.Errorf("close gNMI subscription: %w", err)
		}
	}()

	for {
		recvErr := c.Recv()
		l.Infow("recv hooked")
		if err := fn(); err != nil {
			l.Errorf("handle notification: %v", err)
			common.ShowStackTrace(os.Stderr, err)
		}

		if recvErr == io.EOF {
			l.Infow("EOF")
			return nil
		} else if recvErr != nil {
			return fmt.Errorf("error received on gNMI subscribe channel: %w", recvErr)
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

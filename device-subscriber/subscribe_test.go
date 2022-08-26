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
	"encoding/json"
	nwctlgnmi "github.com/hrk091/nwctl/pkg/gnmi"
	"github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetEntireConfig(t *testing.T) {
	config := []byte("dummy")

	tests := []struct {
		name    string
		handler func(ctx context.Context, request *pb.GetRequest) (*pb.GetResponse, error)
		wantErr bool
	}{
		{
			"ok",
			func(ctx context.Context, request *pb.GetRequest) (*pb.GetResponse, error) {
				v := gnmi.TypedValue{
					Value: &gnmi.TypedValue_JsonIetfVal{
						JsonIetfVal: config,
					},
				}
				resp := &pb.GetResponse{
					Notification: []*pb.Notification{
						{
							Update: []*pb.Update{
								{Path: &pb.Path{}, Val: &v},
							},
						},
					},
				}
				return resp, nil
			},
			false,
		},
		{
			"bad: no content",
			func(ctx context.Context, request *pb.GetRequest) (*pb.GetResponse, error) {
				resp := &pb.GetResponse{
					Notification: []*pb.Notification{},
				}
				return resp, nil
			},
			true,
		},
		{
			"bad: error response",
			func(ctx context.Context, request *pb.GetRequest) (*pb.GetResponse, error) {
				return nil, status.Error(codes.Internal, "error")
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &nwctlgnmi.GnmiMock{
				GetHandler: tt.handler,
			}
			ctx := context.Background()
			s, conn := nwctlgnmi.NewServer(ctx, m)
			defer s.Stop()

			c, err := gnmiclient.NewFromConn(ctx, conn, client.Destination{})
			got, err := GetEntireConfig(ctx, c)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, config, got)
			}
		})
	}
}

func TestPostDeviceConfig(t *testing.T) {
	deviceConfig := "dummy"

	t.Run("ok", func(t *testing.T) {
		cfg := Config{
			Device: "device1",
		}
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req SaveConfigRequest
			exitOnErr(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, req.Device, cfg.Device)
			assert.Equal(t, req.Config, deviceConfig)
		}))
		cfg.AggregatorURL = s.URL

		err := PostDeviceConfig(cfg, []byte(deviceConfig))
		assert.Nil(t, err)
	})

	t.Run("bad: error response", func(t *testing.T) {
		cfg := Config{
			Device: "device1",
		}
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}))
		cfg.AggregatorURL = s.URL

		err := PostDeviceConfig(cfg, []byte(deviceConfig))
		assert.Error(t, err)
	})

	t.Run("bad: wrong url", func(t *testing.T) {
		cfg := Config{
			Device: "device1",
		}
		cfg.AggregatorURL = ":60000"

		err := PostDeviceConfig(cfg, []byte(deviceConfig))
		assert.Error(t, err)
	})

	t.Run("bad: connection error", func(t *testing.T) {
		cfg := Config{
			Device: "device1",
		}
		cfg.AggregatorURL = "http://localhost:60000"

		err := PostDeviceConfig(cfg, []byte(deviceConfig))
		assert.Error(t, err)
	})
}

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
	"context"
	"encoding/json"
	nwctlgnmi "github.com/hrk091/nwctl/pkg/gnmi"
	gclient "github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSubscribe(t *testing.T) {
	v := gnmi.TypedValue{
		Value: &gnmi.TypedValue_JsonIetfVal{
			JsonIetfVal: []byte(`{"foo": "bar"}`),
		},
	}
	m := &nwctlgnmi.GnmiMock{
		SubscribeHandler: func(stream pb.GNMI_SubscribeServer) error {
			for i := 0; i < 3; i++ {
				resp := &pb.SubscribeResponse{
					Response: &pb.SubscribeResponse_Update{
						Update: &pb.Notification{
							Timestamp: time.Now().UnixNano(),
							Update: []*pb.Update{
								{Path: &pb.Path{Target: "*"}, Val: &v},
							},
						},
					},
				}
				if err := stream.Send(resp); err != nil {
					return err
				}
			}
			return nil
		},
	}
	ctx := context.Background()
	gs, conn := nwctlgnmi.NewServer(ctx, m)
	defer gs.Stop()

	client, err := gnmiclient.NewFromConn(ctx, conn, gclient.Destination{})
	exitOnErr(t, err)

	count := 0
	err = Subscribe(ctx, client, func(noti gclient.Notification) error {
		count++
		return nil
	})
	assert.Nil(t, err)
	assert.Equal(t, 3+1, count)
}

func TestSync(t *testing.T) {
	config := []byte(`{
  "openconfig-interfaces:interfaces": {
    "interface": [
      {
        "config": {
          "description": "foo",
          "enabled": true,
          "mtu": 9000,
          "name": "Ethernet1",
          "type": "iana-if-type:ethernetCsmacd"
        },
        "name": "Ethernet1",
        "state": {
          "admin-status": "UP",
          "oper-status": "UP"
        }
      }
    ]
  }
}
`)
	want := `{
	Interface: {
		Ethernet1: {
			AdminStatus: 1
			Description: "foo"
			Enabled:     true
			Mtu:         9000
			Name:        "Ethernet1"
			OperStatus:  1
			Type:        80
		}
	}
}`

	m := &nwctlgnmi.GnmiMock{
		GetHandler: func(ctx context.Context, request *pb.GetRequest) (*pb.GetResponse, error) {
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
	}
	ctx := context.Background()
	gs, conn := nwctlgnmi.NewServer(ctx, m)
	defer gs.Stop()

	client, err := gnmiclient.NewFromConn(ctx, conn, gclient.Destination{})
	exitOnErr(t, err)

	cfg := Config{
		Device: "device1",
	}
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SaveConfigRequest
		exitOnErr(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, req.Device, cfg.Device)
		assert.Equal(t, want, req.Config)
	}))
	cfg.AggregatorURL = hs.URL

	err = Sync(ctx, cfg, client)
	assert.Nil(t, err)
}

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

			c, err := gnmiclient.NewFromConn(ctx, conn, gclient.Destination{})
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

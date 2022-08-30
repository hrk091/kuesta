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

package nwctl_test

import (
	"context"
	"github.com/hrk091/nwctl/pkg/nwctl"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"path/filepath"
	"testing"
)

func TestServeCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(cfg *nwctl.ServeCfg)) *nwctl.ServeCfg {
		cfg := &nwctl.ServeCfg{
			RootCfg: nwctl.RootCfg{
				RootPath: "./",
			},
			Addr: ":9339",
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *nwctl.ServeCfg)
		wantError bool
	}{
		{
			"ok",
			func(cfg *nwctl.ServeCfg) {},
			false,
		},
		{
			"err: addr is empty",
			func(cfg *nwctl.ServeCfg) {
				cfg.Addr = ""
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidStruct(tt.transform)
			err := cfg.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestGnmiPathConverter_Convert(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		prefix  *pb.Path
		path    *pb.Path
		setup   func(dir string)
		want    any
		wantErr bool
	}{
		{
			"ok: service",
			nil,
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			func(dir string) {
				path := filepath.Join(dir, "services", "foo", "metadata.json")
				exitOnErr(t, nwctl.WriteFileWithMkdir(path, []byte(`{"keys": ["bar", "baz"]}`)))
			},
			&nwctl.ServicePath{
				RootDir: dir,
				Service: "foo",
				Keys:    []string{"one", "two"},
			},
			false,
		},
		{
			"ok: service with prefix",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			func(dir string) {
				path := filepath.Join(dir, "services", "foo", "metadata.json")
				exitOnErr(t, nwctl.WriteFileWithMkdir(path, []byte(`{"keys": ["bar", "baz"]}`)))
			},
			&nwctl.ServicePath{
				RootDir: dir,
				Service: "foo",
				Keys:    []string{"one", "two"},
			},
			false,
		},
		{
			"ok: device",
			nil,
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "devices"},
					{Name: "device", Key: map[string]string{"name": "device1"}},
				},
			},
			nil,
			&nwctl.DevicePath{
				RootDir: dir,
				Device:  "device1",
			},
			false,
		},
		{
			"ok: device with prefix",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "devices"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "device", Key: map[string]string{"name": "device1"}},
				},
			},
			nil,
			&nwctl.DevicePath{
				RootDir: dir,
				Device:  "device1",
			},
			false,
		},
		{
			"err: service meta not exist",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "invalid", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			nil,
			nil,
			true,
		},
		{
			"err: elem length is less than 2",
			nil,
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "device", Key: map[string]string{"name": "device1"}},
				},
			},
			nil,
			nil,
			true,
		},
		{
			"err: invalid name",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "invalid"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "device", Key: map[string]string{"name": "device1"}},
				},
			},
			nil,
			nil,
			true,
		},
		{
			"err: invalid service name",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "invalid", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			nil,
			nil,
			true,
		},
		{
			"err: invalid device name",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "devices"},
				},
			},
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "invalid", Key: map[string]string{"name": "device1"}},
				},
			},
			nil,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(dir)
			}
			c := nwctl.NewGnmiPathConverter(&nwctl.ServeCfg{
				RootCfg: nwctl.RootCfg{
					RootPath: dir,
				},
			})
			got, err := c.Convert(tt.prefix, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				switch r := got.(type) {
				case nwctl.ServicePathReq:
					assert.Equal(t, tt.want, r.Path())
				case nwctl.DevicePathReq:
					assert.Equal(t, tt.want, r.Path())
				default:
					t.Fatalf("unexpected type: %T", got)
				}
			}
		})
	}
}

func TestNorthboundServer_Get(t *testing.T) {
	serviceMeta := []byte(`{"keys": ["bar", "baz"]}`)
	serviceInput := []byte(`{port: 1}`)
	serviceInputJson := []byte(`{"port":1}`)
	invalidServiceInput := []byte(`{port: 1`)

	deviceConfig := []byte(`config: {
	Interface: Ethernet1: {
		Name:    1
	}
}`)
	deviceConfigJson := []byte(`{"config":{"Interface":{"Ethernet1":{"Name":1}}}}`)
	invalidDeviceConfig := []byte(`config: {`)

	tests := []struct {
		name    string
		given   *pb.GetRequest
		setup   func(dir string)
		want    *pb.GetResponse
		wantErr codes.Code
	}{
		{
			"ok: without prefix",
			&pb.GetRequest{
				Prefix: nil,
				Path: []*pb.Path{
					{
						Elem: []*pb.PathElem{
							{Name: "services"},
							{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
						},
					},
					{
						Elem: []*pb.PathElem{
							{Name: "devices"},
							{Name: "device", Key: map[string]string{"name": "device1"}},
						},
					},
				},
			},
			func(dir string) {
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), serviceInput))
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), deviceConfig))
			},
			&pb.GetResponse{
				Notification: []*pb.Notification{
					{
						Prefix: nil,
						Update: []*pb.Update{
							{
								Path: &pb.Path{
									Elem: []*pb.PathElem{
										{Name: "services"},
										{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
									},
								},
								Val: &pb.TypedValue{Value: &pb.TypedValue_JsonVal{JsonVal: serviceInputJson}},
							},
						},
					},
					{
						Prefix: nil,
						Update: []*pb.Update{
							{
								Path: &pb.Path{
									Elem: []*pb.PathElem{
										{Name: "devices"},
										{Name: "device", Key: map[string]string{"name": "device1"}},
									},
								},
								Val: &pb.TypedValue{Value: &pb.TypedValue_JsonVal{JsonVal: deviceConfigJson}},
							},
						},
					},
				},
			},
			codes.OK,
		},
		{
			"ok: service with prefix",
			&pb.GetRequest{
				Prefix: &pb.Path{
					Elem: []*pb.PathElem{
						{Name: "services"},
					},
				},
				Path: []*pb.Path{
					{
						Elem: []*pb.PathElem{
							{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
						},
					},
				},
			},
			func(dir string) {
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), serviceInput))
			},
			&pb.GetResponse{
				Notification: []*pb.Notification{
					{
						Prefix: &pb.Path{
							Elem: []*pb.PathElem{
								{Name: "services"},
							},
						},
						Update: []*pb.Update{
							{
								Path: &pb.Path{
									Elem: []*pb.PathElem{
										{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
									},
								},
								Val: &pb.TypedValue{Value: &pb.TypedValue_JsonVal{JsonVal: serviceInputJson}},
							},
						},
					},
				},
			},
			codes.OK,
		},
		{
			"ok: device with prefix",
			&pb.GetRequest{
				Prefix: &pb.Path{
					Elem: []*pb.PathElem{
						{Name: "devices"},
					},
				},
				Path: []*pb.Path{
					{
						Elem: []*pb.PathElem{
							{Name: "device", Key: map[string]string{"name": "device1"}},
						},
					},
				},
			},
			func(dir string) {
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), deviceConfig))
			},
			&pb.GetResponse{
				Notification: []*pb.Notification{
					{
						Prefix: &pb.Path{
							Elem: []*pb.PathElem{
								{Name: "devices"},
							},
						},
						Update: []*pb.Update{
							{
								Path: &pb.Path{
									Elem: []*pb.PathElem{
										{Name: "device", Key: map[string]string{"name": "device1"}},
									},
								},
								Val: &pb.TypedValue{Value: &pb.TypedValue_JsonVal{JsonVal: deviceConfigJson}},
							},
						},
					},
				},
			},
			codes.OK,
		},
		{
			"err: service not found",
			&pb.GetRequest{
				Path: []*pb.Path{
					{
						Elem: []*pb.PathElem{
							{Name: "services"},
							{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
						},
					},
				},
			},
			func(dir string) {
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
			},
			nil,
			codes.NotFound,
		},
		{
			"err: device not found",
			&pb.GetRequest{
				Path: []*pb.Path{
					{
						Elem: []*pb.PathElem{
							{Name: "devices"},
							{Name: "device", Key: map[string]string{"name": "device1"}},
						},
					},
				},
			},
			nil,
			nil,
			codes.NotFound,
		},
		{
			"err: invalid service input",
			&pb.GetRequest{
				Path: []*pb.Path{
					{
						Elem: []*pb.PathElem{
							{Name: "services"},
							{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
						},
					},
				},
			},
			func(dir string) {
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), invalidServiceInput))
			},
			nil,
			codes.Internal,
		},
		{
			"err: invalid device input",
			&pb.GetRequest{
				Path: []*pb.Path{
					{
						Elem: []*pb.PathElem{
							{Name: "devices"},
							{Name: "device", Key: map[string]string{"name": "device1"}},
						},
					},
				},
			},
			func(dir string) {
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "config.cue"), invalidDeviceConfig))
			},
			nil,
			codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.setup != nil {
				tt.setup(dir)
			}
			s := nwctl.NewNorthboundServer(&nwctl.ServeCfg{
				RootCfg: nwctl.RootCfg{
					RootPath: dir,
				},
			})
			got, err := s.Get(context.Background(), tt.given)
			if tt.wantErr != codes.OK {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, status.Code(err))
			} else {
				assert.Nil(t, err)
				for i, noti := range got.Notification {
					assert.Equal(t, tt.want.Notification[i].String(), noti.String())
				}
			}
		})
	}

}

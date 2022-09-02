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

package nwctl_test

import (
	"context"
	"github.com/hrk091/nwctl/pkg/nwctl"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
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

func TestNorthboundServer_Capabilities(t *testing.T) {
	dir := t.TempDir()
	fooMeta := []byte(`{"keys": ["device", "port"], "organization": "org-foo", "version": "0.1.0"}`)
	barMeta := []byte(`{"keys": ["vlan"]}`)
	fooModel := &pb.ModelData{Name: "foo", Organization: "org-foo", Version: "0.1.0"}
	barModel := &pb.ModelData{Name: "bar"}

	exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), fooMeta))
	exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "bar", "metadata.json"), barMeta))
	exitOnErr(t, os.MkdirAll(filepath.Join(dir, "services", "baz"), 0750))

	s, err := nwctl.NewNorthboundServer(&nwctl.ServeCfg{
		RootCfg: nwctl.RootCfg{
			RootPath: dir,
		},
	})
	exitOnErr(t, err)
	got, err := s.Capabilities(context.Background(), &pb.CapabilityRequest{})
	assert.Nil(t, err)
	assert.Contains(t, got.SupportedModels, fooModel)
	assert.Contains(t, got.SupportedModels, barModel)
	assert.NotNil(t, got.SupportedEncodings)
	assert.NotNil(t, got.GNMIVersion)
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
			s, err := nwctl.NewNorthboundServer(&nwctl.ServeCfg{
				RootCfg: nwctl.RootCfg{
					RootPath: dir,
				},
			})
			exitOnErr(t, err)
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

func TestNorthboundServer_DoDelete(t *testing.T) {
	serviceInput := []byte(`{port: 1}`)
	serviceMeta := []byte(`{"keys": ["bar", "baz"]}`)

	tests := []struct {
		name        string
		path        *pb.Path
		setup       func(dir string)
		want        *pb.UpdateResult
		pathRemoved string
		wantErr     codes.Code
	}{
		{
			"ok",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			func(dir string) {
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), serviceInput))
			},
			&pb.UpdateResult{
				Op: pb.UpdateResult_DELETE,
			},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			codes.OK,
		},
		{
			"ok: already deleted",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			func(dir string) {
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
			},
			&pb.UpdateResult{
				Op: pb.UpdateResult_DELETE,
			},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			codes.OK,
		},
		{
			"err: metadata not exist",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			func(dir string) {},
			&pb.UpdateResult{
				Op: pb.UpdateResult_DELETE,
			},
			"",
			codes.InvalidArgument,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			s, err := nwctl.NewNorthboundServer(&nwctl.ServeCfg{
				RootCfg: nwctl.RootCfg{
					RootPath: dir,
				},
			})
			exitOnErr(t, err)

			got, err := s.DoDelete(context.Background(), nil, tt.path)
			if tt.wantErr != codes.OK {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, status.Code(err))
			} else {
				t.Log(err)
				assert.Nil(t, err)
				tt.want.Path = tt.path
				assert.Equal(t, tt.want.String(), got.String())

				_, err := os.ReadFile(filepath.Join(dir, tt.pathRemoved))
				assert.True(t, errors.Is(err, os.ErrNotExist))
			}
		})
	}
}

func TestNorthboundServer_DoReplace(t *testing.T) {
	serviceMeta := []byte(`{"keys": ["bar", "baz"]}`)
	serviceInput := []byte(`{port: 1, mtu: 9000}`)
	requestJson := []byte(`{"port": 2, "desc": "test"}`)
	invalidJson := []byte(`{"port": 2`)

	tests := []struct {
		name        string
		path        *pb.Path
		val         *pb.TypedValue
		setup       func(dir string)
		want        *pb.UpdateResult
		pathUpdated string
		valUpdated  []byte
		wantErr     codes.Code
	}{
		{
			"ok: replace existing",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: requestJson},
			},
			func(dir string) {
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), serviceInput))
			},
			&pb.UpdateResult{
				Op: pb.UpdateResult_REPLACE,
			},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			[]byte(`{
	bar:  "one"
	baz:  "two"
	desc: "test"
	port: 2.0
}`),
			codes.OK,
		},
		{
			"ok: new service",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: requestJson},
			},
			func(dir string) {
				exitOnErr(t, nwctl.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
			},
			&pb.UpdateResult{
				Op: pb.UpdateResult_REPLACE,
			},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			[]byte(`{
	bar:  "one"
	baz:  "two"
	desc: "test"
	port: 2.0
}`),
			codes.OK,
		},
		{
			"err: metadata not exist",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: requestJson},
			},
			func(dir string) {},
			&pb.UpdateResult{
				Op: pb.UpdateResult_REPLACE,
			},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			nil,
			codes.InvalidArgument,
		},
		{
			"err: invalid json input",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: invalidJson},
			},
			func(dir string) {},
			&pb.UpdateResult{
				Op: pb.UpdateResult_REPLACE,
			},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			nil,
			codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			s, err := nwctl.NewNorthboundServer(&nwctl.ServeCfg{
				RootCfg: nwctl.RootCfg{
					RootPath: dir,
				},
			})
			exitOnErr(t, err)

			got, err := s.DoReplace(context.Background(), nil, tt.path, tt.val)
			if tt.wantErr != codes.OK {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, status.Code(err))
			} else {
				t.Log(err)
				assert.Nil(t, err)
				tt.want.Path = tt.path
				assert.Equal(t, tt.want.String(), got.String())

				buf, err := os.ReadFile(filepath.Join(dir, tt.pathUpdated))
				assert.Nil(t, err)
				t.Logf("%s", string(buf))
				assert.Equal(t, tt.valUpdated, buf)
			}
		})
	}
}

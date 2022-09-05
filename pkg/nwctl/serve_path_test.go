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
	"github.com/hrk091/nwctl/pkg/nwctl"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

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
					ConfigRootPath: dir,
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

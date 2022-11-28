/*
 Copyright (c) 2022 NTT Communications Corporation

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

package kuesta_test

import (
	"context"
	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/nttcom/kuesta/pkg/gogit"
	"github.com/nttcom/kuesta/pkg/kuesta"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServeCfg_Validate(t *testing.T) {

	newValidStruct := func(t func(cfg *kuesta.ServeCfg)) *kuesta.ServeCfg {
		cfg := &kuesta.ServeCfg{
			RootCfg: kuesta.RootCfg{
				ConfigRootPath: "./",
			},
			Addr: ":9339",
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *kuesta.ServeCfg)
		wantError bool
	}{
		{
			"ok",
			func(cfg *kuesta.ServeCfg) {},
			false,
		},
		{
			"err: addr is empty",
			func(cfg *kuesta.ServeCfg) {
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

func TestRunSyncLoop(t *testing.T) {
	repo, _, dirBare := setupGitRepoWithRemote(t, "origin")
	repoPuller, dirPuller := cloneRepo(t, &extgogit.CloneOptions{
		URL:           dirBare,
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName("main"),
	})

	wantHash, err := commit(repo, time.Now())
	exitOnErr(t, err)
	exitOnErr(t, push(repo, "main", "origin"))

	git, err := gogit.NewGit(&gogit.GitOptions{
		Path: dirPuller,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kuesta.RunSyncLoop(ctx, git, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		ref, err := repoPuller.Head()
		if err != nil {
			return false
		}
		return wantHash.String() == ref.Hash().String()
	}, time.Second, 100*time.Millisecond)
}

func TestNorthboundServerImpl_Capabilities(t *testing.T) {
	dir := t.TempDir()
	fooMeta := []byte(`{"keys": ["device", "port"], "organization": "org-foo", "version": "0.1.0"}`)
	barMeta := []byte(`{"keys": ["vlan"]}`)
	fooModel := &pb.ModelData{Name: "foo", Organization: "org-foo", Version: "0.1.0"}
	barModel := &pb.ModelData{Name: "bar"}

	exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), fooMeta))
	exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "bar", "metadata.json"), barMeta))
	exitOnErr(t, os.MkdirAll(filepath.Join(dir, "services", "baz"), 0750))

	s := kuesta.NewNorthboundServerImpl(&kuesta.ServeCfg{
		RootCfg: kuesta.RootCfg{
			ConfigRootPath: dir,
		},
	})
	got, err := s.Capabilities(context.Background(), &pb.CapabilityRequest{})
	assert.Nil(t, err)
	assert.Contains(t, got.SupportedModels, fooModel)
	assert.Contains(t, got.SupportedModels, barModel)
	assert.NotNil(t, got.SupportedEncodings)
	assert.NotNil(t, got.GNMIVersion)
}

func TestNorthboundServerImpl_Get(t *testing.T) {
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
		prefix  *pb.Path
		path    *pb.Path
		setup   func(dir string)
		want    *pb.Notification
		wantErr codes.Code
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
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), serviceInput))
			},
			&pb.Notification{
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
			codes.OK,
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
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "actual_config.cue"), deviceConfig))
			},
			&pb.Notification{
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
			codes.OK,
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
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), serviceInput))
			},
			&pb.Notification{
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
			codes.OK,
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
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "actual_config.cue"), deviceConfig))
			},
			&pb.Notification{
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
			codes.OK,
		},
		{
			"err: service not found",
			nil,
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
			},
			nil,
			codes.NotFound,
		},
		{
			"err: device not found",
			nil,
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "devices"},
					{Name: "device", Key: map[string]string{"name": "device1"}},
				},
			},
			nil,
			nil,
			codes.NotFound,
		},
		{
			"err: invalid service input",
			nil,
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "two"}},
				},
			},
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), invalidServiceInput))
			},
			nil,
			codes.Internal,
		},
		{
			"err: invalid device input",
			nil,
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "devices"},
					{Name: "device", Key: map[string]string{"name": "device1"}},
				},
			},
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "devices", "device1", "actual_config.cue"), invalidDeviceConfig))
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
			s := kuesta.NewNorthboundServerImpl(&kuesta.ServeCfg{
				RootCfg: kuesta.RootCfg{
					ConfigRootPath: dir,
					StatusRootPath: dir,
				},
			})
			got, err := s.Get(context.Background(), tt.prefix, tt.path)
			if tt.wantErr != codes.OK {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, status.Code(err))
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNorthboundServerImpl_Delete(t *testing.T) {
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
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "two", "input.cue"), serviceInput))
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
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
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
			s := kuesta.NewNorthboundServerImpl(&kuesta.ServeCfg{
				RootCfg: kuesta.RootCfg{
					ConfigRootPath: dir,
				},
			})

			got, err := s.Delete(context.Background(), nil, tt.path)
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

func TestNorthboundServerImpl_Replace(t *testing.T) {
	serviceMeta := []byte(`{"keys": ["bar", "baz"]}`)
	serviceTransform := []byte(`#Input: {bar: string, baz: int, intVal: int, floatVal: float, strVal: string}`)
	serviceInputToBeUpdated := []byte(`{intVal: 1, floatVal: 1.1, strVal: "blabla"}`)
	requestJson := []byte(`{"bar": "dummy", "baz": 100, "intVal": 2, "floatVal": 2.1, "notDefined": "test"}`)
	invalidJson := []byte(`{"intVal": 2`)

	tests := []struct {
		name      string
		path      *pb.Path
		val       *pb.TypedValue
		setup     func(dir string)
		inputPath string
		wantVal   []byte
		wantErr   codes.Code
	}{
		{
			"ok: replace existing",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "2"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: requestJson},
			},
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "transform.cue"), serviceTransform))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "2", "input.cue"), serviceInputToBeUpdated))
			},
			filepath.Join("services", "foo", "one", "2", "input.cue"),
			[]byte(`{
	bar:        "one"
	baz:        2
	floatVal:   2.1
	intVal:     2
	notDefined: "test"
}`),
			codes.OK,
		},
		{
			"ok: new service",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "2"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: requestJson},
			},
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "transform.cue"), serviceTransform))
			},
			filepath.Join("services", "foo", "one", "2", "input.cue"),
			[]byte(`{
	bar:        "one"
	baz:        2
	floatVal:   2.1
	intVal:     2
	notDefined: "test"
}`),
			codes.OK,
		},
		{
			"err: metadata not exist",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "2"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: requestJson},
			},
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "transform.cue"), serviceTransform))
			},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			nil,
			codes.InvalidArgument,
		},
		{
			"err: transform.cue not exist",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "2"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: requestJson},
			},
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
			},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			nil,
			codes.Internal,
		},
		{
			"err: invalid json input",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "2"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: invalidJson},
			},
			func(dir string) {},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			nil,
			codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			s := kuesta.NewNorthboundServerImpl(&kuesta.ServeCfg{
				RootCfg: kuesta.RootCfg{
					ConfigRootPath: dir,
				},
			})

			got, err := s.Replace(context.Background(), nil, tt.path, tt.val)
			if tt.wantErr != codes.OK {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, status.Code(err))
			} else {
				assert.Nil(t, err)
				want := &pb.UpdateResult{
					Op:   pb.UpdateResult_REPLACE,
					Path: tt.path,
				}
				assert.Equal(t, want.String(), got.String())

				buf, err := os.ReadFile(filepath.Join(dir, tt.inputPath))
				assert.Nil(t, err)
				t.Log(string(buf))
				assert.Equal(t, tt.wantVal, buf)
			}
		})
	}
}

func TestNorthboundServerImpl_Update(t *testing.T) {
	serviceMeta := []byte(`{"keys": ["bar", "baz"]}`)
	serviceTransform := []byte(`#Input: {bar: string, baz: int, intVal: int, floatVal: float, strVal: string}`)
	serviceInputToBeUpdated := []byte(`{intVal: 1, floatVal: 1.1, strVal: "blabla"}`)
	requestJson := []byte(`{"bar": "dummy", "baz": 100, "intVal": 2, "floatVal": 2.1, "notDefined": "test"}`)
	invalidJson := []byte(`{"intVal": 2`)

	tests := []struct {
		name      string
		path      *pb.Path
		val       *pb.TypedValue
		setup     func(dir string)
		inputPath string
		wantVal   []byte
		wantErr   codes.Code
	}{
		{
			"ok: update existing",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "2"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: requestJson},
			},
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "transform.cue"), serviceTransform))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "2", "input.cue"), serviceInputToBeUpdated))
			},
			filepath.Join("services", "foo", "one", "2", "input.cue"),
			[]byte(`{
	bar:        "one"
	baz:        2
	floatVal:   2.1
	intVal:     2
	notDefined: "test"
	strVal:     "blabla"
}`),
			codes.OK,
		},
		{
			"err: not exist",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "2"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: requestJson},
			},
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "transform.cue"), serviceTransform))
			},
			filepath.Join("services", "foo", "one", "2", "input.cue"),
			nil,
			codes.NotFound,
		},
		{
			"err: metadata not exist",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "2"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: requestJson},
			},
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "transform.cue"), serviceTransform))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "2", "input.cue"), serviceInputToBeUpdated))
			},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			nil,
			codes.InvalidArgument,
		},
		{
			"err: transform.cue not exist",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "2"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: requestJson},
			},
			func(dir string) {
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "metadata.json"), serviceMeta))
				exitOnErr(t, kuesta.WriteFileWithMkdir(filepath.Join(dir, "services", "foo", "one", "2", "input.cue"), serviceInputToBeUpdated))
			},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			nil,
			codes.Internal,
		},
		{
			"err: invalid json input",
			&pb.Path{
				Elem: []*pb.PathElem{
					{Name: "services"},
					{Name: "service", Key: map[string]string{"kind": "foo", "bar": "one", "baz": "2"}},
				},
			},
			&pb.TypedValue{
				Value: &pb.TypedValue_JsonVal{JsonVal: invalidJson},
			},
			func(dir string) {},
			filepath.Join("services", "foo", "one", "two", "input.cue"),
			nil,
			codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			s := kuesta.NewNorthboundServerImpl(&kuesta.ServeCfg{
				RootCfg: kuesta.RootCfg{
					ConfigRootPath: dir,
				},
			})

			got, err := s.Update(context.Background(), nil, tt.path, tt.val)
			if tt.wantErr != codes.OK {
				t.Log(err)
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, status.Code(err))
			} else {
				assert.Nil(t, err)
				want := &pb.UpdateResult{
					Op:   pb.UpdateResult_UPDATE,
					Path: tt.path,
				}
				assert.Equal(t, want.String(), got.String())

				buf, err := os.ReadFile(filepath.Join(dir, tt.inputPath))
				assert.Nil(t, err)
				t.Log(string(buf))
				assert.Equal(t, tt.wantVal, buf)
			}
		})
	}
}

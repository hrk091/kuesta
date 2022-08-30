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

package nwctl

import (
	"context"
	"cuelang.org/go/cue/cuecontext"
	"fmt"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/logger"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"sync"
)

type ServeCfg struct {
	RootCfg

	Addr string `validate:"required"`
}

type PathType string

const (
	NodeService              = "service"
	NodeDevice               = "device"
	KeyServiceKind           = "kind"
	KeyDeviceName            = "name"
	PathTypeService PathType = NodeService
	PathTypeDevice  PathType = NodeDevice
)

// Validate validates exposed fields according to the `validate` tag.
func (c *ServeCfg) Validate() error {
	return common.Validate(c)
}

func RunServe(ctx context.Context, cfg *ServeCfg) error {
	l := logger.FromContext(ctx)
	l.Debug("serve called")
	return nil
}

type NorthboundServer struct {
	pb.UnimplementedGNMIServer

	cfg       *ServeCfg
	converter *GnmiPathConverter
	mu        sync.RWMutex // mu is the RW lock to protect the access to config
}

func NewNorthboundServer(cfg *ServeCfg) *NorthboundServer {
	return &NorthboundServer{
		cfg:       cfg,
		converter: NewGnmiPathConverter(cfg),
		mu:        sync.RWMutex{},
	}
}

func (s *NorthboundServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	if !s.mu.TryRLock() {
		return nil, status.Error(codes.Unavailable, "locked")
	}
	l := logger.FromContext(ctx)
	l.Debug("Get called")

	prefix := req.GetPrefix()
	paths := req.GetPath()
	var notifications []*pb.Notification

	// TODO support wildcard
	for _, path := range paths {
		n, err := s.doGet(ctx, prefix, path)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}

	return &pb.GetResponse{Notification: notifications}, nil
}

func (s *NorthboundServer) doGet(ctx context.Context, prefix, path *pb.Path) (*pb.Notification, error) {
	l := logger.FromContext(ctx)
	l.Debugw("get", "prefix", prefix, "path", path)

	pathReq, err := s.converter.Convert(prefix, path)
	if err != nil {
		return nil, err
	}

	var buf []byte
	switch r := pathReq.(type) {
	case ServicePathReq:
		buf, err = r.Path().ReadServiceInput()
	case DevicePathReq:
		buf, err = r.Path().ReadDeviceConfigFile()
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, status.Error(codes.NotFound, "not found")
		} else {
			return nil, err
		}
	}

	cctx := cuecontext.New()
	val, err := NewValueFromBuf(cctx, buf)
	if err != nil {
		fmt.Printf("################### %v\n", err)
		return nil, status.Error(codes.Internal, "failed to load cue")
	}

	// TODO get only nested tree

	jsonDump, err := val.MarshalJSON()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	update := &pb.Update{
		Path: path,
		Val:  &pb.TypedValue{Value: &pb.TypedValue_JsonVal{JsonVal: jsonDump}},
	}
	// TODO use timestamp when updated
	return &pb.Notification{Prefix: prefix, Update: []*pb.Update{update}}, nil
}

type PathReq interface {
	Type() PathType
}

type ServicePathReq struct {
	path *ServicePath
}

func (ServicePathReq) Type() PathType {
	return PathTypeService
}

func (s *ServicePathReq) Path() *ServicePath {
	return s.path
}

type DevicePathReq struct {
	path *DevicePath
}

func (DevicePathReq) Type() PathType {
	return PathTypeDevice
}

func (s DevicePathReq) Path() *DevicePath {
	return s.path
}

type GnmiPathConverter struct {
	cfg  *ServeCfg
	meta map[string]*ServiceMeta
}

func NewGnmiPathConverter(cfg *ServeCfg) *GnmiPathConverter {
	return &GnmiPathConverter{
		cfg:  cfg,
		meta: map[string]*ServiceMeta{},
	}
}

// Convert converts gNMI Path to PathReq.
func (c *GnmiPathConverter) Convert(prefix, path *pb.Path) (PathReq, error) {
	path = gnmiFullPath(prefix, path)
	elem := path.GetElem()
	if len(elem) < 2 {
		return nil, errors.WithStack(fmt.Errorf("path must have at least 2 elem"))
	}
	kindEl := elem[0]
	switch kindEl.GetName() {
	case DirServices:
		return c.convertService(elem[1:])
	case DirDevices:
		return c.convertDevice(elem[1:])
	default:
		return nil, errors.WithStack(fmt.Errorf("name of the first elem must be `%s` or `%s`", DirServices, DirDevices))
	}
}

func (c *GnmiPathConverter) convertService(elem []*pb.PathElem) (ServicePathReq, error) {
	svcEl := elem[0]
	if svcEl.GetName() != NodeService {
		return ServicePathReq{}, errors.WithStack(fmt.Errorf("name of second elem must be `%s`", NodeService))
	}
	keys := svcEl.GetKey()
	svcKind, ok := keys[KeyServiceKind]
	if !ok {
		return ServicePathReq{}, errors.WithStack(fmt.Errorf("`%s` key is required for service path", KeyServiceKind))
	}
	p := ServicePath{RootDir: c.cfg.RootPath, Service: svcKind}

	meta, ok := c.meta[svcKind]
	if !ok {
		m, err := p.ReadServiceMeta()
		if err != nil {
			return ServicePathReq{}, err
		}
		c.meta[svcKind] = m
		meta = m
	}

	for _, k := range meta.Keys {
		if v, ok := keys[k]; ok == true {
			p.Keys = append(p.Keys, v)
		} else {
			return ServicePathReq{}, errors.WithStack(fmt.Errorf("key `%s` of service %s is not supplied in Request Path", k, svcKind))
		}
	}
	return ServicePathReq{path: &p}, nil
}

func (c *GnmiPathConverter) convertDevice(elem []*pb.PathElem) (DevicePathReq, error) {
	svcEl := elem[0]
	if svcEl.GetName() != NodeDevice {
		return DevicePathReq{}, errors.WithStack(fmt.Errorf("name of second elem must be `%s`", NodeDevice))
	}
	keys := svcEl.GetKey()
	deviceName, ok := keys[KeyDeviceName]
	if !ok {
		return DevicePathReq{}, errors.WithStack(fmt.Errorf("`%s` key is required for service path", KeyDeviceName))
	}

	p := DevicePath{RootDir: c.cfg.RootPath, Device: deviceName}
	return DevicePathReq{path: &p}, nil
}

func gnmiFullPath(prefix, path *pb.Path) *pb.Path {
	fullPath := &pb.Path{}
	if path.GetElem() != nil {
		fullPath.Elem = append(prefix.GetElem(), path.GetElem()...)
	}
	return fullPath
}

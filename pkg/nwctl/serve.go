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
	"go.uber.org/zap"
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

func (s *NorthboundServer) Error(ctx context.Context, err error, msg string, kvs ...interface{}) {
	l := logger.FromContext(ctx).WithOptions(zap.AddCallerSkip(1))
	if st := common.GetStackTrace(err); st != "" {
		l = l.With("stacktrace", st)
	}
	l.Errorw(fmt.Sprintf("%s: %v", msg, err), kvs...)
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
		s.Error(ctx, err, "convert path request")
		return nil, status.Error(codes.InvalidArgument, err.Error())
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
			s.Error(ctx, err, "open file")
			return nil, status.Error(codes.NotFound, "not found")
		} else {
			return nil, err
		}
	}

	cctx := cuecontext.New()
	val, err := NewValueFromBuf(cctx, buf)
	if err != nil {
		s.Error(ctx, err, "load cue")
		return nil, status.Error(codes.Internal, "failed to read file")
	}

	// TODO get only nested tree

	jsonDump, err := val.MarshalJSON()
	if err != nil {
		s.Error(ctx, err, "encode json")
		return nil, status.Error(codes.Internal, "failed to encode to json")
	}

	update := &pb.Update{
		Path: path,
		Val:  &pb.TypedValue{Value: &pb.TypedValue_JsonVal{JsonVal: jsonDump}},
	}
	// TODO use timestamp when updated
	return &pb.Notification{Prefix: prefix, Update: []*pb.Update{update}}, nil
}

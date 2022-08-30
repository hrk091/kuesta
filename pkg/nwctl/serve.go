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
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"encoding/json"
	"fmt"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/hrk091/nwctl/pkg/gnmi"
	"github.com/hrk091/nwctl/pkg/gogit"
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

	mu        sync.RWMutex // mu is the RW lock to protect the access to config
	cfg       *ServeCfg
	converter *GnmiPathConverter
	git       *gogit.Git
}

func NewNorthboundServer(cfg *ServeCfg) *NorthboundServer {
	return &NorthboundServer{
		cfg:       cfg,
		converter: NewGnmiPathConverter(cfg),
		mu:        sync.RWMutex{},
	}
}

func (s *NorthboundServer) Error(l *zap.SugaredLogger, err error, msg string, kvs ...interface{}) {
	l = l.WithOptions(zap.AddCallerSkip(1))
	if st := common.GetStackTrace(err); st != "" {
		l = l.With("stacktrace", st)
	}
	l.Errorw(fmt.Sprintf("%s: %v", msg, err), kvs...)
}

var supportedEncodings = []pb.Encoding{pb.Encoding_JSON}

func (s *NorthboundServer) Capabilities(ctx context.Context, req *pb.CapabilityRequest) (*pb.CapabilityResponse, error) {
	l := logger.FromContext(ctx)
	l.Debug("Capabilities called")

	ver, err := gnmi.GetGNMIServiceVersion()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get gnmi service version: %v", err)
	}
	p := ServicePath{RootDir: s.cfg.RootPath}
	mlist, err := p.ReadServiceMetaAll()
	if err != nil {
		return nil, err
	}

	models := make([]*pb.ModelData, len(mlist))
	for i, m := range mlist {
		models[i] = m.ModelData()
	}

	return &pb.CapabilityResponse{
		SupportedModels:    models,
		SupportedEncodings: supportedEncodings,
		GNMIVersion:        *ver,
	}, nil
}

func (s *NorthboundServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	if !s.mu.TryRLock() {
		return nil, status.Error(codes.Unavailable, "locked")
	}
	l := logger.FromContext(ctx)
	l.Debugw("Get called")

	prefix := req.GetPrefix()
	paths := req.GetPath()
	var notifications []*pb.Notification

	// TODO support wildcard
	for _, path := range paths {
		n, grpcerr := s.DoGet(ctx, prefix, path)
		if grpcerr != nil {
			return nil, grpcerr
		}
		notifications = append(notifications, n)
	}

	return &pb.GetResponse{Notification: notifications}, nil
}

func (s *NorthboundServer) DoGet(ctx context.Context, prefix, path *pb.Path) (*pb.Notification, error) {
	l := logger.FromContext(ctx)

	req, err := s.converter.Convert(prefix, path)
	if err != nil {
		s.Error(l, err, "convert path request")
		return nil, status.Errorf(codes.InvalidArgument, "failed to convert path")
	}
	l = l.With("path", req.String())
	l.Debugw("get")

	var buf []byte
	switch r := req.(type) {
	case ServicePathReq:
		buf, err = r.Path().ReadServiceInput()
	case DevicePathReq:
		buf, err = r.Path().ReadDeviceConfigFile()
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, status.Errorf(codes.NotFound, "not found: %s", req.String())
		} else {
			s.Error(l, err, "open file")
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	cctx := cuecontext.New()
	val, err := NewValueFromBuf(cctx, buf)
	if err != nil {
		s.Error(l, err, "load cue")
		return nil, status.Errorf(codes.Internal, "failed to read file: %s", req.String())
	}

	// TODO get only nested tree

	jsonDump, err := val.MarshalJSON()
	if err != nil {
		s.Error(l, err, "encode json")
		return nil, status.Errorf(codes.Internal, "failed to encode to json: %s", req.String())
	}

	update := &pb.Update{
		Path: path,
		Val:  &pb.TypedValue{Value: &pb.TypedValue_JsonVal{JsonVal: jsonDump}},
	}
	// TODO use timestamp when updated
	return &pb.Notification{Prefix: prefix, Update: []*pb.Update{update}}, nil
}

func (s *NorthboundServer) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	l := logger.FromContext(ctx)
	l.Debugw("Set called")

	s.mu.Lock()
	defer func() {
		if err := s.git.Reset(gogit.ResetOptsHard()); err != nil {
			s.Error(l, err, "git reset")
		}
		s.mu.Unlock()
	}()

	// TODO run git merge-devices before set
	// TODO block when git worktree is dirty
	w, err := s.git.Checkout()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to checkout to %s", s.cfg.GitTrunk)
	}

	prefix := req.GetPrefix()
	var results []*pb.UpdateResult

	// TODO performance enhancement
	// TODO support wildcard
	for _, path := range req.GetDelete() {
		res, grpcerr := s.DoDelete(ctx, prefix, path)
		if grpcerr != nil {
			return nil, grpcerr
		}
		results = append(results, res)
	}
	for _, upd := range req.GetReplace() {
		res, grpcerr := s.DoReplace(ctx, prefix, upd.GetPath(), upd.GetVal())
		if grpcerr != nil {
			return nil, grpcerr
		}
		results = append(results, res)
	}
	for _, upd := range req.GetUpdate() {
		res, grpcerr := s.DoUpdate(ctx, prefix, upd.GetPath(), upd.GetVal())
		if grpcerr != nil {
			return nil, grpcerr
		}
		results = append(results, res)
	}

	sp := ServicePath{RootDir: s.cfg.RootPath}
	if _, err := w.Add(sp.ServiceDirPath(ExcludeRoot)); err != nil {
		s.Error(l, err, "git add")
		return nil, status.Errorf(codes.Internal, "failed to git-add")
	}

	serviceApplyCfg := ServiceApplyCfg{RootCfg: s.cfg.RootCfg}
	if err := RunServiceApply(ctx, &serviceApplyCfg); err != nil {
		s.Error(l, err, "service apply")
		return nil, status.Errorf(codes.Internal, "failed to apply service template")
	}

	gitCommitCfg := GitCommitCfg{
		RootCfg:    s.cfg.RootCfg,
		PushToMain: true,
	}
	if err := RunGitCommit(ctx, &gitCommitCfg); err != nil {
		s.Error(l, err, "git commit")
		return nil, status.Errorf(codes.Internal, "failed to git push to %s", s.cfg.GitTrunk)
	}

	return &pb.SetResponse{
		Prefix:   prefix,
		Response: results,
	}, nil
}

func (s *NorthboundServer) DoDelete(ctx context.Context, prefix, path *pb.Path) (*pb.UpdateResult, error) {
	l := logger.FromContext(ctx)

	// TODO delete partial nested data
	req, err := s.converter.Convert(prefix, path)
	if err != nil {
		s.Error(l, err, "convert path request")
		return nil, status.Errorf(codes.InvalidArgument, "failed to convert path")
	}
	r, ok := req.(ServicePathReq)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "only service mutation is supported: %s", r.String())
	}
	l = l.With("path", req.String())
	l.Debugw("delete")

	sp := r.Path()
	if err = os.Remove(sp.ServiceInputPath(IncludeRoot)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		} else {
			s.Error(l, err, "delete file")
			return nil, status.Errorf(codes.Internal, "failed to delete file: %s", r.String())
		}
	}
	return &pb.UpdateResult{Path: path, Op: pb.UpdateResult_DELETE}, nil
}

func (s *NorthboundServer) DoReplace(ctx context.Context, prefix, path *pb.Path, val *pb.TypedValue) (*pb.UpdateResult, error) {
	l := logger.FromContext(ctx)

	// TODO replace partial nested data
	req, err := s.converter.Convert(prefix, path)
	if err != nil {
		s.Error(l, err, "convert path request")
		return nil, status.Errorf(codes.InvalidArgument, "failed to convert path")
	}
	r, ok := req.(ServicePathReq)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "only service mutation is supported: %s", r.String())
	}
	l = l.With("path", req.String())
	l.Debugw("replace")

	cctx := cuecontext.New()

	input := map[string]any{}
	if err := json.Unmarshal(val.GetJsonVal(), &input); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode input: %s", r.String())
	}
	for k, v := range r.Keys() {
		input[k] = v
	}

	inputVal := cctx.Encode(input)
	if inputVal.Err() != nil {
		return nil, status.Errorf(codes.Internal, "failed to encode to cue: %s", r.String())
	}

	b, err := FormatCue(inputVal, cue.Final())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to format cue to bytes: %s", r.String())
	}
	sp := r.Path()
	if err := sp.WriteServiceInputFile(b); err != nil {
		s.Error(l, err, "write service input")
		return nil, status.Errorf(codes.Internal, "failed to write service input: %s", req.String())
	}

	return &pb.UpdateResult{Path: path, Op: pb.UpdateResult_REPLACE}, nil
}

func (s *NorthboundServer) DoUpdate(ctx context.Context, prefix, path *pb.Path, val *pb.TypedValue) (*pb.UpdateResult, error) {
	l := logger.FromContext(ctx)

	// TODO update partial nested data
	req, err := s.converter.Convert(prefix, path)
	if err != nil {
		s.Error(l, err, "convert path request")
		return nil, status.Errorf(codes.InvalidArgument, "failed to convert path")
	}
	r, ok := req.(ServicePathReq)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "only service mutation is supported: %s", r.String())
	}
	l = l.With("path", req.String())
	l.Debugw("update")

	// TODO implement
	return nil, status.Error(codes.Unimplemented, "")
}

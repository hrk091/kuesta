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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"net"
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

	// TODO credential
	//opts := credentials.ServerCredentials()
	g := grpc.NewServer()
	s, err := NewNorthboundServer(cfg)
	if err != nil {
		return fmt.Errorf("init gNMI impl server: %w", err)
	}
	pb.RegisterGNMIServer(g, s)
	reflection.Register(g)

	l.Infow("starting to listen", "address", cfg.Addr)
	listen, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	if err := g.Serve(listen); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

type NorthboundServer struct {
	pb.UnimplementedGNMIServer

	mu   sync.RWMutex // mu is the RW lock to protect the access to config
	cfg  *ServeCfg
	git  *gogit.Git
	impl GnmiRequestHandler
}

// NewNorthboundServer creates new NorthboundServer with supplied ServeCfg.
func NewNorthboundServer(cfg *ServeCfg) (*NorthboundServer, error) {
	git, err := gogit.NewGit(cfg.GitOptions())
	if err != nil {
		return nil, err
	}
	s := &NorthboundServer{
		cfg:  cfg,
		mu:   sync.RWMutex{},
		git:  git,
		impl: NewNorthboundServerImpl(cfg),
	}
	return s, nil
}

// Error shows an error with stacktrace if attached.
func (s *NorthboundServer) Error(l *zap.SugaredLogger, err error, msg string, kvs ...interface{}) {
	l = l.WithOptions(zap.AddCallerSkip(1))
	if st := common.GetStackTrace(err); st != "" {
		l = l.With("stacktrace", st)
	}
	l.Errorw(fmt.Sprintf("%s: %v", msg, err), kvs...)
}

var supportedEncodings = []pb.Encoding{pb.Encoding_JSON}

// Capabilities responds the server capabilities containing the available services.
func (s *NorthboundServer) Capabilities(ctx context.Context, req *pb.CapabilityRequest) (*pb.CapabilityResponse, error) {
	l := logger.FromContext(ctx)
	l.Debug("Capabilities called")

	return s.impl.Capabilities(ctx, req)
}

// Get responds the multiple service inputs requested by GetRequest.
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
		n, grpcerr := s.impl.Get(ctx, prefix, path)
		if grpcerr != nil {
			return nil, grpcerr
		}
		notifications = append(notifications, n)
	}

	return &pb.GetResponse{Notification: notifications}, nil
}

// Set executes specified Replace/Update/Delete operations and responds what is done by SetRequest.
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
		res, grpcerr := s.impl.Delete(ctx, prefix, path)
		if grpcerr != nil {
			return nil, grpcerr
		}
		results = append(results, res)
	}
	for _, upd := range req.GetReplace() {
		res, grpcerr := s.impl.Replace(ctx, prefix, upd.GetPath(), upd.GetVal())
		if grpcerr != nil {
			return nil, grpcerr
		}
		results = append(results, res)
	}
	for _, upd := range req.GetUpdate() {
		res, grpcerr := s.impl.Update(ctx, prefix, upd.GetPath(), upd.GetVal())
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

type GnmiRequestHandler interface {
	Capabilities(ctx context.Context, req *pb.CapabilityRequest) (*pb.CapabilityResponse, error)
	Get(ctx context.Context, prefix, path *pb.Path) (*pb.Notification, error)
	Delete(ctx context.Context, prefix, path *pb.Path) (*pb.UpdateResult, error)
	Update(ctx context.Context, prefix, path *pb.Path, val *pb.TypedValue) (*pb.UpdateResult, error)
	Replace(ctx context.Context, prefix, path *pb.Path, val *pb.TypedValue) (*pb.UpdateResult, error)
}

var _ GnmiRequestHandler = &NorthboundServerImpl{}

type NorthboundServerImpl struct {
	cfg       *ServeCfg
	converter *GnmiPathConverter
}

func NewNorthboundServerImpl(cfg *ServeCfg) *NorthboundServerImpl {
	return &NorthboundServerImpl{
		cfg:       cfg,
		converter: NewGnmiPathConverter(cfg),
	}
}

// Error shows an error with stacktrace if attached.
func (s *NorthboundServerImpl) Error(l *zap.SugaredLogger, err error, msg string, kvs ...interface{}) {
	l = l.WithOptions(zap.AddCallerSkip(1))
	if st := common.GetStackTrace(err); st != "" {
		l = l.With("stacktrace", st)
	}
	l.Errorw(fmt.Sprintf("%s: %v", msg, err), kvs...)
}

// Capabilities responds the server capabilities containing the available services.
func (s *NorthboundServerImpl) Capabilities(ctx context.Context, req *pb.CapabilityRequest) (*pb.CapabilityResponse, error) {
	l := logger.FromContext(ctx)

	ver, err := gnmi.GetGNMIServiceVersion()
	if err != nil {
		s.Error(l, err, "get gnmi service version")
		return nil, status.Errorf(codes.Internal, "failed to get gnmi service version: %v", err)
	}
	p := ServicePath{RootDir: s.cfg.RootPath}
	mlist, err := p.ReadServiceMetaAll()
	if err != nil {
		s.Error(l, err, "get gnmi service version")
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

// Get returns the service input stored at the supplied path.
func (s *NorthboundServerImpl) Get(ctx context.Context, prefix, path *pb.Path) (*pb.Notification, error) {
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

// Delete deletes the service input stored at the supplied path.
func (s *NorthboundServerImpl) Delete(ctx context.Context, prefix, path *pb.Path) (*pb.UpdateResult, error) {
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
		if !errors.Is(err, os.ErrNotExist) {
			s.Error(l, err, "delete file")
			return nil, status.Errorf(codes.Internal, "failed to delete file: %s", r.String())
		}
	}
	return &pb.UpdateResult{Path: path, Op: pb.UpdateResult_DELETE}, nil
}

// Replace replaces the service input stored at the supplied path.
func (s *NorthboundServerImpl) Replace(ctx context.Context, prefix, path *pb.Path, val *pb.TypedValue) (*pb.UpdateResult, error) {
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
	// TODO fix type conversion
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

// Update updates the service input stored at the supplied path.
// TODO test
func (s *NorthboundServerImpl) Update(ctx context.Context, prefix, path *pb.Path, val *pb.TypedValue) (*pb.UpdateResult, error) {
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

	cctx := cuecontext.New()
	sp := r.Path()

	// load current input
	buf, err := sp.ReadServiceInput()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, status.Errorf(codes.NotFound, "not found: %s", req.String())
		} else {
			s.Error(l, err, "open file")
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}
	curInputVal := cctx.CompileBytes(buf)

	curInput := map[string]any{}
	if err := curInputVal.Decode(&curInput); err != nil {
		return nil, status.Errorf(codes.Internal, "decode current input")
	}

	// merge current and new inputs
	input := map[string]any{}
	if err := json.Unmarshal(val.GetJsonVal(), &input); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode input: %s", r.String())
	}
	input = common.MergeMap(curInput, input)
	// TODO set primary keys
	//for k, v := range r.Keys() {
	//	input[k] = v
	//}

	inputVal := cctx.Encode(input)
	if inputVal.Err() != nil {
		return nil, status.Errorf(codes.Internal, "failed to encode to cue: %s", r.String())
	}

	b, err := FormatCue(inputVal, cue.Final())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to format cue to bytes: %s", r.String())
	}
	if err := sp.WriteServiceInputFile(b); err != nil {
		s.Error(l, err, "write service input")
		return nil, status.Errorf(codes.Internal, "failed to write service input: %s", req.String())
	}

	return &pb.UpdateResult{Path: path, Op: pb.UpdateResult_UPDATE}, nil
}

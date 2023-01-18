/*
 Copyright (c) 2022-2023 NTT Communications Corporation

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

package core

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/nttcom/kuesta/internal/gogit"
	"github.com/nttcom/kuesta/internal/util"
	"github.com/nttcom/kuesta/internal/validator"
	"github.com/nttcom/kuesta/pkg/common"
	kcue "github.com/nttcom/kuesta/pkg/cue"
	"github.com/nttcom/kuesta/pkg/kuesta"
	"github.com/nttcom/kuesta/pkg/logger"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type ServeCfg struct {
	RootCfg

	Addr            string `validate:"required"`
	SyncPeriod      int    `validate:"required"`
	PersistGitState bool
	NoTLS           bool
	Insecure        bool
	TLSCrtPath      string
	TLSKeyPath      string
	TLSCACrtPath    string
}

func (c *ServeCfg) TLSServerConfig() *common.TLSServerConfig {
	cfg := &common.TLSServerConfig{
		TLSConfigBase: common.TLSConfigBase{
			NoTLS:     c.NoTLS,
			CrtPath:   c.TLSCrtPath,
			KeyPath:   c.TLSKeyPath,
			CACrtPath: c.TLSCACrtPath,
		},
	}
	if c.Insecure {
		cfg.ClientAuth = tls.VerifyClientCertIfGiven
	} else {
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return cfg
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
	if !c.NoTLS {
		if c.TLSKeyPath == "" || c.TLSCrtPath == "" {
			return fmt.Errorf("tls-key and tls-crt options must be set to use TLS")
		}
	}
	if c.SyncPeriod < 10 {
		c.SyncPeriod = 10
	}
	return validator.Validate(c)
}

func RunServe(ctx context.Context, cfg *ServeCfg) error {
	l := logger.FromContext(ctx)
	l.Debug("serve called")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	credOpts, err := common.GRPCServerCredentials(cfg.TLSServerConfig())
	if err != nil {
		return fmt.Errorf("setup credentials: %w", err)
	}
	g := grpc.NewServer(credOpts...)
	s, err := NewNorthboundServer(cfg)
	if err != nil {
		return fmt.Errorf("init gNMI impl server: %w", err)
	}
	if err := s.cGit.Pull(); err != nil {
		return fmt.Errorf("git pull config repo: %w", err)
	}
	if err := s.sGit.Pull(); err != nil {
		return fmt.Errorf("git pull status repo: %w", err)
	}

	pb.RegisterGNMIServer(g, s)
	reflection.Register(g)

	dur := time.Duration(s.cfg.SyncPeriod) * time.Second
	s.RunConfigSyncLoop(ctx, dur)
	s.RunStatusSyncLoop(ctx, dur)

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
	smu  sync.Mutex   // smu is the lock to avoid git operation conflicts
	cfg  *ServeCfg
	cGit *gogit.Git
	sGit *gogit.Git
	impl GnmiRequestHandler
}

// NewNorthboundServer creates new NorthboundServer with supplied ServeCfg.
func NewNorthboundServer(cfg *ServeCfg) (*NorthboundServer, error) {
	cGit, err := gogit.NewGit(cfg.ConfigGitOptions().ShouldCloneIfNotExist())
	if err != nil {
		return nil, err
	}
	sGit, err := gogit.NewGit(cfg.StatusGitOptions().ShouldCloneIfNotExist())
	if err != nil {
		return nil, err
	}
	s := &NorthboundServer{
		cfg:  cfg,
		mu:   sync.RWMutex{},
		smu:  sync.Mutex{},
		cGit: cGit,
		sGit: sGit,
		impl: NewNorthboundServerImpl(cfg),
	}
	return s, nil
}

func NewNorthboundServerWithGit(cfg *ServeCfg, cGit, sGit *gogit.Git) *NorthboundServer {
	return &NorthboundServer{
		cfg:  cfg,
		mu:   sync.RWMutex{},
		smu:  sync.Mutex{},
		cGit: cGit,
		sGit: sGit,
		impl: NewNorthboundServerImpl(cfg),
	}
}

func (s *NorthboundServer) RunStatusSyncLoop(ctx context.Context, dur time.Duration) {
	syncStatusFunc := func() {
		if _, err := s.sGit.Checkout(); err != nil {
			logger.Error(ctx, err, "git checkout")
		}
		if err := s.sGit.Pull(); err != nil {
			logger.Error(ctx, err, "git pull")
		}
	}
	util.SetInterval(ctx, syncStatusFunc, dur, "sync from status repo")
}

func (s *NorthboundServer) RunConfigSyncLoop(ctx context.Context, dur time.Duration) {
	syncConfigFunc := func() {
		s.smu.Lock()
		defer s.smu.Unlock()
		if _, err := s.cGit.Checkout(); err != nil {
			logger.Error(ctx, err, "git checkout")
		}
		if err := s.cGit.Pull(); err != nil {
			logger.Error(ctx, err, "git pull")
		}
	}
	util.SetInterval(ctx, syncConfigFunc, dur, "sync from config repo")
}

// Error shows an error with stacktrace if attached.
func (s *NorthboundServer) Error(l *zap.SugaredLogger, err error, msg string, kvs ...interface{}) {
	l = l.WithOptions(zap.AddCallerSkip(1))
	if st := logger.GetStackTrace(err); st != "" {
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
	defer func() {
		s.mu.RUnlock()
	}()
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

	notifications[0].Timestamp = s.getCommitTimeOrNow()
	return &pb.GetResponse{Notification: notifications}, nil
}

// Set executes specified Replace/Update/Delete operations and responds what is done by SetRequest.
func (s *NorthboundServer) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	l := logger.FromContext(ctx)
	l.Debugw("Set called")

	if !s.mu.TryLock() {
		return nil, status.Error(codes.Unavailable, "locked")
	}
	s.smu.Lock()
	defer func() {
		if !s.cfg.PersistGitState {
			if err := s.cGit.Reset(gogit.ResetOptsHard()); err != nil {
				s.Error(l, err, "git reset")
			}
			if _, err := s.cGit.Checkout(); err != nil {
				s.Error(l, err, "git checkout")
			}
		}
		s.smu.Unlock()
		s.mu.Unlock()
	}()
	if err := s.cGit.Reset(gogit.ResetOptsHard()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to perform 'git reset --hard'")
	}
	if _, err := s.cGit.Checkout(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to perform 'git checkout' to %s", s.cfg.GitTrunk)
	}
	if err := s.cGit.Pull(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to perform 'git pull'")
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

	sp := kuesta.ServicePath{RootDir: s.cfg.ConfigRootPath}
	if err := s.cGit.Add(sp.ServiceDirPath(kuesta.ExcludeRoot)); err != nil {
		s.Error(l, err, "git add")
		return nil, status.Errorf(codes.Internal, "failed to git-add")
	}

	serviceApplyCfg := ServiceApplyCfg{RootCfg: s.cfg.RootCfg}
	if err := RunServiceApply(ctx, &serviceApplyCfg); err != nil {
		s.Error(l, err, "service apply")
		return nil, status.Errorf(codes.Internal, "failed to apply service template")
	}

	gitCommitCfg := GitCommitCfg{
		RootCfg: s.cfg.RootCfg,
	}
	if err := RunGitCommit(ctx, &gitCommitCfg); err != nil {
		s.Error(l, err, "git commit")
		return nil, status.Errorf(codes.Internal, "failed to git push to %s", s.cfg.GitTrunk)
	}

	return &pb.SetResponse{
		Prefix:    prefix,
		Response:  results,
		Timestamp: s.getCommitTimeOrNow(),
	}, nil
}

func (s *NorthboundServer) getCommitTimeOrNow() int64 {
	commit, err := s.cGit.Head()
	if err != nil {
		return time.Now().UnixNano()
	}
	return commit.Author.When.UnixNano()
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
	if st := logger.GetStackTrace(err); st != "" {
		l = l.With("stacktrace", st)
	}
	l.Errorw(fmt.Sprintf("%s: %v", msg, err), kvs...)
}

// Capabilities responds the server capabilities containing the available services.
func (s *NorthboundServerImpl) Capabilities(ctx context.Context, req *pb.CapabilityRequest) (*pb.CapabilityResponse, error) {
	l := logger.FromContext(ctx)

	ver, err := GetGNMIServiceVersion()
	if err != nil {
		s.Error(l, err, "get gnmi service version")
		return nil, status.Errorf(codes.Internal, "failed to get gnmi service version: %v", err)
	}
	mlist, err := kuesta.ReadServiceMetaAll(s.cfg.ConfigRootPath)
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
		GNMIVersion:        ver,
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
		buf, err = r.Path().ReadActualDeviceConfigFile()
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
	val, err := kcue.NewValueFromBytes(cctx, buf)
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
	if err = os.Remove(sp.ServiceInputPath(kuesta.IncludeRoot)); err != nil {
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
	sp := r.Path()

	// new input
	input := map[string]any{}
	if err := json.Unmarshal(val.GetJsonVal(), &input); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode input: %s", r.String())
	}

	// path keys
	transformer, err := sp.ReadServiceTransform(cctx)
	if err != nil {
		s.Error(l, err, "load transform file")
		return nil, status.Errorf(codes.Internal, "load transform file: %s", r.String())
	}
	convertedKeys, err := transformer.ConvertInputType(r.Keys())
	if err != nil {
		s.Error(l, err, "convert types of path keys")
		return nil, status.Errorf(codes.InvalidArgument, "convert types of path keys")
	}

	expr := kcue.NewAstExpr(util.MergeMap(input, convertedKeys))
	inputVal := cctx.BuildExpr(expr)
	if inputVal.Err() != nil {
		return nil, status.Errorf(codes.InvalidArgument, "encode to cue value: %v", inputVal.Err())
	}

	b, err := kcue.FormatCue(inputVal, cue.Final())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to format cue to bytes: %s", r.String())
	}
	if err := sp.WriteServiceInputFile(b); err != nil {
		s.Error(l, err, "write service input")
		return nil, status.Errorf(codes.Internal, "failed to write service input: %s", req.String())
	}

	return &pb.UpdateResult{Path: path, Op: pb.UpdateResult_REPLACE}, nil
}

// Update updates the service input stored at the supplied path.
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

	// current input
	buf, err := sp.ReadServiceInput()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, status.Errorf(codes.NotFound, "not found: %s", req.String())
		} else {
			s.Error(l, err, "open file")
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
	}
	curInputVal := cctx.CompileBytes(buf)

	curInput := map[string]any{}
	if err := curInputVal.Decode(&curInput); err != nil {
		return nil, status.Errorf(codes.Internal, "decode current input")
	}

	// new input
	input := map[string]any{}
	if err := json.Unmarshal(val.GetJsonVal(), &input); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode input: %s", r.String())
	}

	// path keys
	transformer, err := sp.ReadServiceTransform(cctx)
	if err != nil {
		s.Error(l, err, "load transform file")
		return nil, status.Errorf(codes.Internal, "load transform file: %s", r.String())
	}
	convertedKeys, err := transformer.ConvertInputType(r.Keys())
	if err != nil {
		s.Error(l, err, "convert types of path keys")
		return nil, status.Errorf(codes.InvalidArgument, "convert types of path keys")
	}

	expr := kcue.NewAstExpr(util.MergeMap(curInput, input, convertedKeys))
	inputVal := cctx.BuildExpr(expr)
	if inputVal.Err() != nil {
		return nil, status.Errorf(codes.Internal, "failed to encode to cue: %s", r.String())
	}

	b, err := kcue.FormatCue(inputVal, cue.Final())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to format cue to bytes: %s", r.String())
	}
	if err := sp.WriteServiceInputFile(b); err != nil {
		s.Error(l, err, "write service input")
		return nil, status.Errorf(codes.Internal, "failed to write service input: %s", req.String())
	}

	return &pb.UpdateResult{Path: path, Op: pb.UpdateResult_UPDATE}, nil
}

// GetGNMIServiceVersion returns a pointer to the gNMI service version string.
// The method is non-trivial because of the way it is defined in the proto file.
func GetGNMIServiceVersion() (string, error) {
	gzB, _ := (&pb.Update{}).Descriptor() // nolint
	r, err := gzip.NewReader(bytes.NewReader(gzB))
	if err != nil {
		return "", fmt.Errorf("error in initializing gzip reader: %w", err)
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("error in reading gzip data: %w", err)
	}
	desc := &descriptor.FileDescriptorProto{}
	if err := proto.Unmarshal(b, desc); err != nil {
		return "", fmt.Errorf("error in unmarshaling proto: %w", err)
	}
	ver := proto.GetExtension(desc.Options, pb.E_GnmiService)
	return (ver).(string), nil
}

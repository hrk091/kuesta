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

package gnmi

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

type GnmiMock struct {
	pb.UnimplementedGNMIServer
	CapabilitiesHandler func(context.Context, *pb.CapabilityRequest) (*pb.CapabilityResponse, error)
	GetHandler          func(context.Context, *pb.GetRequest) (*pb.GetResponse, error)
	SetHandler          func(context.Context, *pb.SetRequest) (*pb.SetResponse, error)
	SubscribeHandler    func(pb.GNMI_SubscribeServer) error
}

func (s *GnmiMock) Capabilities(ctx context.Context, r *pb.CapabilityRequest) (*pb.CapabilityResponse, error) {
	if s.CapabilitiesHandler == nil {
		return s.UnimplementedGNMIServer.Capabilities(ctx, r)
	}
	return s.CapabilitiesHandler(ctx, r)
}

func (s *GnmiMock) Get(ctx context.Context, r *pb.GetRequest) (*pb.GetResponse, error) {
	if s.GetHandler == nil {
		return s.UnimplementedGNMIServer.Get(ctx, r)
	}
	return s.GetHandler(ctx, r)
}

func (s *GnmiMock) Set(ctx context.Context, r *pb.SetRequest) (*pb.SetResponse, error) {
	if s.SetHandler == nil {
		return s.UnimplementedGNMIServer.Set(ctx, r)
	}
	return s.SetHandler(ctx, r)
}

func (s *GnmiMock) Subscribe(stream pb.GNMI_SubscribeServer) error {
	if s.SubscribeHandler == nil {
		return s.UnimplementedGNMIServer.Subscribe(stream)
	}
	return s.SubscribeHandler(stream)
}

func NewServer(ctx context.Context, s pb.GNMIServer, opts ...grpc.DialOption) (*grpc.Server, *grpc.ClientConn) {
	lis := bufconn.Listen(bufSize)
	g := grpc.NewServer()

	pb.RegisterGNMIServer(g, s)

	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}
	opts = append([]grpc.DialOption{grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials())}, opts...)
	conn, err := grpc.DialContext(ctx, "bufnet", opts...)
	if err != nil {
		panic(err)
	}

	go func() {
		if err := g.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()
	return g, conn
}

func NewServerWithListener(s pb.GNMIServer, lis net.Listener) *grpc.Server {
	g := grpc.NewServer()
	pb.RegisterGNMIServer(g, s)
	go func() {
		if err := g.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()
	return g
}

// GetGNMIServiceVersion returns a pointer to the gNMI service version string.
// The method is non-trivial because of the way it is defined in the proto file.
func GetGNMIServiceVersion() (*string, error) {
	gzB, _ := (&pb.Update{}).Descriptor()
	r, err := gzip.NewReader(bytes.NewReader(gzB))
	if err != nil {
		return nil, fmt.Errorf("error in initializing gzip reader: %v", err)
	}
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error in reading gzip data: %v", err)
	}
	desc := &dpb.FileDescriptorProto{}
	if err := proto.Unmarshal(b, desc); err != nil {
		return nil, fmt.Errorf("error in unmarshaling proto: %v", err)
	}
	ver, err := proto.GetExtension(desc.Options, pb.E_GnmiService)
	if err != nil {
		return nil, fmt.Errorf("error in getting version from proto extension: %v", err)
	}
	return ver.(*string), nil
}

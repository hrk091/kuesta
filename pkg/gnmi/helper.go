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

package gnmi

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"io/ioutil"
	"log"
	"net"
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

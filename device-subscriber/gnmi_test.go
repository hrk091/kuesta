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

package main

import (
	"context"
	"github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"log"
	"net"
	"testing"
)

const bufSize = 1024 * 1024

type gnmiMock struct {
	pb.UnimplementedGNMIServer
	getHandler func(context.Context, *pb.GetRequest) (*pb.GetResponse, error)
}

func (s *gnmiMock) Get(ctx context.Context, r *pb.GetRequest) (*pb.GetResponse, error) {
	return s.getHandler(ctx, r)
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

func TestGetEntireConfig(t *testing.T) {
	config := []byte("dummy")

	tests := []struct {
		name    string
		handler func(ctx context.Context, request *pb.GetRequest) (*pb.GetResponse, error)
		wantErr bool
	}{
		{
			"ok",
			func(ctx context.Context, request *pb.GetRequest) (*pb.GetResponse, error) {
				v := gnmi.TypedValue{
					Value: &gnmi.TypedValue_JsonIetfVal{
						JsonIetfVal: config,
					},
				}
				resp := &pb.GetResponse{
					Notification: []*pb.Notification{
						{
							Update: []*pb.Update{
								{Path: &pb.Path{}, Val: &v},
							},
						},
					},
				}
				return resp, nil
			},
			false,
		},
		{
			"bad: no content",
			func(ctx context.Context, request *pb.GetRequest) (*pb.GetResponse, error) {
				resp := &pb.GetResponse{
					Notification: []*pb.Notification{},
				}
				return resp, nil
			},
			true,
		},
		{
			"bad: error response",
			func(ctx context.Context, request *pb.GetRequest) (*pb.GetResponse, error) {
				return nil, status.Error(codes.Internal, "error")
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &gnmiMock{
				getHandler: tt.handler,
			}
			ctx := context.Background()
			s, conn := NewServer(ctx, m)
			defer s.Stop()

			c, err := gnmiclient.NewFromConn(ctx, conn, client.Destination{})
			got, err := GetEntireConfig(ctx, c)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, config, got)
			}
		})
	}

}

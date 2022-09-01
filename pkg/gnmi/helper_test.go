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

package gnmi_test

import (
	"context"
	"github.com/hrk091/nwctl/pkg/gnmi"
	gclient "github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

func TestGetGNMIServiceVersion(t *testing.T) {
	ver, err := gnmi.GetGNMIServiceVersion()
	assert.Nil(t, err)
	re := regexp.MustCompile(`(\d+)(\.\d+)?(\.\d+)?`)
	assert.NotNil(t, re.FindStringIndex(*ver))
}

func TestNewServer(t *testing.T) {
	getCalled := false
	setCalled := false
	capabilitiesCalled := false
	subscribeCalled := false
	m := &gnmi.GnmiMock{
		GetHandler: func(ctx context.Context, request *pb.GetRequest) (*pb.GetResponse, error) {
			getCalled = true
			return &pb.GetResponse{}, nil
		},
		SetHandler: func(ctx context.Context, request *pb.SetRequest) (*pb.SetResponse, error) {
			setCalled = true
			return &pb.SetResponse{}, nil
		},
		CapabilitiesHandler: func(ctx context.Context, request *pb.CapabilityRequest) (*pb.CapabilityResponse, error) {
			capabilitiesCalled = true
			return &pb.CapabilityResponse{}, nil
		},
		SubscribeHandler: func(stream pb.GNMI_SubscribeServer) error {
			subscribeCalled = true
			_ = stream.Send(&pb.SubscribeResponse{})
			return nil
		},
	}
	ctx := context.Background()
	gs, conn := gnmi.NewServer(ctx, m)
	defer gs.Stop()

	client, err := gnmiclient.NewFromConn(ctx, conn, gclient.Destination{})
	if err != nil {
		t.Fatal(err)
	}

	client.Get(ctx, &pb.GetRequest{})
	client.Set(ctx, &pb.SetRequest{})
	client.Capabilities(ctx, &pb.CapabilityRequest{})
	q := gclient.Query{
		Type:                gclient.Stream,
		NotificationHandler: nil,
	}
	client.Subscribe(ctx, q)

	assert.True(t, getCalled)
	assert.True(t, setCalled)
	assert.True(t, capabilitiesCalled)
	assert.True(t, subscribeCalled)
}

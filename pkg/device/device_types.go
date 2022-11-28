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

package v1alpha1

import (
	"fmt"
	gnmiclient "github.com/openconfig/gnmi/client"
	"time"
)

const (
	RefField = ".spec.rolloutRef"
)

type DeviceResource interface {
	nwctlDevice()

	SpecCopy() *DeviceSpec
	UpdateSpec(func(*DeviceSpec) error) error
	StatusCopy() *DeviceStatus
	UpdateStatus(func(*DeviceStatus) error) error
}

var _ DeviceResource = &Device{}

type Device struct {
	Spec   DeviceSpec   `json:"spec,omitempty"`
	Status DeviceStatus `json:"status,omitempty"`
}

func (Device) nwctlDevice() {}

func (d *Device) SpecCopy() *DeviceSpec {
	return d.Spec.DeepCopy()
}

func (d *Device) StatusCopy() *DeviceStatus {
	return d.Status.DeepCopy()
}

func (d *Device) UpdateSpec(fn func(*DeviceSpec) error) error {
	return fn(&d.Spec)
}

func (d *Device) UpdateStatus(fn func(*DeviceStatus) error) error {
	return fn(&d.Status)
}

// DeviceSpec defines the basic specs required to manage target device.
type DeviceSpec struct {

	// RolloutRef is the name of DeviceRollout to which this device belongs.
	RolloutRef string `json:"rolloutRef"`

	// BaseRevision is the git revision to assume that the device config of the specified version has been already provisioned.
	BaseRevision string `json:"baseRevision,omitempty"`

	ConnectionInfo `json:",inline"`
}

func (s *DeviceSpec) GnmiDestination() gnmiclient.Destination {
	return gnmiclient.Destination{
		Addrs:       []string{fmt.Sprintf("%s:%d", s.Address, s.Port)},
		Target:      "",
		Timeout:     10 * time.Second,
		Credentials: s.GnmiCredentials(),
	}
}

// DeviceStatus defines the observed state of OcDemo
type DeviceStatus struct {

	// Checksum is a hash to uniquely identify the entire device config.
	Checksum string `json:"checksum,omitempty"`

	// LastApplied is the device config applied at the previous transaction.
	LastApplied []byte `json:"lastApplied,omitempty"`

	// BaseRevision is the git revision to assume that the device config of the specified version has been already provisioned.
	BaseRevision string `json:"baseRevision,omitempty"`
}

// ConnectionInfo defines the parameters to connect target device.
type ConnectionInfo struct {
	Address  string `json:"address,omitempty"`
	Port     uint16 `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (d *ConnectionInfo) GnmiCredentials() *gnmiclient.Credentials {
	if d.Username == "" || d.Password == "" {
		return nil
	}

	return &gnmiclient.Credentials{
		Username: d.Username,
		Password: d.Password,
	}
}

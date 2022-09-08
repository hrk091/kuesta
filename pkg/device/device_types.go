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

package v1alpha1

import (
	gnmiclient "github.com/openconfig/gnmi/client"
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

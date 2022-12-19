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
	"github.com/nttcom/kuesta/pkg/common"
	gnmiclient "github.com/openconfig/gnmi/client"
	core "k8s.io/api/core/v1"
	"time"
)

const (
	RefField = ".spec.rolloutRef"
)

type DeviceResource interface {
	kuestaDevice()

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

func (Device) kuestaDevice() {}

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

	TLS TLSSpec `json:"tls,omitempty"`
}

func (s *DeviceSpec) GnmiDestination(sData map[string][]byte) (gnmiclient.Destination, error) {
	dest := gnmiclient.Destination{
		Addrs:       []string{fmt.Sprintf("%s:%d", s.Address, s.Port)},
		Target:      "",
		Timeout:     10 * time.Second,
		Credentials: s.GnmiCredentials(),
	}
	if s.TLS.NoTLS {
		return dest, nil
	}
	clientCfg := s.TLS.TLSClientConfig(sData)
	tlsCfg, err := common.NewTLSConfig(clientCfg.Certificates(false), clientCfg.VerifyServer())
	if err != nil {
		return gnmiclient.Destination{}, fmt.Errorf("new tls config: %w", err)
	}
	dest.TLS = tlsCfg

	return dest, nil
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

type TLSSpec struct {
	NoTLS bool `json:"notls,omitempty"`

	// Skip verifying server cert
	SkipVerifyServer bool `json:"skipVerify,omitempty"`

	// Path to the cert file
	SecretName string `json:"secretName,omitempty"`

	// To verify the server hostname
	ServerName string `json:"serverName,omitempty"`
}

func (c *TLSSpec) TLSClientConfig(secretData map[string][]byte) *common.TLSClientConfig {
	crtData := secretData[core.TLSCertKey]
	keyData := secretData[core.TLSPrivateKeyKey]
	caCrtData := secretData[core.ServiceAccountRootCAKey]

	return &common.TLSClientConfig{
		TLSConfigBase: common.TLSConfigBase{
			NoTLS:     c.NoTLS,
			CrtData:   crtData,
			KeyData:   keyData,
			CACrtData: caCrtData,
		},
		SkipVerifyServer: c.SkipVerifyServer,
		ServerName:       c.ServerName,
	}
}

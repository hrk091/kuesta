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

package device

import (
	gnmiclient "github.com/openconfig/gnmi/client"
)

// DeviceSpec defines the basic specs required to manage target device.
type DeviceSpec struct {

	// RolloutRef is the name of DeviceRollout to which this device belongs.
	RolloutRef string `json:"rolloutRef"`

	ConnectionInfo `json:",inline"`
}

// DeviceStatus defines the observed state of OcDemo
type DeviceStatus struct {

	// Checksum is a hash to uniquely identify the entire device config.
	Checksum string `json:"checksum"`
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

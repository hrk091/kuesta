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

package nwctl

import (
	"fmt"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
)

type PathReq interface {
	Type() PathType
}

type ServicePathReq struct {
	path *ServicePath
}

func (ServicePathReq) Type() PathType {
	return PathTypeService
}

func (s *ServicePathReq) Path() *ServicePath {
	return s.path
}

type DevicePathReq struct {
	path *DevicePath
}

func (DevicePathReq) Type() PathType {
	return PathTypeDevice
}

func (s DevicePathReq) Path() *DevicePath {
	return s.path
}

type GnmiPathConverter struct {
	cfg *ServeCfg

	// meta caches service metadata
	// TODO clear cache periodically
	meta map[string]*ServiceMeta
}

func NewGnmiPathConverter(cfg *ServeCfg) *GnmiPathConverter {
	return &GnmiPathConverter{
		cfg:  cfg,
		meta: map[string]*ServiceMeta{},
	}
}

// Convert converts gNMI Path to PathReq.
func (c *GnmiPathConverter) Convert(prefix, path *gnmi.Path) (PathReq, error) {
	path = gnmiFullPath(prefix, path)
	elem := path.GetElem()
	if len(elem) < 2 {
		return nil, errors.WithStack(fmt.Errorf("path must have at least 2 elem"))
	}
	kindEl := elem[0]
	switch kindEl.GetName() {
	case DirServices:
		return c.convertService(elem[1:])
	case DirDevices:
		return c.convertDevice(elem[1:])
	default:
		return nil, errors.WithStack(fmt.Errorf("name of the first elem must be `%s` or `%s`", DirServices, DirDevices))
	}
}

func (c *GnmiPathConverter) convertService(elem []*gnmi.PathElem) (ServicePathReq, error) {
	svcEl := elem[0]
	if svcEl.GetName() != NodeService {
		return ServicePathReq{}, errors.WithStack(fmt.Errorf("name of second elem must be `%s`", NodeService))
	}
	keys := svcEl.GetKey()
	svcKind, ok := keys[KeyServiceKind]
	if !ok {
		return ServicePathReq{}, errors.WithStack(fmt.Errorf("`%s` key is required for service path", KeyServiceKind))
	}
	p := ServicePath{RootDir: c.cfg.RootPath, Service: svcKind}

	meta, ok := c.meta[svcKind]
	if !ok {
		m, err := p.ReadServiceMeta()
		if err != nil {
			return ServicePathReq{}, err
		}
		c.meta[svcKind] = m
		meta = m
	}

	for _, k := range meta.Keys {
		if v, ok := keys[k]; ok == true {
			p.Keys = append(p.Keys, v)
		} else {
			return ServicePathReq{}, errors.WithStack(fmt.Errorf("key `%s` of service %s is not supplied in Request Path", k, svcKind))
		}
	}
	return ServicePathReq{path: &p}, nil
}

func (c *GnmiPathConverter) convertDevice(elem []*gnmi.PathElem) (DevicePathReq, error) {
	svcEl := elem[0]
	if svcEl.GetName() != NodeDevice {
		return DevicePathReq{}, errors.WithStack(fmt.Errorf("name of second elem must be `%s`", NodeDevice))
	}
	keys := svcEl.GetKey()
	deviceName, ok := keys[KeyDeviceName]
	if !ok {
		return DevicePathReq{}, errors.WithStack(fmt.Errorf("`%s` key is required for service path", KeyDeviceName))
	}

	p := DevicePath{RootDir: c.cfg.RootPath, Device: deviceName}
	return DevicePathReq{path: &p}, nil
}

func gnmiFullPath(prefix, path *gnmi.Path) *gnmi.Path {
	fullPath := &gnmi.Path{}
	if path.GetElem() != nil {
		fullPath.Elem = append(prefix.GetElem(), path.GetElem()...)
	}
	return fullPath
}
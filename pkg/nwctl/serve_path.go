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

package nwctl

import (
	"fmt"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
)

type PathReq interface {
	Type() PathType
	String() string
}

type ServicePathReq struct {
	path    *ServicePath
	service string
	keys    map[string]string
}

func (ServicePathReq) Type() PathType {
	return PathTypeService
}

func (s ServicePathReq) String() string {
	return s.path.ServicePath(ExcludeRoot)
}

func (s *ServicePathReq) Path() *ServicePath {
	return s.path
}

func (s *ServicePathReq) Keys() map[string]string {
	return s.keys
}

type DevicePathReq struct {
	path   *DevicePath
	device string
}

func (DevicePathReq) Type() PathType {
	return PathTypeDevice
}

func (s DevicePathReq) String() string {
	return s.path.DevicePath(ExcludeRoot)
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
	elemKey := svcEl.GetKey()
	svcKind, ok := elemKey[KeyServiceKind]
	if !ok {
		return ServicePathReq{}, errors.WithStack(fmt.Errorf("`%s` key is required for service path", KeyServiceKind))
	}
	p := ServicePath{RootDir: c.cfg.ConfigRootPath, Service: svcKind}

	meta, ok := c.meta[svcKind]
	if !ok {
		m, err := p.ReadServiceMeta()
		if err != nil {
			return ServicePathReq{}, err
		}
		c.meta[svcKind] = m
		meta = m
	}

	keys := map[string]string{}
	for _, k := range meta.Keys {
		if v, ok := elemKey[k]; ok == true {
			keys[k] = v
			p.Keys = append(p.Keys, v)
		} else {
			return ServicePathReq{}, errors.WithStack(fmt.Errorf("key `%s` of service %s is not supplied in Request Path", k, svcKind))
		}
	}
	return ServicePathReq{path: &p, service: svcKind, keys: keys}, nil
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

	p := DevicePath{RootDir: c.cfg.StatusRootPath, Device: deviceName}
	return DevicePathReq{path: &p, device: deviceName}, nil
}

func gnmiFullPath(prefix, path *gnmi.Path) *gnmi.Path {
	fullPath := &gnmi.Path{}
	if path.GetElem() != nil {
		fullPath.Elem = append(prefix.GetElem(), path.GetElem()...)
	}
	return fullPath
}

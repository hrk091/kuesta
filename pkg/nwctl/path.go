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
	"crypto/sha256"
	"fmt"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	DirServices         = "services"
	DirDevices          = "devices"
	DirComputed         = "computed"
	FileInputCue        = "input.cue"
	FileTransformCue    = "transform.cue"
	FileConfigCue       = "config.cue"
	FileActualConfigCue = "actual_config.cue"
)

type PathType string

const (
	ExcludeRoot PathType = ""
	IncludeRoot PathType = "INCLUDE_ROOT"
)

var (
	_sep = string(filepath.Separator)
)

type ServicePath struct {
	RootDir string `validate:"required"`

	Service string
	Keys    []string `validate:"dive,required"`
}

// Validate validates exposed fields according to the `validate` tag.
func (p *ServicePath) Validate() error {
	return common.Validate(p)
}

// RootPath returns the path to repository root.
func (p *ServicePath) RootPath() string {
	return filepath.FromSlash(p.RootDir)
}

func (p *ServicePath) serviceDirElem() []string {
	return []string{DirServices}
}

func (p *ServicePath) servicePathElem() []string {
	return append(p.serviceDirElem(), p.Service)
}

func (p *ServicePath) serviceItemPathElem() []string {
	return append(p.servicePathElem(), p.Keys...)
}

func (p *ServicePath) serviceComputedPathElem() []string {
	return append(p.serviceItemPathElem(), DirComputed)
}

func (p *ServicePath) addRoot(path string, t PathType) string {
	if t == ExcludeRoot {
		return path
	} else {
		return filepath.Join(p.RootPath(), path)
	}
}

// ServiceDirPath returns the path to the service directory.
func (p *ServicePath) ServiceDirPath(t PathType) string {
	return p.addRoot(filepath.Join(p.serviceDirElem()...), t)
}

// ServiceInputPath returns the path to the specified service's input file.
func (p *ServicePath) ServiceInputPath(t PathType) string {
	el := append(p.serviceItemPathElem(), FileInputCue)
	return p.addRoot(filepath.Join(el...), t)
}

// ReadServiceInput loads the specified service's input file.
func (p *ServicePath) ReadServiceInput() ([]byte, error) {
	buf, err := os.ReadFile(p.ServiceInputPath(IncludeRoot))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf, nil
}

// ServiceTransformPath returns the path to the specified service's transform file.
func (p *ServicePath) ServiceTransformPath(t PathType) string {
	el := append(p.servicePathElem(), FileTransformCue)
	return p.addRoot(filepath.Join(el...), t)
}

// ReadServiceTransform loads the specified service's transform file.
func (p *ServicePath) ReadServiceTransform() ([]byte, error) {
	buf, err := os.ReadFile(p.ServiceTransformPath(IncludeRoot))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf, nil
}

// ServiceComputedDirPath returns the path to the specified service's computed dir.
func (p *ServicePath) ServiceComputedDirPath(t PathType) string {
	return p.addRoot(filepath.Join(p.serviceComputedPathElem()...), t)
}

// ServiceComputedPath returns the path to the specified service's computed result of given device.
func (p *ServicePath) ServiceComputedPath(device string, t PathType) string {
	el := append(p.serviceComputedPathElem(), fmt.Sprintf("%s.cue", device))
	return p.addRoot(filepath.Join(el...), t)
}

// ReadServiceComputedFile loads the partial device config computed from specified service.
func (p *ServicePath) ReadServiceComputedFile(device string) ([]byte, error) {
	buf, err := os.ReadFile(p.ServiceComputedPath(device, IncludeRoot))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf, nil
}

// WriteServiceComputedFile writes the partial device config computed from service to the corresponding computed dir.
func (p *ServicePath) WriteServiceComputedFile(device string, buf []byte) error {
	return WriteFileWithMkdir(p.ServiceComputedPath(device, IncludeRoot), buf)
}

type DevicePath struct {
	RootDir string `validate:"required"`

	Device string
}

// NewDevicePathList returns the slice of DevicePath placed in the given root dir.
func NewDevicePathList(dir string) ([]*DevicePath, error) {
	dp := DevicePath{RootDir: dir}
	path := dp.DeviceDirPath(IncludeRoot)
	devices, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read devices dir: %w", err)
	}
	var dpList []*DevicePath
	for _, d := range devices {
		if !d.IsDir() {
			continue
		}
		dpList = append(dpList, &DevicePath{RootDir: dir, Device: d.Name()})
	}
	return dpList, nil
}

// Validate validates exposed fields according to the `validate` tag.
func (p *DevicePath) Validate() error {
	return common.Validate(p)
}

// RootPath returns the path to repository root.
func (p *DevicePath) RootPath() string {
	return filepath.FromSlash(p.RootDir)
}

func (p *DevicePath) deviceDirElem() []string {
	return []string{DirDevices}
}

func (p *DevicePath) devicePathElem() []string {
	return append(p.deviceDirElem(), p.Device)
}

func (p *DevicePath) addRoot(path string, t PathType) string {
	if t == ExcludeRoot {
		return path
	} else {
		return filepath.Join(p.RootPath(), path)
	}
}

// DeviceDirPath returns the path to the devices directory.
func (p *DevicePath) DeviceDirPath(t PathType) string {
	return p.addRoot(filepath.Join(p.deviceDirElem()...), t)
}

// DeviceConfigPath returns the path to specified device config.
func (p *DevicePath) DeviceConfigPath(t PathType) string {
	el := append(p.devicePathElem(), FileConfigCue)
	return p.addRoot(filepath.Join(el...), t)
}

// ReadDeviceConfigFile loads the device config.
func (p *DevicePath) ReadDeviceConfigFile() ([]byte, error) {
	buf, err := os.ReadFile(p.DeviceConfigPath(IncludeRoot))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf, nil
}

// WriteDeviceConfigFile writes the merged device config to the corresponding device dir.
func (p *DevicePath) WriteDeviceConfigFile(buf []byte) error {
	return WriteFileWithMkdir(p.DeviceConfigPath(IncludeRoot), buf)
}

// CheckSum returns the SHA256 checksum of the device config.
func (p *DevicePath) CheckSum() (string, error) {
	f, err := os.Open(p.DeviceConfigPath(IncludeRoot))
	if err != nil {
		return "", errors.WithStack(err)
	}
	hasher := sha256.New()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return "", errors.WithStack(err)
	}
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))
	return checksum, nil
}

// DeviceActualConfigPath returns the path to specified device actual config.
func (p *DevicePath) DeviceActualConfigPath(t PathType) string {
	el := append(p.devicePathElem(), FileActualConfigCue)
	return p.addRoot(filepath.Join(el...), t)
}

// ReadActualDeviceConfigFile loads the device actual config.
func (p *DevicePath) ReadActualDeviceConfigFile() ([]byte, error) {
	buf, err := os.ReadFile(p.DeviceActualConfigPath(IncludeRoot))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf, nil
}

// ParseServiceInputPath parses service model `input.cue` filepath and returns its service and keys.
func ParseServiceInputPath(path string) (string, []string, error) {
	if !isServiceInputPath(path) {
		return "", nil, errors.WithStack(fmt.Errorf("invalid service input path: %s", path))
	}
	dir, _ := filepath.Split(path)
	dirElem := strings.Split(strings.TrimRight(dir, _sep), _sep)
	return dirElem[1], dirElem[2:], nil
}

func isServiceInputPath(path string) bool {
	dir, file := filepath.Split(path)
	dirElem := strings.Split(dir, string(filepath.Separator))
	if dirElem[0] != "services" {
		return false
	}
	if file != "input.cue" {
		return false
	}
	return true
}

// ParseServiceComputedFilePath parses service computed filepath and returns its device name.
func ParseServiceComputedFilePath(path string) (string, error) {
	if !isServiceComputedFilePath(path) {
		return "", errors.WithStack(fmt.Errorf("invalid service computed path: %s", path))
	}
	return getFileNameNoExt(path), nil
}

func isServiceComputedFilePath(path string) bool {
	dir, _ := filepath.Split(path)
	dirElem := strings.Split(strings.TrimRight(dir, _sep), _sep)
	if dirElem[0] != DirServices {
		return false
	}
	if dirElem[len(dirElem)-1] != DirComputed {
		return false
	}
	return true
}

func getFileNameNoExt(path string) string {
	return filepath.Base(path[:len(path)-len(filepath.Ext(path))])
}

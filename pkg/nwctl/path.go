package nwctl

import (
	"fmt"
	"github.com/hrk091/nwctl/pkg/common"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	DirServices      = "services"
	DirDevices       = "devices"
	DirComputed      = "computed"
	FileInputCue     = "input.cue"
	FileTransformCue = "transform.cue"
	FileConfigCue    = "config.cue"
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

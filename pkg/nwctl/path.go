package nwctl

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	DirServices      = "services"
	DirDevices       = "devices"
	DirComputed      = "computed"
	FileInputCue     = "input.cue"
	FileTransformCue = "transform.cue"
)

type PathType string

const (
	ExcludeRoot PathType = ""
	IncludeRoot PathType = "INCLUDE_ROOT"
)

type ServicePath struct {
	RootDir string `validate:"required"`

	Service string   `validate:"required"`
	Keys    []string `validate:"gt=0,dive,required"`
}

// Validate validates exposed fields according to the `validate` tag.
func (p *ServicePath) Validate() error {
	return validate(p)
}

// RootPath returns the path to repository root.
func (p *ServicePath) RootPath() string {
	return filepath.FromSlash(p.RootDir)
}

func (p *ServicePath) servicePathElem() []string {
	return []string{DirServices, p.Service}
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

// ServiceInputPath returns path to the specified service's input file.
func (p *ServicePath) ServiceInputPath(t PathType) string {
	el := append(p.serviceItemPathElem(), FileInputCue)
	return p.addRoot(filepath.Join(el...), t)
}

// ReadServiceInput loads the specified service's input file.
func (p *ServicePath) ReadServiceInput() ([]byte, error) {
	buf, err := os.ReadFile(p.ServiceInputPath(IncludeRoot))
	if err != nil {
		return nil, err
	}
	return buf, err
}

// ServiceTransformPath returns path to the specified service's transform file.
func (p *ServicePath) ServiceTransformPath(t PathType) string {
	el := append(p.servicePathElem(), FileTransformCue)
	return p.addRoot(filepath.Join(el...), t)
}

// ReadServiceTransform loads the specified service's transform file.
func (p *ServicePath) ReadServiceTransform() ([]byte, error) {
	buf, err := os.ReadFile(p.ServiceTransformPath(IncludeRoot))
	if err != nil {
		return nil, err
	}
	return buf, err
}

// ServiceComputedDirPath returns path to the specified service's computed dir.
func (p *ServicePath) ServiceComputedDirPath(t PathType) string {
	return p.addRoot(filepath.Join(p.serviceComputedPathElem()...), t)
}

// ServiceComputedPath returns path to the specified service's computed result of given device.
func (p *ServicePath) ServiceComputedPath(device string, t PathType) string {
	el := append(p.serviceComputedPathElem(), fmt.Sprintf("%s.cue", device))
	return p.addRoot(filepath.Join(el...), t)
}

// WriteServiceComputedFile writes partial device config computed from service to the corresponding computed dir.
func (p *ServicePath) WriteServiceComputedFile(device string, buf []byte) error {
	return WriteFileWithMkdir(p.ServiceComputedPath(device, IncludeRoot), buf)
}

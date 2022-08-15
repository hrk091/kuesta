package nwctl

import (
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

func (p *ServicePath) addRoot(path string, t PathType) string {
	if t == ExcludeRoot {
		return path
	} else {
		return filepath.Join(p.RootPath(), path)
	}
}

// ServiceInputPath returns the path to specified service's input file.
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

// ServiceTransformPath returns the path to specified service's transform file.
func (p *ServicePath) ServiceTransformPath(t PathType) string {
	el := append(p.servicePathElem(), FileTransformCue)
	return p.addRoot(filepath.Join(el...), t)
}

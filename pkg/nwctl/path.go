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
	return append([]string{p.RootPath(), DirServices, p.Service}, p.Keys...)
}

// ServiceInputPath returns the path to specified service's input file.
func (p *ServicePath) ServiceInputPath() string {
	el := append(p.servicePathElem(), FileInputCue)
	return filepath.Join(el...)
}

// ReadServiceInput loads the specified service's input file.
func (p *ServicePath) ReadServiceInput() ([]byte, error) {
	buf, err := os.ReadFile(p.ServiceInputPath())
	if err != nil {
		return nil, err
	}
	return buf, err
}

// ServiceTransformPath returns the path to specified service's transform file.
func (p *ServicePath) ServiceTransformPath() string {
	el := append(p.servicePathElem(), FileTransformCue)
	return filepath.Join(el...)
}

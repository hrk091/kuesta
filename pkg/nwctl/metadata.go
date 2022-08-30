package nwctl

import (
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

type ServiceMeta struct {
	Name         string   `json:"name,omitempty"`         // Name of the model.
	Organization string   `json:"organization,omitempty"` // Organization publishing the model.
	Version      string   `json:"version,omitempty"`      // Semantic version of the model.
	Keys         []string `json:"keys"`
}

func (m *ServiceMeta) ModelData() *pb.ModelData {
	return &pb.ModelData{
		Name:         m.Name,
		Organization: m.Organization,
		Version:      m.Version,
	}
}

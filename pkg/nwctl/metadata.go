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
	"encoding/json"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
	"os"
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

func ReadServiceMeta(service, path string) (*ServiceMeta, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var meta ServiceMeta
	if err := json.Unmarshal(buf, &meta); err != nil {
		return nil, errors.WithStack(err)
	}
	meta.Name = service
	return &meta, nil
}

type ServiceTransformer struct {
	cue []byte
}

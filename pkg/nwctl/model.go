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
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"encoding/json"
	"fmt"
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

// ModelData returns the gnmi.ModelData.
func (m *ServiceMeta) ModelData() *pb.ModelData {
	return &pb.ModelData{
		Name:         m.Name,
		Organization: m.Organization,
		Version:      m.Version,
	}
}

// ReadServiceMeta returns ServiceMeta loaded from the metadata file on the given path.
func ReadServiceMeta(path string) (*ServiceMeta, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var meta ServiceMeta
	if err := json.Unmarshal(buf, &meta); err != nil {
		return nil, errors.WithStack(err)
	}
	return &meta, nil
}

type ServiceTransformer struct {
	value cue.Value
}

// NewServiceTransformer creates ServiceTransformer with cue build instance.
func NewServiceTransformer(cctx *cue.Context, filepaths []string, dir string) (*ServiceTransformer, error) {
	v, err := NewValueWithInstance(cctx, filepaths, &load.Config{Dir: dir})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &ServiceTransformer{value: v}, nil
}

// Value returns the cue value contained.
func (t *ServiceTransformer) Value() cue.Value {
	return t.value
}

// Apply performs cue evaluation of transform.cue using given input.
// It returns cue.Iterator which iterates items including device name label and device config cue.Value.
func (t *ServiceTransformer) Apply(input cue.Value) (*cue.Iterator, error) {
	cctx := t.value.Context()
	template := cctx.CompileString(cueTypeStrTemplate, cue.Scope(t.value))
	if template.Err() != nil {
		return nil, errors.WithStack(template.Err())
	}
	filled := template.FillPath(cue.ParsePath(cuePathInput), input)
	if filled.Err() != nil {
		return nil, errors.WithStack(filled.Err())
	}

	filledIn := filled.LookupPath(cue.ParsePath(cuePathOutput))
	if err := filledIn.Validate(cue.Concrete(true)); err != nil {
		return nil, errors.WithStack(err)
	}
	out := filledIn.Eval()
	if out.Err() != nil {
		return nil, errors.WithStack(out.Err())
	}
	it, err := out.LookupPath(cue.ParsePath(cuePathDevice)).Fields()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return it, nil
}

// ConvertInputType converts the type of given input according to the type defined as #Input in transform.cue.
func (t *ServiceTransformer) ConvertInputType(input map[string]string) (map[string]any, error) {
	converted := map[string]any{}
	for k, v := range input {
		kind := CueKindOf(t.value, fmt.Sprintf("%s.%s", cueTypeStrInput, k))
		if kind == cue.BottomKind {
			return nil, fmt.Errorf("key=%s is not defined in input types", k)
		}
		convert, err := NewStrConvFunc(kind)
		if err != nil {
			return nil, fmt.Errorf("the type of key=%s must be in string|int|float|bool|null: %w", k, err)
		}
		vv, err := convert(v)
		if err != nil {
			return nil, fmt.Errorf("type mismatch: key=%s, value=%s: %w", k, v, err)
		}
		converted[k] = vv
	}
	return converted, nil
}

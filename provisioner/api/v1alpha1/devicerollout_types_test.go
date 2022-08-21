/*
Copyright 2022 Hiroki Okui.

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

package v1alpha1_test

import (
	"github.com/hrk091/nwctl/provisioner/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeviceConfigMap_Equal(t *testing.T) {
	m := v1alpha1.DeviceConfigMap{
		"d1": v1alpha1.DeviceConfig{
			Checksum:    "Checksum1",
			GitRevision: "rev1",
		},
		"d2": v1alpha1.DeviceConfig{
			Checksum:    "Checksum2",
			GitRevision: "rev2",
		},
	}
	var testcases = []struct {
		name  string
		given v1alpha1.DeviceConfigMap
		want  bool
	}{
		{
			name: "the same one without copy",
			given: v1alpha1.DeviceConfigMap{
				"d1": v1alpha1.DeviceConfig{
					Checksum:    "Checksum1",
					GitRevision: "rev1",
				},
				"d2": v1alpha1.DeviceConfig{
					Checksum:    "Checksum2",
					GitRevision: "rev2",
				},
			},
			want: true,
		},
		{
			name: "the different one without copy",
			given: v1alpha1.DeviceConfigMap{
				"d1": v1alpha1.DeviceConfig{
					Checksum:    "Checksum1",
					GitRevision: "rev2",
				},
				"d2": v1alpha1.DeviceConfig{
					Checksum:    "Checksum2",
					GitRevision: "rev2",
				},
			},
			want: false,
		},
		{
			name:  "the same one with deepcopy",
			given: m.DeepCopy(),
			want:  true,
		},
	}
	for _, tc := range testcases {
		assert.Equal(t, tc.want, m.Equal(tc.given))
	}

}

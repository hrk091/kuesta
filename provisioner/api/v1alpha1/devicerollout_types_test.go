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
	apiv1alpha1 "github.com/hrk091/nwctl/provisioner/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeviceConfigMap_Equal(t *testing.T) {
	m := apiv1alpha1.DeviceConfigMap{
		"d1": apiv1alpha1.DeviceConfig{
			Checksum:    "Checksum1",
			GitRevision: "rev1",
		},
		"d2": apiv1alpha1.DeviceConfig{
			Checksum:    "Checksum2",
			GitRevision: "rev2",
		},
	}
	var testcases = []struct {
		name  string
		given apiv1alpha1.DeviceConfigMap
		want  bool
	}{
		{
			name: "the same one without copy",
			given: apiv1alpha1.DeviceConfigMap{
				"d1": apiv1alpha1.DeviceConfig{
					Checksum:    "Checksum1",
					GitRevision: "rev1",
				},
				"d2": apiv1alpha1.DeviceConfig{
					Checksum:    "Checksum2",
					GitRevision: "rev2",
				},
			},
			want: true,
		},
		{
			name: "the different one without copy",
			given: apiv1alpha1.DeviceConfigMap{
				"d1": apiv1alpha1.DeviceConfig{
					Checksum:    "Checksum1",
					GitRevision: "rev2",
				},
				"d2": apiv1alpha1.DeviceConfig{
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

func TestDeviceRolloutStatus_IsRunning(t *testing.T) {
	s := apiv1alpha1.DeviceRolloutStatus{}
	assert.False(t, s.IsRunning())
	s.Status = apiv1alpha1.RolloutStatusRunning
	assert.True(t, s.IsRunning())
}

func TestDeviceRolloutStatus_DeviceStatus(t *testing.T) {
	tests := []struct {
		name            string
		given           apiv1alpha1.DeviceRolloutStatus
		wantTxCompleted bool
		wantTxFailed    bool
		wantTxRunning   bool
		wantTxIdle      bool
	}{
		{
			"false: not initialized",
			apiv1alpha1.DeviceRolloutStatus{},
			false,
			false,
			false,
			false,
		},
		{
			"all completed",
			apiv1alpha1.DeviceRolloutStatus{
				DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
					"completed": apiv1alpha1.DeviceStatusCompleted,
					"synced":    apiv1alpha1.DeviceStatusSynced,
					"purged":    apiv1alpha1.DeviceStatusPurged,
				},
			},
			true,
			false,
			false,
			true,
		},
		{
			"all running or completed",
			apiv1alpha1.DeviceRolloutStatus{
				DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
					"completed": apiv1alpha1.DeviceStatusCompleted,
					"running":   apiv1alpha1.DeviceStatusRunning,
					"purged":    apiv1alpha1.DeviceStatusPurged,
				},
			},
			false,
			false,
			true,
			false,
		},
		{
			"some failed",
			apiv1alpha1.DeviceRolloutStatus{
				DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
					"failed":    apiv1alpha1.DeviceStatusFailed,
					"running":   apiv1alpha1.DeviceStatusRunning,
					"completed": apiv1alpha1.DeviceStatusCompleted,
					"purged":    apiv1alpha1.DeviceStatusPurged,
				},
			},
			false,
			true,
			true,
			false,
		},
		{
			"some connection error",
			apiv1alpha1.DeviceRolloutStatus{
				DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
					"connError": apiv1alpha1.DeviceStatusConnectionError,
					"running":   apiv1alpha1.DeviceStatusRunning,
					"completed": apiv1alpha1.DeviceStatusCompleted,
					"purged":    apiv1alpha1.DeviceStatusPurged,
				},
			},
			false,
			true,
			true,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantTxCompleted, tt.given.IsTxCompleted())
			assert.Equal(t, tt.wantTxFailed, tt.given.IsTxFailed())
			assert.Equal(t, tt.wantTxRunning, tt.given.IsTxRunning())
			assert.Equal(t, tt.wantTxIdle, tt.given.IsTxIdle())
		})
	}
}

func TestDeviceRolloutStatus_StartTx(t *testing.T) {
	tests := []struct {
		name  string
		given apiv1alpha1.DeviceRolloutStatus
		want  map[string]apiv1alpha1.DeviceStatus
	}{
		{
			"init statusMap without record",
			apiv1alpha1.DeviceRolloutStatus{
				DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{},
				DeviceStatusMap:        nil,
			},
			map[string]apiv1alpha1.DeviceStatus{},
		},
		{
			"init statusMap ",
			apiv1alpha1.DeviceRolloutStatus{
				DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
					"new1": apiv1alpha1.DeviceConfig{},
					"new2": apiv1alpha1.DeviceConfig{},
				},
				DeviceStatusMap: nil,
			},
			map[string]apiv1alpha1.DeviceStatus{
				"new1": apiv1alpha1.DeviceStatusRunning,
				"new2": apiv1alpha1.DeviceStatusRunning,
			},
		},
		{
			"update statusMap along with purging",
			apiv1alpha1.DeviceRolloutStatus{
				DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
					"curr1": apiv1alpha1.DeviceConfig{},
					"curr2": apiv1alpha1.DeviceConfig{},
					"new":   apiv1alpha1.DeviceConfig{},
				},
				DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
					"gone":  apiv1alpha1.DeviceStatusCompleted,
					"curr1": apiv1alpha1.DeviceStatusSynced,
					"curr2": apiv1alpha1.DeviceStatusFailed,
				},
			},
			map[string]apiv1alpha1.DeviceStatus{
				"gone":  apiv1alpha1.DeviceStatusPurged,
				"curr1": apiv1alpha1.DeviceStatusRunning,
				"curr2": apiv1alpha1.DeviceStatusRunning,
				"new":   apiv1alpha1.DeviceStatusRunning,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.given.StartTx()
			assert.Equal(t, tt.want, tt.given.DeviceStatusMap)
		})
	}
}

func TestDeviceRolloutStatus_ResolveNextDeviceConfig(t *testing.T) {
	desired := apiv1alpha1.DeviceConfig{GitRevision: "desired"}
	prev := apiv1alpha1.DeviceConfig{GitRevision: "prev"}

	tests := []struct {
		name    string
		phase   apiv1alpha1.RolloutPhase
		want    apiv1alpha1.DeviceConfig
		wantErr bool
	}{
		{
			"ok: healthy",
			apiv1alpha1.RolloutPhaseHealthy,
			desired,
			false,
		},
		{
			"ok: rollback",
			apiv1alpha1.RolloutPhaseRollback,
			prev,
			false,
		},
		{
			"bad: not set",
			"",
			apiv1alpha1.DeviceConfig{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := apiv1alpha1.DeviceRolloutStatus{
				Phase: tt.phase,
				DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
					"device1": desired,
				},
				PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
					"device1": prev,
				},
			}

			got, err := s.ResolveNextDeviceConfig("device1")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDeviceRolloutStatus_GetDeviceStatus(t *testing.T) {
	t.Run("not initialized", func(t *testing.T) {
		s := apiv1alpha1.DeviceRolloutStatus{}
		assert.Equal(t, apiv1alpha1.DeviceStatusUnknown, s.GetDeviceStatus("not-exist"))
	})

	t.Run("record not set", func(t *testing.T) {
		s := apiv1alpha1.DeviceRolloutStatus{
			DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{},
		}
		s.SetDeviceStatus("test", apiv1alpha1.DeviceStatusRunning)
		assert.Equal(t, apiv1alpha1.DeviceStatusUnknown, s.GetDeviceStatus("not-exist"))
	})

	t.Run("record set", func(t *testing.T) {
		s := apiv1alpha1.DeviceRolloutStatus{
			DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{},
		}
		s.SetDeviceStatus("test", apiv1alpha1.DeviceStatusRunning)
		assert.Equal(t, apiv1alpha1.DeviceStatusRunning, s.GetDeviceStatus("test"))
	})

}

func TestDeviceRollout_UpdateStatus(t *testing.T) {
	desired := apiv1alpha1.DeviceConfig{GitRevision: "desired"}
	curr := apiv1alpha1.DeviceConfig{GitRevision: "curr"}
	prev := apiv1alpha1.DeviceConfig{GitRevision: "prev"}

	t.Run("running", func(t *testing.T) {
		tests := []struct {
			name        string
			given       apiv1alpha1.DeviceRollout
			want        apiv1alpha1.DeviceRolloutStatus
			wantChanged bool
		}{
			{
				"on completed",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseHealthy,
						Status: apiv1alpha1.RolloutStatusRunning,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"completed": curr,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"completed": apiv1alpha1.DeviceStatusCompleted,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseHealthy,
					Status: apiv1alpha1.RolloutStatusCompleted,
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"completed": apiv1alpha1.DeviceStatusCompleted,
					},
				},
				true,
			},
			{
				"on running",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseHealthy,
						Status: apiv1alpha1.RolloutStatusRunning,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"running": curr,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"running": apiv1alpha1.DeviceStatusRunning,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseHealthy,
					Status: apiv1alpha1.RolloutStatusRunning,
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"running": apiv1alpha1.DeviceStatusRunning,
					},
				},
				false,
			},
			{
				"on failed healthy",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseHealthy,
						Status: apiv1alpha1.RolloutStatusRunning,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"failed": curr,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"failed": apiv1alpha1.DeviceStatusFailed,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseRollback,
					Status: apiv1alpha1.RolloutStatusRunning,
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"failed": apiv1alpha1.DeviceStatusRunning,
					},
				},
				true,
			},
			{
				"on failed rollback",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseRollback,
						Status: apiv1alpha1.RolloutStatusRunning,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"failed": curr,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"failed": apiv1alpha1.DeviceStatusFailed,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseRollback,
					Status: apiv1alpha1.RolloutStatusFailed,
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"failed": apiv1alpha1.DeviceStatusFailed,
					},
				},
				true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := tt.given.DeepCopy()
				got := s.UpdateStatus()
				assert.Equal(t, tt.wantChanged, got)
				assert.Equal(t, tt.want.Phase, s.Status.Phase)
				assert.Equal(t, tt.want.Status, s.Status.Status)
				assert.Equal(t, tt.want.DeviceStatusMap, s.Status.DeviceStatusMap)
				// check not changed
				assert.Equal(t, tt.given.Status.DesiredDeviceConfigMap, s.Status.DesiredDeviceConfigMap)
			})
		}
	})

	t.Run("idle", func(t *testing.T) {

		t.Run("spec not updated", func(t *testing.T) {
			oldDr := apiv1alpha1.DeviceRollout{
				Spec: apiv1alpha1.DeviceRolloutSpec{
					DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": curr,
					},
				},
				Status: apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseHealthy,
					Status: apiv1alpha1.RolloutStatusCompleted,
					DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": curr,
					},
					PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": prev,
					},
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"device1": apiv1alpha1.DeviceStatusCompleted,
					},
				},
			}
			newDr := oldDr.DeepCopy()
			got := newDr.UpdateStatus()
			assert.False(t, got)
			assert.Equal(t, oldDr.Status, newDr.Status)
		})

		tests := []struct {
			name        string
			given       apiv1alpha1.DeviceRollout
			want        apiv1alpha1.DeviceRolloutStatus
			wantChanged bool
		}{
			{
				"spec updated: on healthy",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseHealthy,
						Status: apiv1alpha1.RolloutStatusCompleted,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": curr,
						},
						PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": prev,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"device1": apiv1alpha1.DeviceStatusCompleted,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseHealthy,
					Status: apiv1alpha1.RolloutStatusRunning,
					DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": desired,
					},
					PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": curr,
					},
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"device1": apiv1alpha1.DeviceStatusRunning,
					},
				},
				true,
			},
			{
				"spec updated: on rollback",
				apiv1alpha1.DeviceRollout{
					Spec: apiv1alpha1.DeviceRolloutSpec{
						DeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": desired,
						},
					},
					Status: apiv1alpha1.DeviceRolloutStatus{
						Phase:  apiv1alpha1.RolloutPhaseRollback,
						Status: apiv1alpha1.RolloutStatusFailed,
						DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": curr,
						},
						PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
							"device1": prev,
						},
						DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
							"device1": apiv1alpha1.DeviceStatusFailed,
						},
					},
				},
				apiv1alpha1.DeviceRolloutStatus{
					Phase:  apiv1alpha1.RolloutPhaseHealthy,
					Status: apiv1alpha1.RolloutStatusRunning,
					DesiredDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": desired,
					},
					PrevDeviceConfigMap: apiv1alpha1.DeviceConfigMap{
						"device1": prev,
					},
					DeviceStatusMap: map[string]apiv1alpha1.DeviceStatus{
						"device1": apiv1alpha1.DeviceStatusRunning,
					},
				},
				true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := tt.given.UpdateStatus()
				assert.Equal(t, tt.wantChanged, got)
				assert.Equal(t, tt.want, tt.given.Status)
			})
		}
	})

}

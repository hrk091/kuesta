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

package v1alpha1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

const (
	AnnKeyResetStatus = "nwctl.hrk091.dev/reset"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName="dr"
//+kubebuilder:printcolumn:name="PHASE",type="string",JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=`.status.status`

// DeviceRollout is the Schema for the devicerollouts API
type DeviceRollout struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceRolloutSpec   `json:"spec,omitempty"`
	Status DeviceRolloutStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeviceRolloutList contains a list of DeviceRollout
type DeviceRolloutList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeviceRollout `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeviceRollout{}, &DeviceRolloutList{})
}

// DeviceRolloutSpec defines the desired state of DeviceRollout
type DeviceRolloutSpec struct {

	// DeviceConfigMap is a map to bind device name and DeviceConfig to be provisioned
	DeviceConfigMap DeviceConfigMap `json:"deviceConfigMap"`
}

// DeviceConfig provides a digest and other required info of the device config to be provisioned
type DeviceConfig struct {
	// Digest is a digest to uniquely identify the overall device config
	Checksum string `json:"checksum"`

	// GitRevision is a revision from which this device config is provided
	GitRevision string `json:"gitRevision"`
}

type DeviceConfigMap map[string]DeviceConfig

func (m DeviceConfigMap) Equal(o DeviceConfigMap) bool {
	return reflect.DeepEqual(m, o)
}

// DeviceRolloutStatus defines the observed state of DeviceRollout
type DeviceRolloutStatus struct {

	// Phase is the rollout phase
	// +optional
	Phase RolloutPhase `json:"phase,omitempty"`

	// Status is the rollout status
	// +optional
	Status RolloutStatus `json:"status,omitempty"`

	// DeviceStatusMap is the rollout status
	// +optional
	DeviceStatusMap map[string]DeviceStatus `json:"deviceStatusMap,omitempty"`

	// PrevDeviceConfigMap represents the successfully provisioned device configs in the previous transaction
	// +optional
	PrevDeviceConfigMap DeviceConfigMap `json:"prevDeviceConfigMap,omitempty"`

	// DesiredDeviceConfigMap represents the desired device configs to be provisioned in the current transaction
	// +optional
	DesiredDeviceConfigMap DeviceConfigMap `json:"desiredDeviceConfigMap,omitempty"`
}

// RolloutPhase are a set of rollout phases
type RolloutPhase string

const (
	// RolloutPhaseHealthy indicates a rollout is healthy
	RolloutPhaseHealthy RolloutPhase = "Healthy"
	// RolloutPhaseRollback indicates a rollout is degraded and under rollback
	RolloutPhaseRollback RolloutPhase = "Rollback"
)

// RolloutStatus are a set of rollout progress
type RolloutStatus string

const (
	// RolloutStatusCompleted indicates a transaction is completed
	RolloutStatusCompleted RolloutStatus = "Completed"

	// RolloutStatusRunning indicates that a transaction is in progress
	RolloutStatusRunning RolloutStatus = "Running"

	// RolloutStatusFailed indicates that a transaction is failed and stopped.
	// Manual recover is needed to start next transaction
	RolloutStatusFailed RolloutStatus = "Failed"
)

// DeviceStatus are a set of rollout progress
type DeviceStatus string

const (
	// DeviceStatusRunning indicates that a transaction is in progress.
	DeviceStatusRunning DeviceStatus = "Running"

	// DeviceStatusSynced indicates provision is completed and config/actual-config are synced.
	DeviceStatusSynced DeviceStatus = "Synced"

	// DeviceStatusCompleted indicates provision is completed.
	DeviceStatusCompleted DeviceStatus = "Completed"

	// DeviceStatusFailed indicates that provision is failed.
	DeviceStatusFailed DeviceStatus = "Failed"

	// DeviceStatusConnectionError indicates that provision is failed due to connection error.
	DeviceStatusConnectionError DeviceStatus = "ConnectionError"

	// DeviceStatusPurged indicates that the device is not included in desired state.
	DeviceStatusPurged DeviceStatus = "Purged"

	// DeviceStatusUnknown indicates that the device is missing in device status.
	DeviceStatusUnknown DeviceStatus = "Unknown"
)

// IsRunning returns true when rollout status is in `Running`.
func (s *DeviceRolloutStatus) IsRunning() bool {
	return s.Status == RolloutStatusRunning
}

// IsTxCompleted returns true when all device statuses are `Completed` or `Synced`.
func (s *DeviceRolloutStatus) IsTxCompleted() bool {
	if s.DeviceStatusMap == nil {
		return false
	}
	for _, v := range s.DeviceStatusMap {
		if v == DeviceStatusPurged {
			continue
		}
		// TODO remove device completed from transaction completion condition
		if v != DeviceStatusSynced && v != DeviceStatusCompleted {
			return false
		}
	}
	return true
}

// IsTxFailed returns true when one or more device statuses are `Failed` or `ConnectionError`.
func (s *DeviceRolloutStatus) IsTxFailed() bool {
	if s.DeviceStatusMap == nil {
		return false
	}
	for _, v := range s.DeviceStatusMap {
		if v == DeviceStatusPurged {
			continue
		}
		if v == DeviceStatusFailed || v == DeviceStatusConnectionError {
			return true
		}
	}
	return false
}

// IsTxRunning returns true when one or more device statuses are `Running`.
func (s *DeviceRolloutStatus) IsTxRunning() bool {
	if s.DeviceStatusMap == nil {
		return false
	}
	for _, v := range s.DeviceStatusMap {
		if v == DeviceStatusPurged {
			continue
		}
		if v == DeviceStatusRunning {
			return true
		}
	}
	return false
}

// IsTxIdle returns true all device statuses are not `Running`.
func (s *DeviceRolloutStatus) IsTxIdle() bool {
	if s.DeviceStatusMap == nil {
		return false
	}
	for _, v := range s.DeviceStatusMap {
		if v == DeviceStatusPurged {
			continue
		}
		if v == DeviceStatusRunning {
			return false
		}
	}
	return true
}

// StartTx initializes device transaction statuses.
func (s *DeviceRolloutStatus) StartTx() {
	if s.DeviceStatusMap == nil {
		s.DeviceStatusMap = map[string]DeviceStatus{}
	}
	for k, _ := range s.DesiredDeviceConfigMap {
		s.DeviceStatusMap[k] = DeviceStatusRunning
	}
	// purge devices not included in desired device config
	for k, _ := range s.DeviceStatusMap {
		if _, ok := s.DesiredDeviceConfigMap[k]; !ok {
			s.DeviceStatusMap[k] = DeviceStatusPurged
		}
	}
}

// ResolveNextDeviceConfig returns the next device config to transition to according to the current RolloutPhase.
func (s *DeviceRolloutStatus) ResolveNextDeviceConfig(name string) (DeviceConfig, error) {
	if s.Phase == "" {
		return DeviceConfig{}, fmt.Errorf("phase is not set")
	}
	if s.Phase == RolloutPhaseHealthy {
		return s.DesiredDeviceConfigMap[name], nil
	} else {
		return s.PrevDeviceConfigMap[name], nil
	}
}

// GetDeviceStatus returns the device config of the given name.
func (s *DeviceRolloutStatus) GetDeviceStatus(name string) DeviceStatus {
	if s.DeviceStatusMap == nil {
		return DeviceStatusUnknown
	}
	if s, ok := s.DeviceStatusMap[name]; ok {
		return s
	} else {
		return DeviceStatusUnknown
	}
}

// SetDeviceStatus records the given device config to the device status map.
func (s *DeviceRolloutStatus) SetDeviceStatus(name string, status DeviceStatus) {
	s.DeviceStatusMap[name] = status
}

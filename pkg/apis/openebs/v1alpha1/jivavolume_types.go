package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JivaVolumeSpec defines the desired state of JivaVolume
// +k8s:openapi-gen=true
type JivaVolumeSpec struct {
	// Capacity represents the actual resources of the underlying
	// jiva volume.
	PVC      string `json:"pvc,omitempty"`
	Capacity int64  `json:"capacity"`
	// ReplicationFactor represents the actual replica count for the underlying
	// jiva volume
	ReplicationFactor string `json:"replicationFactor"`
	TargetIP          string `json:"targetIP"`
	TargetPort        int32  `json:"targetPort"`
	Iqn               string `json:"iqn"`
	TargetPortal      string `json:"targetPortal"`
}

// JivaVolumeStatus defines the observed state of JivaVolume
// +k8s:openapi-gen=true
type JivaVolumeStatus struct {
	Name            string          `json:"name"`
	Status          string          `json:"status"`
	ReplicaCount    int             `json:"replicaCount"`
	ReplicaStatuses []ReplicaStatus `json:"replicaStatus"`
	// Phase represents the current phase of JivaVolume.
	Phase JivaVolumePhase `json:"phase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JivaVolume is the Schema for the jivavolumes API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type JivaVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JivaVolumeSpec   `json:"spec,omitempty"`
	Status JivaVolumeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JivaVolumeList contains a list of JivaVolume
type JivaVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JivaVolume `json:"items"`
}

// ReplicaStatus stores the status of replicas
type ReplicaStatus struct {
	Address string `json:"address"`
	Mode    string `json:"mode"`
}

// JivaVolumePhase represents the current phase of JivaVolume.
type JivaVolumePhase string

const (
	// JivaVolumePhasePending indicates that the jvc is still waiting for
	// the jivavolume to be created and bound
	JivaVolumePhasePending JivaVolumePhase = "Pending"

	// JivaVolumePhaseSyncing indicates that the jivavolume has been
	// provisioned and replicas are syncing
	JivaVolumePhaseSyncing JivaVolumePhase = "Syncing"

	// JivaVolumePhaseFailed indicates that the jivavolume provisioning
	// has failed
	JivaVolumePhaseFailed JivaVolumePhase = "Failed"

	// JivaVolumePhaseCreated indicates that the jivavolume provisioning
	// has Created
	JivaVolumePhaseCreated JivaVolumePhase = "Created"
)

func init() {
	SchemeBuilder.Register(&JivaVolume{}, &JivaVolumeList{})
}
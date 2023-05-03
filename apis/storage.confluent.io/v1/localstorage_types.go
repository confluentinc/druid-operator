/*

 */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LocalStorageSpec defines the desired state of LocalStorage
type LocalStorageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of LocalStorage. Edit localstorage_types.go to remove/update
	Replicas           int    `json:"replicas"`
	InstanceType       string `json:"instanceType"`
	ForceDeploy        bool   `json:"forceDeploy"`
	EKSImage           string `json:"eksImage"`
	NodeGrabberImage   string `json:"nodeGrabberImage"`
	LocalVolumeImage   string `json:"localVolumeImage"`
	StorageClassName   string `json:"storageClassName"`
	ServiceAccountName string `json:"serviceAccountName"`
}

// LocalStorageStatus defines the observed state of LocalStorage
type LocalStorageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LastUpdated            string   `json:"lastUpdated,omitempty"`
	NodeGrabber            string   `json:"nodeGrabber,omitempty"`
	LocalVolumeProvisioner string   `json:"localVolumeProvisioner,omitempty"`
	EKSNVMEProvisioner     string   `json:"eksNvmeProvisioner,omitempty"`
	Pods                   []string `json:"pods,omitempty"`
	ConfigMaps             []string `json:"configMaps,omitempty"`
	PersistentVolumes      []string `json:"persistentVolumes,omitempty"`
	StorageClasses         []string `json:"storageClass,omitempty"`
	Deployments            []string `json:"deployments,omitempty"`
	DaemonSets             []string `json:"daemonSets,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LocalStorage is the Schema for the localstorages API
type LocalStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LocalStorageSpec   `json:"spec,omitempty"`
	Status LocalStorageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LocalStorageList contains a list of LocalStorage
type LocalStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LocalStorage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LocalStorage{}, &LocalStorageList{})
}

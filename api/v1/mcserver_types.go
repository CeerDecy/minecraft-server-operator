/*
Copyright 2024.

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// McServerSpec defines the desired state of McServer
type McServerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Image string `json:"image,omitempty"`

	Version string `json:"version,omitempty"`

	StorageSize resource.Quantity `json:"storageSize,omitempty"`

	StorageClassName string `json:"storageClassName,omitempty"`

	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`

	ServerProperties corev1.ConfigMap `json:"serverProperties,omitempty"`
}

// McServerStatus defines the observed state of McServer
type McServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Version string `json:"version,omitempty"`
	State   string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// McServer is the Schema for the mcservers API
type McServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   McServerSpec   `json:"spec,omitempty"`
	Status McServerStatus `json:"status,omitempty"`
}

func (m *McServer) NewLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/component": "mcserver",
		"app.kubernetes.io/name":      m.Name,
		"app.kubernetes.io/version":   m.Status.Version,
	}
}

// +kubebuilder:object:root=true

// McServerList contains a list of McServer
type McServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []McServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&McServer{}, &McServerList{})
}

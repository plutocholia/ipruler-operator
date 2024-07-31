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
	"github.com/plutocholia/ipruler-controller/internal/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FullConfigSpec defines the desired state of FullConfig
type FullConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	NodeSelector  map[string]string  `json:"nodeSelector,omitempty"`
	ClusterConfig models.ConfigModel `json:"clusterConfig,omitempty"`
	NodeConfig    models.ConfigModel `json:"nodeConfig,omitempty"`
	MergedConfig  models.ConfigModel `json:"mergedConfig,omitempty"`
}

// FullConfigStatus defines the observed state of FullConfig
type FullConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	HasNodeConfig    bool `json:"hasNodeConfig"`
	HasClusterConfig bool `json:"hasClusterConfig"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// FullConfig is the Schema for the fullconfigs API
type FullConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FullConfigSpec   `json:"spec,omitempty"`
	Status FullConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FullConfigList contains a list of FullConfig
type FullConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FullConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FullConfig{}, &FullConfigList{})
}

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CompatibilityConfig is the Schema for the Compatibility Matrix Configuration
type CompatibilityConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CompatibilitySpec `json:"spec,omitempty"`
}

// CompatibilitySpec is the Schema to get MatrixURL
type CompatibilitySpec struct {
	MatrixURL string `json:"matrixURL,omitempty"`
}

//+kubebuilder:object:root=true

// CompatibilityConfigList contains a list of CompatibilityConfig
type CompatibilityConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CompatibilityConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CompatibilityConfig{}, &CompatibilityConfigList{})
}

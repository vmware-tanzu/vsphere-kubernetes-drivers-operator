/*
Copyright 2021.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigStatus string

const (
	VsphereConfigFailed   ConfigStatus = "failed"
	VsphereConfigVerified ConfigStatus = "verified"
)

// VsphereCloudConfigSpec defines the desired state of VsphereCloudConfig
type VsphereCloudConfigSpec struct {
	// VCIP refers to IP of the vcenter which is used to configure for VDO
	VcIP string `json:"vcIp"`
	// Insecure flag determines if connection to VC can be insecured
	Insecure bool `json:"insecure"`
	// Credentials refers to the name of k8s secret storing the VC creds
	Credentials string `json:"credentials"`
	// thumbprint refers to the SSL Thumbprint to be used to establish a secure connection to VC
	Thumbprint string `json:"thumbprint,omitempty"`
	// datacenters refers to list of datacenters on the VC which the configured user account can access
	DataCenters []string `json:"datacenters"`
}

// VsphereCloudConfigStatus defines the observed state of VsphereCloudConfig
type VsphereCloudConfigStatus struct {
	//Config represents the verification status of VDO configuration
	// +kubebuilder:validation:Enum=verified;failed;
	Config ConfigStatus `json:"config"`
	//Message displays text indicating the reason for failure in validating VDO config
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VsphereCloudConfig is the Schema for the vspherecloudconfigs API
type VsphereCloudConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VsphereCloudConfigSpec   `json:"spec,omitempty"`
	Status VsphereCloudConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VsphereCloudConfigList contains a list of VsphereCloudConfig
type VsphereCloudConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VsphereCloudConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VsphereCloudConfig{}, &VsphereCloudConfigList{})
}

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

type CloudProviderConfig struct {
	// VsphereCloudConfigs refers to the collection of the vSphereCloudConfig resource that holds the vSphere configuration
	VsphereCloudConfigs []string `json:"vsphereCloudConfigs,omitempty"`
	// Topology represents the information required for configuring CPI with zone and region
	Topology TopologyInfo `json:"topology,omitempty"`
}

type TopologyInfo struct {
	Zone   string `json:"zone"`
	Region string `json:"region"`
}

// VDOConfigSpec defines the desired state of VDOConfig
type VDOConfigSpec struct {
	// CloudProvider refers to the section of config that is required to configure CPI driver
	CloudProvider CloudProviderConfig `json:"cloudProvider,omitempty"`
	// StorageProvider refers to the section of config that is required to configure CSI driver
	StorageProvider StorageProviderConfig `json:"storageProvider"`
}

type StorageProviderConfig struct {
	// VsphereCloudConfig refers to the name of the vSphereCloudConfig resource that holds the vSphere configuration
	VsphereCloudConfig string `json:"vsphereCloudConfig"`
	// ClusterDistribution refers to the type of k8s distribution such as TKGI, OpenShift
	ClusterDistribution string `json:"clusterDistribution,omitempty"`
	// FileVolumes refers to the configuration required for file volumes
	FileVolumes FileVolume `json:"fileVolumes,omitempty"`
	// CustomKubeletPath refers to the kubelet Path required
	CustomKubeletPath string `json:"customKubeletPath,omitempty"`
}

type FileVolume struct {
	// VSanDataStoreUrl refers to the list of datastores that the CSI drivers can access
	VSanDataStoreUrl []string `json:"vsanDataStoreUrl,omitempty"`
	// NetPermissions refers to the list of Net permissions required for CSI driver to access file based volumes
	NetPermissions []NetPermission `json:"netPermissions,omitempty"`
}

type NetPermission struct {
	// Ip refers to IP Subnet or Range to which these restrictions apply
	Ip string `json:"ips"`
	// Permission refers to access to the volume such as READ_WRITE, READ_ONLY
	Permission string `json:"permissions,omitempty"`
	// RootSquash refers to the access for root user to the volumes.
	// If false, root access is confirmed for all volumes in this IP range
	RootSquash bool `json:"rootSquash,omitempty"`
}

// NodeStatus is used to type the constants describing possible node states w.r.t CPI configuration.
type NodeStatus string

const (
	// NodeStatusPending means that the CPI is yet to configure the node
	NodeStatusPending = NodeStatus("pending")

	// NodeStatusFailed means that CPI failed to configure the node
	NodeStatusFailed = NodeStatus("failed")

	// NodeStatusReady means that the node is configured successfully by CPI.
	NodeStatusReady = NodeStatus("ready")
)

type VDOConfigPhase string

const (
	// Deploying means that VDOConfig is in deploying stage
	Deploying VDOConfigPhase = "Deploying"
	// Deployed means that VDOConfig has been deployed successfully
	Deployed VDOConfigPhase = "Deployed"
	// Configuring means that VDOConfig in in configuring state
	Configuring VDOConfigPhase = "Configuring"
	// Configured means that VDOConfig has configured successfully
	Configured VDOConfigPhase = "Configured"
	// Failed means VDOConfig failed to configure
	Failed VDOConfigPhase = "Failed"
)

type CPIStatus struct {
	// +kubebuilder:validation:Enum=Deploying;Deployed;Configuring;Configured;Failed
	// Phase is used to indicate the Phase of the CPI driver
	Phase VDOConfigPhase `json:"phase,omitempty"`
	// StatusMsg is used to display messages in reference to the Phase of the CPI driver
	StatusMsg string `json:"statusMsg,omitempty"`
	// NodeStatus indicates the status of CPI driver with respect to each node in the cluster.
	NodeStatus map[string]NodeStatus `json:"nodeStatus ,omitempty"`
}

type CSIStatus struct {
	// +kubebuilder:validation:Enum=Deploying;Deployed;Configuring;Configured;Failed
	// Phase is used to indicate the Phase of the CSI driver
	Phase VDOConfigPhase `json:"phase,omitempty"`
	// StatusMsg is used to display messages in reference to the Phase of the CSI driver
	StatusMsg string `json:"statusMsg,omitempty"`
}

// VDOConfigStatus defines the observed state of VDOConfig
type VDOConfigStatus struct {
	// CPIStatus refers to the configuration status of the CPI driver
	CPIStatus CPIStatus `json:"cpi,omitempty"`
	// CSIStatus refers to the configuration status of the CSI driver
	CSIStatus CSIStatus `json:"csi,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VDOConfig is the Schema for the vdoconfigs API
type VDOConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VDOConfigSpec   `json:"spec,omitempty"`
	Status VDOConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VDOConfigList contains a list of VDOConfig
type VDOConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VDOConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VDOConfig{}, &VDOConfigList{})
}

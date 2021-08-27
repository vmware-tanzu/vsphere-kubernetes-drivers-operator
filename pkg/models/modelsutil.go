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

package models

// VersionRange defines the min and max version
type VersionRange struct {
	// Min defines the minumum required version
	Min string `json:"min"`
	// Max defines the maximum required version
	Max string `json:"max"`
}

// CSIVersionInfo defines the CPI Config Specs for various versions
type CSIVersionInfo struct {
	// VsphereVersion defines the min and max version for vSphere
	VSphereVersion VersionRange `json:"vSphere"`
	// k8sVersion defines the skewVersion for k8s
	K8sVersion VersionRange `json:"k8s"`
	// IsCPIRequired is a flag to check if CPI needs to be configured
	IsCPIRequired bool `json:"isCPIRequired"`
	// DeploymentPaths defines list of deployment URLs
	DeploymentPaths []string `json:"deploymentPath"`
}

// SkewVersion defines the skew version for k8s
type SkewVersion struct {
	SkewVersion string `json:"skewVersion"`
}

// CPIVersionInfo defines the CPI Config Specs for various versions
type CPIVersionInfo struct {
	// VsphereVersion defines the min and max version for vSphere
	VSphereVersion VersionRange `json:"vSphere"`
	// k8sVersion defines the skewVersion for k8s
	K8sVersion SkewVersion `json:"k8s"`
	// DeploymentPaths defines list of deployment URLs
	DeploymentPaths []string `json:"deploymentPath"`
}

// Matrix defines the Spec List for CPI and CSI
type CompatMatrix struct {
	// CSISpecList defines list of CSI Version Specs
	CSISpecList map[string]CSIVersionInfo `json:"CSI"`
	// CPISpecList defines the list of CPI Version Specs
	CPISpecList map[string]CPIVersionInfo `json:"CPI"`
}

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

package csi

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

var _ = Describe("TestSecretCreation", func() {
	Context("Secret creation without file volumes should be successful", func() {
		RegisterFailHandler(Fail)

		vc_user := "test_user"
		vc_pwd := "test_user_pwd"
		cloudConfig := createVsphereConfig()

		expectedConfigData := "[Global]\ncluster-id = \"1.1.1.1\"\n\n[VirtualCenter \"1.1.1.1\"]\ninsecure-flag = \"true\"\nuser          = \"test_user\"\npassword      = \"test_user_pwd\"\ndatacenters   = \"datacenter-1\"\n\n"

		It("should be equal to the required secret structure", func() {
			testConfigData, err := CreateCSISecretConfig(&v1alpha1.VDOConfig{}, &cloudConfig, vc_user, vc_pwd, "test_config.conf")

			Expect(err).To(BeNil())
			Expect(reflect.DeepEqual(testConfigData, expectedConfigData)).To(BeTrue())
		})
	})

	Context("Secret creation with file volumes should be successful", func() {
		RegisterFailHandler(Fail)

		vc_user := "test_user"
		vc_pwd := "test_user_pwd"
		cloudConfig := createVsphereConfig()

		vdoConfig := &v1alpha1.VDOConfig{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-resource-A",
				Namespace: "default",
			},
			Spec: v1alpha1.VDOConfigSpec{
				CloudProvider: v1alpha1.CloudProviderConfig{
					VsphereCloudConfigs: []string{"test-resource"},
					Topology: v1alpha1.TopologyInfo{
						Zone:   "k8s-zone-A",
						Region: "k8s-region-A",
					},
				},
				StorageProvider: v1alpha1.StorageProviderConfig{
					VsphereCloudConfig:  "test-resource",
					ClusterDistribution: "",
					FileVolumes: v1alpha1.FileVolume{
						VSanDataStoreUrl: []string{"ds:///vmfs/volumes/vsan:123/"},
						NetPermissions: []v1alpha1.NetPermission{
							v1alpha1.NetPermission{
								Ip:         "10.10.10.0/24",
								Permission: "READ_WRITE",
								RootSquash: true,
							},
						},
					},
				},
			},
			Status: v1alpha1.VDOConfigStatus{
				CPIStatus: v1alpha1.CPIStatus{},
				CSIStatus: v1alpha1.CSIStatus{},
			},
		}

		expectedConfigData := "[Global]\ncluster-id = \"1.1.1.1\"\n\n[VirtualCenter \"1.1.1.1\"]\ninsecure-flag                    = \"true\"\nuser                             = \"test_user\"\npassword                         = \"test_user_pwd\"\ndatacenters                      = \"datacenter-1\"\ntargetvSANFileShareDatastoreURLs = \"ds:///vmfs/volumes/vsan:123/\"\n\n[NetPermissions \"A\"]\nips         = \"10.10.10.0/24\"\npermissions = \"READ_WRITE\"\nrootsquash  = \"true\"\n\n"

		It("should be equal to the required secret structure", func() {
			testConfigData, err := CreateCSISecretConfig(vdoConfig, &cloudConfig, vc_user, vc_pwd, "test_config.conf")
			Expect(err).To(BeNil())
			Expect(reflect.DeepEqual(testConfigData, expectedConfigData)).To(BeTrue())
		})
	})
})

func createVsphereConfig() v1alpha1.VsphereCloudConfig {
	cloudConfig := v1alpha1.VsphereCloudConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		},
		Spec: v1alpha1.VsphereCloudConfigSpec{
			VcIP:        "1.1.1.1",
			Insecure:    true,
			Credentials: "secret-ref",
			DataCenters: []string{"datacenter-1"},
		},
		Status: v1alpha1.VsphereCloudConfigStatus{},
	}

	return cloudConfig
}

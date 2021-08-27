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

package cpi

import (
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("TestSecretCreation", func() {
	Context("Secret creation should be successful", func() {

		cloudconfiglist := createVsphereConfigList()

		testStringdata := map[string][]byte{
			"1.1.1.1.password": []byte("test_user_pwd"),
			"1.1.1.1.username": []byte("test_user"),
			"2.2.2.2.password": []byte("test_user_pwd"),
			"2.2.2.2.username": []byte("test_user"),
		}

		secretTestKey := types.NamespacedName{
			Name:      "testsecret",
			Namespace: "default",
		}

		expectedCpiSecret := v1.Secret{Data: testStringdata, ObjectMeta: metav1.ObjectMeta{
			Name:      secretTestKey.Name,
			Namespace: secretTestKey.Namespace,
		},
		}

		data := make(map[string][]byte)

		It("should be equal to the required data map", func() {

			for _, cloudConfig := range cloudconfiglist {

				vc_user := "test_user"
				vc_pwd := "test_user_pwd"

				AddVCSectionToDataMap(cloudConfig, vc_user, vc_pwd, data)
			}
			Expect(reflect.DeepEqual(data, testStringdata)).To(BeTrue())
		})

		It("should be equal to the required secret structure", func() {
			testCpiSecret := CreateSecret(secretTestKey, data)
			Expect(reflect.DeepEqual(testCpiSecret, expectedCpiSecret)).To(BeTrue())
		})
	})
})

var _ = Describe("TestConfigMapCreationAndUpdate", func() {
	CPI_VSPHERE_CONF_FILE = "test_config.conf"

	secretTestKey := types.NamespacedName{
		Name:      "testsecret",
		Namespace: "default",
	}

	configmapTestKey := types.NamespacedName{
		Name:      "testconfigmap",
		Namespace: "default",
	}

	cloudconfiglist := createVsphereConfigList()

	testConfigMap := v1.ConfigMap{}

	Context(" Creating ConfigMap when zones and regions is not provided", func() {
		vdoConfig := &v1alpha1.VDOConfig{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-resource-A",
				Namespace: "default",
			},
			Spec: v1alpha1.VDOConfigSpec{
				CloudProvider: v1alpha1.CloudProviderConfig{
					VsphereCloudConfigs: []string{"test-resource"},
				},
				StorageProvider: v1alpha1.StorageProviderConfig{
					VsphereCloudConfig:  "test-resource",
					ClusterDistribution: "",
					FileVolumes:         v1alpha1.FileVolume{},
				},
			},
			Status: v1alpha1.VDOConfigStatus{
				CPIStatus: v1alpha1.CPIStatus{},
				CSIStatus: v1alpha1.CSIStatus{},
			},
		}

		predefinedData := createdatawithoutLabels("1.1.1.1", "2.2.2.2")
		expectedConfigMap := createTestConfigMap(predefinedData, configmapTestKey)

		It("should create configMap without error", func() {
			data, err := CreateVsphereConfig(vdoConfig, cloudconfiglist, secretTestKey)
			Expect(err).To(BeNil())

			testConfigMap, err = CreateConfigMap(data, configmapTestKey)
			Expect(err).To(BeNil())
		})

		It("should be equal to the required configmap structure", func() {
			Expect(reflect.DeepEqual(testConfigMap, expectedConfigMap)).To(BeTrue())

		})
	})

	Context(" Creating ConfigMap when zones and regions are provided", func() {

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
					FileVolumes:         v1alpha1.FileVolume{},
				},
			},
			Status: v1alpha1.VDOConfigStatus{
				CPIStatus: v1alpha1.CPIStatus{},
				CSIStatus: v1alpha1.CSIStatus{},
			},
		}
		predefinedData := createdatawithLabels("1.1.1.1", "2.2.2.2", vdoConfig.Spec.CloudProvider.Topology.Region, vdoConfig.Spec.CloudProvider.Topology.Zone)
		expectedConfigMap := createTestConfigMap(predefinedData, configmapTestKey)

		It("should create configMap without error", func() {
			data, err := CreateVsphereConfig(vdoConfig, cloudconfiglist, secretTestKey)
			Expect(err).To(BeNil())

			testConfigMap, err = CreateConfigMap(data, configmapTestKey)
			Expect(err).To(BeNil())
		})

		It("should be equal to the required configmap structure", func() {
			Expect(reflect.DeepEqual(testConfigMap, expectedConfigMap)).To(BeTrue())

		})
	})

	Context(" Creating ConfigMap when only region is provided ", func() {

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
						Region: "k8s-region-A",
					},
				},
				StorageProvider: v1alpha1.StorageProviderConfig{
					VsphereCloudConfig:  "test-resource",
					ClusterDistribution: "",
					FileVolumes:         v1alpha1.FileVolume{},
				},
			},
			Status: v1alpha1.VDOConfigStatus{
				CPIStatus: v1alpha1.CPIStatus{},
				CSIStatus: v1alpha1.CSIStatus{},
			},
		}
		predefinedData := createdatawithLabels("1.1.1.1", "2.2.2.2", vdoConfig.Spec.CloudProvider.Topology.Region, vdoConfig.Spec.CloudProvider.Topology.Zone)
		expectedConfigMap := createTestConfigMap(predefinedData, configmapTestKey)

		It("should create configMap without error", func() {
			data, err := CreateVsphereConfig(vdoConfig, cloudconfiglist, secretTestKey)
			Expect(err).To(BeNil())

			testConfigMap, err = CreateConfigMap(data, configmapTestKey)
			Expect(err).To(BeNil())
		})

		It("should not match the required configmap", func() {
			Expect(reflect.DeepEqual(testConfigMap, expectedConfigMap)).NotTo(BeTrue())
		})
	})

})

func createVsphereConfigList() []v1alpha1.VsphereCloudConfig {
	cloudConfig1 := v1alpha1.VsphereCloudConfig{
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
	cloudConfig2 := v1alpha1.VsphereCloudConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		},
		Spec: v1alpha1.VsphereCloudConfigSpec{
			VcIP:        "2.2.2.2",
			Insecure:    true,
			Credentials: "secret-ref",
			DataCenters: []string{"datacenter-1"},
		},
		Status: v1alpha1.VsphereCloudConfigStatus{},
	}
	var cloudconfiglist []v1alpha1.VsphereCloudConfig
	cloudconfiglist = append(cloudconfiglist, cloudConfig1, cloudConfig2)
	return cloudconfiglist
}

func createTestConfigMap(data map[string]string, configmapTestKey types.NamespacedName) v1.ConfigMap {

	configMapObject := metav1.ObjectMeta{Name: configmapTestKey.Name, Namespace: configmapTestKey.Namespace}
	testConfigMap := v1.ConfigMap{Data: data, ObjectMeta: configMapObject}

	return testConfigMap
}

func createdatawithoutLabels(vcIp1 string, vcIp2 string) map[string]string {

	data := map[string]string{
		VSPHERECONFIG: fmt.Sprintf("vcenter:\n    %s:\n        server: %s\n        datacenters:\n            - datacenter-1\n        secretName: testsecret\n        secretNamespace: default\n        port: 443\n        insecureFlag: true\n    %s:\n        server: %s\n        datacenters:\n            - datacenter-1\n        secretName: testsecret\n        secretNamespace: default\n        port: 443\n        insecureFlag: true\n", vcIp1, vcIp1, vcIp2, vcIp2),
	}

	return data
}

func createdatawithLabels(vcIp1 string, vcIp2 string, region string, zone string) map[string]string {

	data := map[string]string{
		VSPHERECONFIG: fmt.Sprintf("vcenter:\n    %s:\n        server: %s\n        datacenters:\n            - datacenter-1\n        secretName: testsecret\n        secretNamespace: default\n        port: 443\n        insecureFlag: true\n    %s:\n        server: %s\n        datacenters:\n            - datacenter-1\n        secretName: testsecret\n        secretNamespace: default\n        port: 443\n        insecureFlag: true\nlabels:\n    region: %s\n    zone: %s\n", vcIp1, vcIp1, vcIp2, vcIp2, region, zone),
	}

	return data
}

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
	"os"

	vdov1alpha1 "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Vcenter struct {
	Server          string   `yaml:"server"`
	Datacenters     []string `yaml:"datacenters"`
	SecretName      string   `yaml:"secretName"`
	SecretNamespace string   `yaml:"secretNamespace"`
	Port            uint     `yaml:"port"`
	Insecure        bool     `yaml:"insecureFlag"`
}

type Labels struct {
	Region string `yaml:"region"`
	Zone   string `yaml:"zone"`
}

type Config struct {
	Vcenter map[string]Vcenter `yaml:"vcenter"`
	Labels  Labels             `yaml:"labels,omitempty"`
}

const (
	PORT          = 443
	VSPHERECONFIG = "vsphere.conf"
)

var CPI_VSPHERE_CONF_FILE = "/etc/kubernetes/vsphere.conf"

func AddVCSectionToDataMap(config vdov1alpha1.VsphereCloudConfig, vcUser string, vcUserPwd string, stringData map[string][]byte) {

	vcIP := config.Spec.VcIP

	stringData[fmt.Sprintf("%s.username", vcIP)] = []byte(vcUser)
	stringData[fmt.Sprintf("%s.password", vcIP)] = []byte(vcUserPwd)

}
func CreateSecret(cpiSecretKey types.NamespacedName, data map[string][]byte) v1.Secret {
	object := metav1.ObjectMeta{Name: cpiSecretKey.Name, Namespace: cpiSecretKey.Namespace}
	cpiSecret := v1.Secret{Data: data, ObjectMeta: object}
	return cpiSecret
}

func CreateConfigMap(data map[string]string, configMapKey types.NamespacedName) (v1.ConfigMap, error) {

	configMapObject := metav1.ObjectMeta{Name: configMapKey.Name, Namespace: configMapKey.Namespace}
	vsphereConfigMap := v1.ConfigMap{Data: data, ObjectMeta: configMapObject}

	return vsphereConfigMap, nil
}

func CreateVsphereConfig(vdoConfig *vdov1alpha1.VDOConfig, cloudConfigs []vdov1alpha1.VsphereCloudConfig, cpiSecretKey types.NamespacedName) (map[string]string, error) {
	vcMap := make(map[string]Vcenter)
	for _, config := range cloudConfigs {
		vcMap[config.Spec.VcIP] = Vcenter{
			Server:          config.Spec.VcIP,
			Datacenters:     config.Spec.DataCenters,
			Insecure:        config.Spec.Insecure,
			Port:            PORT,
			SecretName:      cpiSecretKey.Name,
			SecretNamespace: cpiSecretKey.Namespace,
		}
	}

	vsphereConfig := Config{
		Vcenter: vcMap,
		Labels:  Labels{vdoConfig.Spec.CloudProvider.Topology.Region, vdoConfig.Spec.CloudProvider.Topology.Zone},
	}

	out, _ := yaml.Marshal(vsphereConfig)

	file, err := os.OpenFile(CPI_VSPHERE_CONF_FILE, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	if err := os.Truncate(CPI_VSPHERE_CONF_FILE, 0); err != nil {
		return nil, err
	}

	defer file.Close()

	_, err = file.Write(out)
	if err != nil {
		return nil, err
	}

	data := map[string]string{
		VSPHERECONFIG: string(out),
	}
	return data, err
}

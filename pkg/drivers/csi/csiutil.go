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
	"fmt"
	vdov1alpha1 "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	"gopkg.in/ini.v1"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"strconv"
	"strings"
)

const (
	CSI_SECRET_CONFIG_FILENAME = "csi-vsphere.conf"
	VIRTUAL_CENTER             = "VirtualCenter "
	NET_PERMISSIONS            = "NetPermissions "
	GLOBAL                     = "Global"
	CLUSTER_ID                 = "cluster-id"
	INSECURE_FLAG              = "insecure-flag"
	USER                       = "user"
	PASSWORD                   = "password"
	DATACENTERS                = "datacenters"
	VSAN_DATASTORE_URL         = "targetvSANFileShareDatastoreURLs"
	NETPERMISSIONS_IP          = "ips"
	PERMISSIONS                = "permissions"
	ROOTSQUASH                 = "rootsquash"
)

func CreateCSISecret(configData string, csiSecretKey types.NamespacedName) v1.Secret {
	data := map[string][]byte{
		CSI_SECRET_CONFIG_FILENAME: []byte(configData),
	}

	object := metav1.ObjectMeta{Name: csiSecretKey.Name, Namespace: csiSecretKey.Namespace}
	csiSecret := v1.Secret{Data: data, ObjectMeta: object}

	return csiSecret
}

func CreateCSISecretConfig(vdoConfig *vdov1alpha1.VDOConfig, cloudConfig *vdov1alpha1.VsphereCloudConfig, vcUser string, vcUserPwd string, csiSecretFileName string) (string, error) {

	file, err := os.OpenFile(csiSecretFileName, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return "", err
	}

	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	configFile, err := ini.Load(csiSecretFileName)

	if err != nil {
		return "", err
	}
	vcIP := cloudConfig.Spec.VcIP
	insecure := strconv.FormatBool(cloudConfig.Spec.Insecure)
	datacenters := cloudConfig.Spec.DataCenters

	configFile.Section(GLOBAL).Key(CLUSTER_ID).SetValue(fmt.Sprintf("\"%s\"", vcIP))
	configFile.Section(VIRTUAL_CENTER + fmt.Sprintf("\"%s\"", vcIP)).Key(INSECURE_FLAG).SetValue(fmt.Sprintf("\"%s\"", insecure))
	configFile.Section(VIRTUAL_CENTER + fmt.Sprintf("\"%s\"", vcIP)).Key(USER).SetValue(fmt.Sprintf("\"%s\"", vcUser))
	configFile.Section(VIRTUAL_CENTER + fmt.Sprintf("\"%s\"", vcIP)).Key(PASSWORD).SetValue(fmt.Sprintf("\"%s\"", vcUserPwd))
	configFile.Section(VIRTUAL_CENTER + fmt.Sprintf("\"%s\"", vcIP)).Key(DATACENTERS).SetValue(fmt.Sprintf("\"%s\"", strings.Join(datacenters, ", ")))

	if vdoConfig.Spec.StorageProvider.FileVolumes.VSanDataStoreUrl != nil {
		vsanDatastoreUrl := vdoConfig.Spec.StorageProvider.FileVolumes.VSanDataStoreUrl
		configFile.Section(VIRTUAL_CENTER + fmt.Sprintf("\"%s\"", vcIP)).Key(VSAN_DATASTORE_URL).SetValue(fmt.Sprintf("\"%s\"", strings.Join(vsanDatastoreUrl, ", ")))
	}

	if len(vdoConfig.Spec.StorageProvider.FileVolumes.NetPermissions) > 0 {
		netPermissions := vdoConfig.Spec.StorageProvider.FileVolumes.NetPermissions
		sequenceCh := 'A'
		for _, netPermission := range netPermissions {
			ip := netPermission.Ip
			configFile.Section(NET_PERMISSIONS + fmt.Sprintf("\"%c\"", sequenceCh)).Key(NETPERMISSIONS_IP).SetValue(fmt.Sprintf("\"%s\"", ip))
			if netPermission.Permission != "" {
				permission := netPermission.Permission
				configFile.Section(NET_PERMISSIONS + fmt.Sprintf("\"%c\"", sequenceCh)).Key(PERMISSIONS).SetValue(fmt.Sprintf("\"%s\"", permission))
			}
			if netPermission.RootSquash {
				rootSquash := strconv.FormatBool(netPermission.RootSquash)
				configFile.Section(NET_PERMISSIONS + fmt.Sprintf("\"%c\"", sequenceCh)).Key(ROOTSQUASH).SetValue(fmt.Sprintf("\"%s\"", rootSquash))
			}
			sequenceCh++
		}
	}

	err = configFile.SaveTo(csiSecretFileName)

	if err != nil {
		return "", err
	}

	s, err := ioutil.ReadFile(csiSecretFileName)

	if err != nil {
		return "", err
	}
	csiSecretConfig := string(s)

	return csiSecretConfig, nil

}

func CompareCSISecret(csiSecret *v1.Secret, configData string) bool {

	return string(csiSecret.Data[CSI_SECRET_CONFIG_FILENAME]) == configData

}

func UpdateCSISecret(csiSecret *v1.Secret, configData string) {
	data := map[string][]byte{
		CSI_SECRET_CONFIG_FILENAME: []byte(configData),
	}
	csiSecret.Data = data

}

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
	"github.com/pkg/errors"
	vdov1alpha1 "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	"gopkg.in/ini.v1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"reflect"
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
	TLS_CRT                    = "tls.crt"
	TLS_KEY                    = "tls.key"
)

func CreateCSISecret(configData string, csiSecretKey types.NamespacedName) v1.Secret {
	data := map[string][]byte{
		CSI_SECRET_CONFIG_FILENAME: []byte(configData),
	}

	object := metav1.ObjectMeta{Name: csiSecretKey.Name, Namespace: csiSecretKey.Namespace}
	csiSecret := v1.Secret{Data: data, ObjectMeta: object}

	return csiSecret
}
func CreateCSIWebhookSecret(dataMap map[string][]byte, secretKey types.NamespacedName) v1.Secret {

	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretKey.Name,
			Namespace: secretKey.Namespace,
		},
		Data: dataMap,
	}

	return secret
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

	s, err := os.ReadFile(csiSecretFileName)

	if err != nil {
		return "", err
	}
	csiSecretConfig := string(s)

	return csiSecretConfig, nil

}

func CreateCSIWebhookCertSecretData(tlsCrt string, tlsKey string, webhookconf string) (map[string][]byte, error) {

	webhookCert, err := os.ReadFile(tlsCrt)
	if err != nil {
		return nil, err
	}

	webhookKey, err := os.ReadFile(tlsKey)
	if err != nil {
		return nil, err
	}

	webhookConfig, err := os.ReadFile(webhookconf)
	if err != nil {
		return nil, err
	}

	data := map[string][]byte{
		TLS_CRT:     webhookCert,
		TLS_KEY:     webhookKey,
		webhookconf: webhookConfig,
	}

	return data, nil

}

func CompareCSISecret(csiSecret *v1.Secret, configData string) bool {

	return string(csiSecret.Data[CSI_SECRET_CONFIG_FILENAME]) == configData

}

func CompareWebhookSecret(secret *v1.Secret, dataMap map[string][]byte) bool {
	return reflect.DeepEqual(secret.Data, dataMap)

}

func UpdateWebhookSecret(secret *v1.Secret, dataMap map[string][]byte) {
	secret.Data = dataMap
}

func UpdateCSISecret(csiSecret *v1.Secret, configData string) {
	data := map[string][]byte{
		CSI_SECRET_CONFIG_FILENAME: []byte(configData),
	}
	csiSecret.Data = data

}

func CreateValidatingWebhookConfiguration() (*admissionv1.ValidatingWebhookConfiguration, error) {
	caBundleString := os.Getenv("CA_BUNDLE")
	if caBundleString == "" {
		return nil, errors.New("couldn't fetch cabundle from environment")
	}
	caBundle := []byte(caBundleString)

	path := "/validate"
	scope := admissionv1.NamespacedScope
	sideEffects := admissionv1.SideEffectClassNone
	failurePolicy := admissionv1.Fail
	validatingWebhookConfiguration := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "validation.csi.vsphere.vmware.com",
		},
		Webhooks: []admissionv1.ValidatingWebhook{
			{
				Name: "validation.csi.vsphere.vmware.com",
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{
						Name:      "vsphere-webhook-svc",
						Namespace: "vmware-system-csi",
						Path:      &path,
					},
					CABundle: caBundle,
				},
				Rules: []admissionv1.RuleWithOperations{
					{
						Operations: []admissionv1.OperationType{admissionv1.Create, admissionv1.Update},
						Rule: admissionv1.Rule{
							APIGroups:   []string{"storage.k8s.io"},
							APIVersions: []string{"v1", "v1beta1"},
							Resources:   []string{"storageclasses"},
						},
					},
					{
						Operations: []admissionv1.OperationType{admissionv1.Update, admissionv1.Delete},
						Rule: admissionv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1", "v1beta1"},
							Resources:   []string{"persistentvolumeclaims"},
							Scope:       &scope,
						},
					},
				},
				SideEffects:             &sideEffects,
				AdmissionReviewVersions: []string{"v1"},
				FailurePolicy:           &failurePolicy,
			},
		},
	}
	return validatingWebhookConfiguration, nil
}

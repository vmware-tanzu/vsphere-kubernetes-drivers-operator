/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
package cmd

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/session"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/vdoctl/pkg/utils"

	"github.com/thanhpk/randstr"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type credentials struct {
	errorMsg            string
	username            string
	password            string
	vcIp                string
	insecure            bool
	datacenters         []string
	vSANDataStoresUrl   []string
	vsphereCloudConfig  string
	topology            v1alpha1.TopologyInfo
	netPermissions      []v1alpha1.NetPermission
	vsphereCloudConfigs []string
	thumbprint          string
}

const (
	KubeSystemNamespace = "kube-system"
	vdoConfigName       = "vdo-config"
	secretType          = "kubernetes.io/basic-auth"
	ClusterDistribution = "OpenShift"
)

// driversCmd represents the drivers command
var driversCmd = &cobra.Command{
	Use:   "drivers",
	Short: "Command to configure VDO",
	Long:  `This command helps to specify the details required to configure CloudProvider and StorageProvider drivers.`,

	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		cpi := credentials{}
		csi := credentials{}

		thumbprintMap := make(map[string]string)

		var isCPIMultiVC bool
		var vsphereCloudConfigList, vcIPList []string

		labels := credentials{
			username: "Username",
			password: "Password",
			vcIp:     "VC_IP",
			topology: v1alpha1.TopologyInfo{
				Zone:   "Zones",
				Region: "Regions",
			},
			thumbprint: "SSL Thumbprint",
		}

		isCPIRequired := utils.PromptGetInput("Do you want to configure CloudProvider? (Y/N)", errors.New("invalid input"), utils.IsString)

		if strings.EqualFold(isCPIRequired, "Y") {

		multivcloop:
			for {
				fetchVCIP(&cpi, labels, "CloudProvider")
			thumbprintloop:
				for {
					if !cpi.insecure {
						fetchThumbprint(&cpi, labels)
						thumbprintMap[cpi.vcIp] = cpi.thumbprint
					}
					fmt.Print("Please provide the credentials for configuring CloudProvider\n")
				cpiCredsLoop:
					for {
						fetchCredentials(&cpi, labels)
					dcloop:
						for {
							fetchDatacenters(&cpi)
							_, err := session.GetOrCreate(ctx, cpi.vcIp, cpi.datacenters, cpi.username, cpi.password, cpi.thumbprint)
							if err != nil {
								fmt.Printf("Invalid input for VC: %s. Error: %v\nPlease provide the input again\n", cpi.vcIp, err)
								if checkPattern("unable to find datacenter", err) {
									continue dcloop
								} else if checkPattern("incorrect user name or password", err) {
									continue cpiCredsLoop
								} else if checkPattern("thumbprint does not match", err) {
									continue thumbprintloop
								}
							}
							break
						}
						break
					}
					break
				}
				vcIPList = append(vcIPList, cpi.vcIp)
				secret, err := createSecret(K8sClient, ctx, cpi, "cpi")
				if err != nil {
					cobra.CheckErr(err)
				}

				vcc, err := createVsphereCloudConfig(K8sClient, ctx, cpi, secret.Name, "cpi")
				if err != nil {
					cobra.CheckErr(err)
				}
				vsphereCloudConfigList = append(vsphereCloudConfigList, vcc.Name)

				multiVC := utils.PromptGetInput("Do you want to configure another vcenter for CloudProvider? (Y/N)", errors.New("invalid input"), utils.IsString)

				if strings.EqualFold(multiVC, "Y") {
					isCPIMultiVC = true
					cpi = credentials{}
					continue multivcloop

				}
				cpi.vsphereCloudConfigs = vsphereCloudConfigList
				break
			}

			topology := utils.PromptGetInput("Do you want to configure zones/regions for CloudProvider? (Y/N)", errors.New("invalid input"), utils.IsString)

			if strings.EqualFold(topology, "Y") {
				cpi.topology.Zone = utils.PromptGetInput(labels.topology.Zone, errors.New("unable to get the zones"), utils.IsString)
				cpi.topology.Region = utils.PromptGetInput(labels.topology.Region, errors.New("unable to get the regions"), utils.IsString)
			}

			fmt.Println("You have now completed configuration of CloudProvider. We will now proceed to configure StorageProvider.")

		}

		if isCPIMultiVC {
			csi.vcIp = utils.PromptGetSelect(vcIPList, "Please select vcenter for configuring StorageProvider?")
			if _, ok := thumbprintMap[csi.vcIp]; ok {
				csi.thumbprint = thumbprintMap[csi.vcIp]
				csi.insecure = false
			} else {
				csi.insecure = true
			}

		} else if strings.EqualFold(isCPIRequired, "Y") {
			csi.vcIp = cpi.vcIp
			csi.insecure = cpi.insecure
			csi.thumbprint = cpi.thumbprint
		} else {
			fetchVCIP(&csi, labels, "StorageProvider")
		}

		fmt.Print("Please provide the credentials for configuring StorageProvider\n")
	credentialsLoop:
		for {
			fetchCredentials(&csi, labels)
			_, err := session.GetOrCreate(ctx, csi.vcIp, csi.datacenters, csi.username, csi.password, csi.thumbprint)
			if err != nil {
				fmt.Printf("invalid credentials for VC: %s. Error: %v\nPlease provide the input again\n", csi.vcIp, err)
				continue credentialsLoop
			}
			break
		}

		secret, err := createSecret(K8sClient, ctx, csi, "csi")
		if err != nil {
			cobra.CheckErr(err)
		}

		vcc, err := createVsphereCloudConfig(K8sClient, ctx, csi, secret.Name, "csi")
		if err != nil {
			cobra.CheckErr(err)
		}
		csi.vsphereCloudConfig = vcc.Name

		advConfig := utils.PromptGetInput("Do you wish to configure File Volumes? (Y/N)", errors.New("invalid input"), utils.IsString)

		if strings.EqualFold(advConfig, "Y") {

			vsanDSurl := utils.PromptGetInput("Do you wish to provide for vSAN DataStore Url for File Volumes? (Y/N)", errors.New("invalid input"), utils.IsString)
			if strings.EqualFold(vsanDSurl, "Y") {
				res := utils.PromptGetInput("vSAN DataStore Url(s)", errors.New("unable to get the vSAN DataStore Url"), utils.IsString)
				csi.vSANDataStoresUrl = strings.SplitAfter(res, ",")
			}

			netPerms := utils.PromptGetInput("Do you wish to configure Net File permissions for File Volumes? (Y/N)", errors.New("invalid input"), utils.IsString)
			if strings.EqualFold(netPerms, "Y") {

				for {
					netPermission := v1alpha1.NetPermission{}
					netPermission.Ip = utils.PromptGetInput("IP Address", errors.New("unable to get the IP Address"), utils.IsString)

					netPermission.Permission = utils.PromptGetSelect([]string{"READ_ONLY", "READ_WRITE"}, "Permissions")

					rs := utils.PromptGetInput("Allow Root Access? (Y/N)", errors.New("invalid input"), utils.IsString)

					if strings.EqualFold(rs, "Y") {
						netPermission.RootSquash = true
					}

					csi.netPermissions = append(csi.netPermissions, netPermission)
					netPerms = utils.PromptGetInput("Do you want to configure another Net permissions (Y/N)", errors.New("invalid input"), utils.IsString)

					if strings.EqualFold(netPerms, "Y") {
						continue
					}
					break
				}
			}
		}

		err = createVDOConfig(K8sClient, ctx, cpi, csi)
		if err != nil {
			cobra.CheckErr(err)
		}
		fmt.Println("Thanks For configuring VDO. The drivers should be installed/configured soon")

	},
}

func init() {
	configureCmd.AddCommand(driversCmd)

}

func createSecret(cl client.Client, ctx context.Context, cred credentials, driver string) (v1.Secret, error) {

	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-creds-%s", cred.vcIp, driver),
			Namespace: KubeSystemNamespace,
		},
		Type: secretType,

		Data: map[string][]byte{
			"username": []byte(cred.username),
			"password": []byte(cred.password),
		},
	}

	err := cl.Create(ctx, &secret, &runtimeclient.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return secret, nil
		}
	}
	return secret, err

}

func createVsphereCloudConfig(cl client.Client, ctx context.Context, cred credentials, secretName string, driver string) (v1alpha1.VsphereCloudConfig, error) {
	vcc := &v1alpha1.VsphereCloudConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", cred.vcIp, driver),
			Namespace: VdoNamespace,
		},
		Spec: v1alpha1.VsphereCloudConfigSpec{
			VcIP:        cred.vcIp,
			Insecure:    cred.insecure,
			Credentials: secretName,
			Thumbprint:  cred.thumbprint,
			DataCenters: cred.datacenters,
		},
		Status: v1alpha1.VsphereCloudConfigStatus{},
	}

	err := cl.Create(ctx, vcc, &runtimeclient.CreateOptions{})

	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return *vcc, nil
		}
	}
	return *vcc, err
}

func createVDOConfig(cl client.Client, ctx context.Context, cpi credentials, csi credentials) error {

	vdoConfig := &v1alpha1.VDOConfig{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      vdoConfigName + randstr.Hex(4),
			Namespace: VdoNamespace,
		},
		Spec: v1alpha1.VDOConfigSpec{
			StorageProvider: v1alpha1.StorageProviderConfig{
				VsphereCloudConfig:  csi.vsphereCloudConfig,
				ClusterDistribution: ClusterDistribution,
				FileVolumes: v1alpha1.FileVolume{
					VSanDataStoreUrl: csi.vSANDataStoresUrl,
					NetPermissions:   csi.netPermissions,
				},
			},
		},
	}

	if len(cpi.vsphereCloudConfigs) > 0 {
		vdoConfig.Spec.CloudProvider = v1alpha1.CloudProviderConfig{
			VsphereCloudConfigs: cpi.vsphereCloudConfigs,
			Topology:            cpi.topology,
		}
	}

	err := cl.Create(ctx, vdoConfig, &runtimeclient.CreateOptions{})

	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
	}
	return err
}

func fetchVCIP(cred *credentials, labels credentials, driver string) {
	fmt.Printf("Please provide the vcenter IP for configuring %s \n", driver)

	vcIp := utils.PromptGetInput(labels.vcIp, errors.New("unable to get the VC_IP"), utils.IsIP)
	cred.vcIp = vcIp

	res := utils.PromptGetInput("Do you want to establish a secure connection? (Y/N)", errors.New("invalid input"), utils.IsString)
	if strings.EqualFold(res, "Y") {
		cred.insecure = false

	} else {
		cred.insecure = true
	}
}

func fetchThumbprint(cred *credentials, labels credentials) {
	thumbprint := utils.PromptGetInput(labels.thumbprint, errors.New("invalid input"), utils.IsString)
	cred.thumbprint = thumbprint

}

func fetchCredentials(cred *credentials, labels credentials) {
	cred.username = utils.PromptGetInput(labels.username, errors.New("unable to get the username"), utils.IsString)

	cred.password = utils.PromptGetInput(labels.password, errors.New("unable to get the password"), utils.IsPwd)

}

func fetchDatacenters(cred *credentials) {
	dc := utils.PromptGetInput("Datacenter(s)", errors.New("unable to get the datacenters"), utils.IsString)
	cred.datacenters = strings.SplitAfter(dc, ",")
}

func checkPattern(pattern string, err error) bool {
	MyRegex, _ := regexp.Compile(pattern)
	return MyRegex.MatchString(err.Error())
}

//TODO Add validations for File Volumes and zones/regions input

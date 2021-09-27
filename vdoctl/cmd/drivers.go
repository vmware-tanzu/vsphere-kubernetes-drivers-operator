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
	"flag"
	"fmt"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/vdoctl/pkg/utils"
	"strings"

	"github.com/thanhpk/randstr"
	"k8s.io/client-go/rest"

	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/client-go/kubernetes/scheme"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"path/filepath"

	"os"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

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
	VdoNamespace        = "vmware-system-vdo"
	GroupName           = "vdo.vmware.com"
	GroupVersion        = "v1alpha1"
	KubeSystemNamespace = "kube-system"
	vdoConfigName       = "vdo-config"
	secretType          = "kubernetes.io/basic-auth"
	ClusterDistribution = "OpenShift"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
)

// driversCmd represents the drivers command
var driversCmd = &cobra.Command{
	Use:   "drivers",
	Short: "Command to configure VDO",
	Long:  `This command helps to specify the details required to configure CloudProvider and Storage Provider drivers.`,

	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		cpi := credentials{}
		csi := credentials{}

		thumbprintMap := make(map[string]string)

		var isCPIMultiVC bool
		var vsphereCloudConfigList, vcIPList []string

		config, err := buildConfig()
		if err != nil {
			panic(err)
		}

		//TODO remove code for creating client

		client, err := runtimeclient.New(config, client.Options{
			Scheme: scheme.Scheme,
		})
		if err != nil {
			panic(err)
		}

		err = SchemeBuilder.AddToScheme(scheme.Scheme)
		if err != nil {
			panic(err)
		}

		labels := credentials{
			errorMsg: "unable to get VC creds",
			username: "Username",
			password: "Password",
			vcIp:     "VC_IP",
			topology: v1alpha1.TopologyInfo{
				Zone:   "Zones",
				Region: "Regions",
			},
			thumbprint: "SSL Thumbprint",
		}

		isCPIRequired := utils.PromptGetInput("Do you want to configure Cloud Provider? (Y/N)", errors.New("invalid input"), utils.IsString)

		if isCPIRequired == "Y" || isCPIRequired == "y" {
			fetchVCIP(&cpi, labels, "Cloud Provider")
			vcIPList = append(vcIPList, cpi.vcIp)
			if !cpi.insecure {
				thumbprintMap[cpi.vcIp] = cpi.thumbprint
			}

		multivcloop:
			for {

				fetchCredentials(&cpi, labels, "Cloud Provider")

				secret, err := createSecret(client, ctx, cpi, "cpi")
				if err != nil {
					panic(err)
				}

				vcc, err := createVsphereCloudConfig(client, ctx, cpi, secret.Name, "cpi")
				if err != nil {
					panic(err)
				}
				vsphereCloudConfigList = append(vsphereCloudConfigList, vcc.Name)

				multiVC := utils.PromptGetInput("Do you want to configure another VC for CPI? (Y/N)", errors.New("invalid input"), utils.IsString)

				if multiVC == "Y" || multiVC == "y" {
					isCPIMultiVC = true
					cpi = credentials{}

					fetchVCIP(&cpi, labels, "Cloud Provider")
					vcIPList = append(vcIPList, cpi.vcIp)
					if !cpi.insecure {
						thumbprintMap[cpi.vcIp] = cpi.thumbprint
					}
					continue multivcloop

				}
				cpi.vsphereCloudConfigs = vsphereCloudConfigList
				break

			}

			topology := utils.PromptGetInput("Do you want to configure zones/regions for CPI? (Y/N)", errors.New("invalid input"), utils.IsString)

			if topology == "Y" || topology == "y" {
				cpi.topology.Zone = utils.PromptGetInput(labels.topology.Zone, errors.New("unable to get the zones"), utils.IsString)
				cpi.topology.Region = utils.PromptGetInput(labels.topology.Region, errors.New("unable to get the regions"), utils.IsString)
			}

			fmt.Println("You have now completed configuration of Cloud Provider. We will now proceed to configure Storage Provider. ")

		}

		if isCPIMultiVC {
			csi.vcIp = utils.PromptGetSelect(vcIPList, "Please select vcenter for configuring Storage Provider?")
			if _, ok := thumbprintMap[csi.vcIp]; ok {
				csi.thumbprint = thumbprintMap[csi.vcIp]
				csi.insecure = false
			} else {
				csi.insecure = true
			}

		} else if isCPIRequired == "Y" || isCPIRequired == "y" {
			csi.vcIp = cpi.vcIp
			csi.insecure = cpi.insecure
			csi.thumbprint = cpi.thumbprint
		} else {
			fetchVCIP(&csi, labels, "Storage Provider")
		}

		fetchCredentials(&csi, labels, "Storage Provider")

		secret, err := createSecret(client, ctx, csi, "csi")

		if err != nil {
			panic(err)
		}

		vcc, err := createVsphereCloudConfig(client, ctx, csi, secret.Name, "csi")
		if err != nil {
			panic(err)
		}
		csi.vsphereCloudConfig = vcc.Name

		advConfig := utils.PromptGetInput("Do you wish to configure File Volumes? (Y/N)", errors.New("invalid input"), utils.IsString)

		if advConfig == "Y" || advConfig == "y" {

			vsanDSurl := utils.PromptGetInput("vSANDataStoresUrl", errors.New("unable to get the vSANDataStoresUrl"), utils.IsString)
			csi.vSANDataStoresUrl = strings.SplitAfter(vsanDSurl, ",")

			netPerms := utils.PromptGetInput("Do you wish to configure netPermission? (Y/N)", errors.New("invalid input"), utils.IsString)
			if netPerms == "Y" || netPerms == "y" {

				for {
					netPermission := v1alpha1.NetPermission{}

					fmt.Println("Please provide the IP Subnet/Range for volumes")
					netPermission.Ip = utils.PromptGetInput("IP", errors.New("unable to get the IP"), utils.IsString)

					netPermission.Permission = utils.PromptGetSelect([]string{"READ_ONLY", "READ_WRITE"}, "Please select the permission type to access the volume?")

					rs := utils.PromptGetInput("Do you want to provide access for root user to the volumes? (Y/N)", errors.New("invalid input"), utils.IsString)

					if rs == "Y" || rs == "y" {
						netPermission.RootSquash = true
					}

					csi.netPermissions = append(csi.netPermissions, netPermission)
					netPerms = utils.PromptGetInput("Do you want to configure another Net permissions (Y/N)", errors.New("invalid input"), utils.IsString)

					if netPerms == "Y" || netPerms == "y" {
						continue
					}
					break
				}
			}
		}
		err = createVDOConfig(client, ctx, cpi, csi)
		if err != nil {
			panic(err)
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
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)

	}
	return *vcc, err
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&v1alpha1.VsphereCloudConfig{},
		&v1alpha1.VsphereCloudConfigList{},
		&v1alpha1.VDOConfig{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)

	return nil
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
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)

	}
	return err
}

func buildConfig() (*rest.Config, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, err
}

func fetchVCIP(cred *credentials, labels credentials, driver string) {
	fmt.Printf("Please provide the vcenter IP for configuring %s \n", driver)

	vcIp := utils.PromptGetInput(labels.vcIp, errors.New("unable to get the VC_IP"), utils.IsIP)
	cred.vcIp = vcIp

	res := utils.PromptGetInput("Do you want to establish a secure connection? (Y/N)", errors.New("invalid input"), utils.IsString)
	if res == "Y" || res == "y" {
		fmt.Println("Please provide the SSL Thumbprint")
		thumbprint := utils.PromptGetInput(labels.thumbprint, errors.New("invalid input"), utils.IsString)
		cred.insecure = false
		cred.thumbprint = thumbprint

	} else {
		cred.insecure = true
	}
}

func fetchCredentials(cred *credentials, labels credentials, driver string) {
	fmt.Printf("Please provide the credentials for configuring %s \n", driver)

	cred.username = utils.PromptGetInput(labels.username, errors.New("unable to get the username"), utils.IsString)

	cred.password = utils.PromptGetInput(labels.password, errors.New("unable to get the password"), utils.IsPwd)

	dc := utils.PromptGetInput("Datacenter(s)", errors.New("unable to get the datacenters"), utils.IsString)

	cred.datacenters = strings.SplitAfter(dc, ",")

}

//TODO Add loggers and validations for IP login

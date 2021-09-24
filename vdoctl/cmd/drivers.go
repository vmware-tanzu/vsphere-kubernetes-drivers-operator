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
	"strings"

	"github.com/thanhpk/randstr"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/vdoctl/pkg"
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
	"k8s.io/client-go/kubernetes"
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

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err)
		}

		cl, _ := runtimeclient.New(config, client.Options{
			Scheme: scheme.Scheme,
		})

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

		isCPIRequired := pkg.PromptGetInput("Do you want to configure Cloud Provider ? (Y/N)", errors.New("invalid input"), pkg.IsString)

		if isCPIRequired == "Y" || isCPIRequired == "y" {
			getVCIP(&cpi, labels, "Cloud Provider")
			vcIPList = append(vcIPList, cpi.vcIp)

		multivcloop:
			for {
				fmt.Println("Please provide the credentials")

				cpi.username = pkg.PromptGetInput(labels.username, errors.New("unable to get the username"), pkg.IsString)
				cpi.password = pkg.PromptGetInput(labels.password, errors.New("unable to get the password"), pkg.IsPwd)
				dc := pkg.PromptGetInput("Datacenter(s)", errors.New("unable to get the datacenters"), pkg.IsString)
				cpi.datacenters = strings.SplitAfter(dc, ",")

				secret, err := createSecret(clientset, ctx, cpi, "cpi")
				if err != nil {
					panic(err)
				}

				vcc, err := createVsphereCloudConfig(cl, ctx, cpi, secret.Name, "cpi")
				if err != nil {
					panic(err)
				}
				vsphereCloudConfigList = append(vsphereCloudConfigList, vcc.Name)
				multiVC := pkg.PromptGetInput("Do you want to configure another VC for CPI? (Y/N)", errors.New("invalid input"), pkg.IsString)

				if multiVC == "Y" || multiVC == "y" {
					isCPIMultiVC = true
					cpi = credentials{}

					getVCIP(&cpi, labels, "Cloud Provider")
					vcIPList = append(vcIPList, cpi.vcIp)
					continue multivcloop

				}
				cpi.vsphereCloudConfigs = vsphereCloudConfigList
				break

			}

			topology := pkg.PromptGetInput("Do you want to configure zones/regions for CPI? (Y/N)", errors.New("invalid input"), pkg.IsString)

			if topology == "Y" || topology == "y" {
				cpi.topology.Zone = pkg.PromptGetInput(labels.topology.Zone, errors.New("unable to get the zones"), pkg.IsString)
				cpi.topology.Region = pkg.PromptGetInput(labels.topology.Region, errors.New("unable to get the regions"), pkg.IsString)
			}

			fmt.Println("You have now completed configuration of Cloud Provider. We will now proceed to configure Storage Provider. ")

		}

		if isCPIMultiVC {
			csi.vcIp = pkg.PromptGetSelect(vcIPList, "Please select vcenter for configuring Storage Provider?")
			if _, ok := thumbprintMap[csi.vcIp]; ok {
				csi.thumbprint = thumbprintMap[csi.vcIp]
			}
		} else {
			getVCIP(&csi, labels, "Storage Provider")
		}

		fmt.Println("Please provide the credentials for Storage Provider")

		csi.username = pkg.PromptGetInput(labels.username, errors.New("unable to get the username"), pkg.IsString)

		csi.password = pkg.PromptGetInput(labels.password, errors.New("unable to get the password"), pkg.IsPwd)

		dc := pkg.PromptGetInput("Datacenter(s)", errors.New("unable to get the datacenters"), pkg.IsString)

		csi.datacenters = strings.SplitAfter(dc, ",")

		secret, err := createSecret(clientset, ctx, csi, "csi")

		if err != nil {
			panic(err)
		}

		vcc, err := createVsphereCloudConfig(cl, ctx, csi, secret.Name, "csi")
		if err != nil {
			panic(err)
		}
		csi.vsphereCloudConfig = vcc.Name

		advConfig := pkg.PromptGetInput("Do you wish to configure File Volumes? (Y/N)", errors.New("invalid input"), pkg.IsString)

		if advConfig == "Y" || advConfig == "y" {

			vsanDSurl := pkg.PromptGetInput("vSANDataStoresUrl", errors.New("unable to get the vSANDataStoresUrl"), pkg.IsString)
			csi.vSANDataStoresUrl = strings.SplitAfter(vsanDSurl, ",")

			netPerms := pkg.PromptGetInput("Do you wish to configure netPermission? (Y/N)", errors.New("invalid input"), pkg.IsString)
			if netPerms == "Y" || netPerms == "y" {

				for {
					netPermission := v1alpha1.NetPermission{}

					fmt.Println("Please provide the IP Subnet/Range for volumes")
					netPermission.Ip = pkg.PromptGetInput("IP", errors.New("unable to get the IP"), pkg.IsString)

					netPermission.Permission = pkg.PromptGetSelect([]string{"READ_ONLY", "READ_WRITE"}, "Please select the permission type to access the volume?")

					rs := pkg.PromptGetInput("Do you want to provide access for root user to the volumes? (Y/N)", errors.New("invalid input"), pkg.IsString)

					if rs == "Y" || rs == "y" {
						netPermission.RootSquash = true
					}

					csi.netPermissions = append(csi.netPermissions, netPermission)
					netPerms = pkg.PromptGetInput("Do you want to configure another Net permissions (Y/N)", errors.New("invalid input"), pkg.IsString)

					if netPerms == "Y" || netPerms == "y" {
						continue
					}
					break
				}
			}
		}
		err = createVDOConfig(cl, ctx, cpi, csi)
		if err != nil {
			panic(err)
		}
		fmt.Println("Thanks For configuring VDO. The drivers should be installed/configured soon")

	},
}

func init() {
	configureCmd.AddCommand(driversCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// vcCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// vcCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func createSecret(clientset *kubernetes.Clientset, ctx context.Context, cred credentials, driver string) (v1.Secret, error) {

	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-creds-%s", cred.vcIp, driver),
			Namespace: KubeSystemNamespace,
		},
		Type: "kubernetes.io/basic-auth",

		Data: map[string][]byte{
			"username": []byte(cred.username),
			"password": []byte(cred.password),
		},
	}

	_, err := clientset.CoreV1().Secrets(KubeSystemNamespace).Create(ctx, &secret, metav1.CreateOptions{})
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
				ClusterDistribution: "",
				FileVolumes: v1alpha1.FileVolume{
					VSanDataStoreUrl: csi.vSANDataStoresUrl,
					NetPermissions:   csi.netPermissions,
				},
			},
		},
		Status: v1alpha1.VDOConfigStatus{
			CPIStatus: v1alpha1.CPIStatus{},
			CSIStatus: v1alpha1.CSIStatus{},
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

func getVCIP(cred *credentials, labels credentials, driver string) {
	fmt.Printf("Please provide the VC_IP for configuring %s \n", driver)
	vcIp := pkg.PromptGetInput(labels.vcIp, errors.New("unable to get the VC_IP"), pkg.IsIP)
	cred.vcIp = vcIp

	res := pkg.PromptGetInput("Do you want to establish a secure connection? (Y/N)", errors.New("invalid input"), pkg.IsString)
	if res == "Y" || res == "y" {
		fmt.Println("Please provide the SSL Thumbprint")
		thumbprint := pkg.PromptGetInput(labels.thumbprint, errors.New("invalid input"), pkg.IsString)
		cred.thumbprint = thumbprint
	} else {
		cred.insecure = true
	}
}

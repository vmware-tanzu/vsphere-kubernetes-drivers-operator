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
package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/vdoctl/pkg"
	"strings"

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
	errorMsg          string
	username          string
	password          string
	vcIp              string
	insecure          bool
	datacenters       []string
	vSANDataStoresUrl []string
	topology          v1alpha1.TopologyInfo
	netPermissions    []v1alpha1.NetPermission
}

//type cloudProvider struct {
//	credentials credentials
//	topology          v1alpha1.TopologyInfo
//}
//
//type storageProvider struct {
//	credentials credentials
//	vSANDataStoresUrl []string
//	netPermissions    []v1alpha1.NetPermission
//
//}

const (
	VdoNamespace = "vmware-system-vdo"
	GroupName    = "vdo.vmware.com"
	GroupVersion = "v1alpha1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
)

// vcCmd represents the vc command
var vcCmd = &cobra.Command{
	Use:   "vc",
	Short: "command to configure VC",
	Long: `This command helps to specify the details required to configure VDO for
a vcenter.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		cpi := credentials{}
		csi := credentials{}

		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
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
		}
		fmt.Println("Please provide the VC_IP")
		vcIp := pkg.PromptGetInput(labels.vcIp, errors.New("unable to get the VC_IP"), false, false, true)
		csi.vcIp = vcIp

		res := pkg.PromptGetInput("Do you want to establish a secure connection? (Y/N)", errors.New("invalid input"), false, false, false)

		if res == "Y" || res == "y" {
			cpi.insecure = false
			csi.insecure = false
		}

		if err != nil {
			panic(err)
		}

		isCPIRequired := pkg.PromptGetInput("Do you want this VC to be Cloud Provider configured? (Y/N)", errors.New("invalid input"), false, false, false)

		var vsphereCloudConfigList []string

		if isCPIRequired == "Y" || isCPIRequired == "y" {
			cpi.vcIp = vcIp
			vsphereCloudConfigList = []string{vcIp}

		multivcloop:
			for {
				fmt.Println("Please provide the credentials for CPI")
				cpi.username = pkg.PromptGetInput(labels.username, errors.New("unable to get the username"), false, false, false)

				cpi.password = pkg.PromptGetInput(labels.password, errors.New("unable to get the password"), false, true, false)

				dc := pkg.PromptGetInput("Datacenters", errors.New("unable to get the datacenters"), false, false, false)

				cpi.datacenters = strings.SplitAfter(dc, ",")

				secret, err := createSecret(clientset, ctx, cpi, "cpi")
				if err != nil {
					panic(err)
				}

				err = createVsphereCloudConfig(cl, ctx, cpi, secret.Name, "cpi")
				if err != nil {
					panic(err)
				}

				multiVC := pkg.PromptGetInput("Do you want to configure another VC for CPI? (Y/N)", errors.New("invalid input"), false, false, false)

				if multiVC == "Y" || multiVC == "y" {
					cpi = credentials{}

					fmt.Println("Please provide the VC_IP")
					vcIp := pkg.PromptGetInput(labels.vcIp, errors.New("unable to get the VC_IP"), false, false, true)
					cpi.vcIp = vcIp

					vsphereCloudConfigList = append(vsphereCloudConfigList, vcIp)

					res := pkg.PromptGetInput("Do you want to establish a secure connection? (Y/N)", errors.New("invalid input"), false, false, false)
					if res == "Y" || res == "y" {
						cpi.insecure = false
					}

					continue multivcloop

				}
				break

			}

			topology := pkg.PromptGetInput("Do you want to configure zones/regions for CPI? (Y/N)", errors.New("invalid input"), false, false, false)

			if topology == "Y" || topology == "y" {
				cpi.topology.Zone = pkg.PromptGetInput(labels.topology.Zone, errors.New("unable to get the zones"), false, false, false)
				cpi.topology.Region = pkg.PromptGetInput(labels.topology.Region, errors.New("unable to get the regions"), false, false, false)
			}

		}

		fmt.Println("You have now completed configuration of Cloud Provider. We will now proceed to configure Storage Provider. \n Please provide the credentials for Storage Provider")

		csi.username = pkg.PromptGetInput(labels.username, errors.New("unable to get the username"), false, false, false)

		csi.password = pkg.PromptGetInput(labels.password, errors.New("unable to get the password"), false, true, false)

		dc := pkg.PromptGetInput("Datacenters", errors.New("unable to get the datacenters"), false, false, false)

		csi.datacenters = strings.SplitAfter(dc, ",")

		secret, err := createSecret(clientset, ctx, csi, "csi")

		if err != nil {
			panic(err)
		}

		err = createVsphereCloudConfig(cl, ctx, csi, secret.Name, "csi")
		if err != nil {
			panic(err)
		}

		advConfig := pkg.PromptGetInput("Do you want advanced configuration for CSI? (Y/N)", errors.New("invalid input"), false, false, false)

		if advConfig == "Y" || advConfig == "y" {
			vsanDSurl := pkg.PromptGetInput("vSANDataStoresUrl", errors.New("unable to get the vSANDataStoresUrl"), false, false, false)
			csi.vSANDataStoresUrl = strings.SplitAfter(vsanDSurl, ",")

			for {
				netpermission := v1alpha1.NetPermission{}
				netpermission.Ip = pkg.PromptGetInput("IP", errors.New("unable to get the IP"), false, false, true)

				netpermission.Permission = promptGetSelect([]string{"READ_ONLY", "READ_WRITE"}, "Please select the permission type to access the volume?")

				rs := pkg.PromptGetInput("Do you want to provide access for root user to the volumes? (Y/N)", errors.New("invalid input"), false, false, false)

				if rs == "Y" || rs == "y" {
					netpermission.RootSquash = true
				}

				csi.netPermissions = append(csi.netPermissions, netpermission)
				netPerms := pkg.PromptGetInput("Do you want to configure another netpermissions (Y/N)", errors.New("invalid input"), false, false, false)

				if netPerms == "Y" || netPerms == "y" {
					continue
				}
				break
			}
		}
		err = createVDOConfig(cl, ctx, vsphereCloudConfigList, cpi, csi)
		if err != nil {
			panic(err)
		}

	},
}

func init() {
	rootCmd.AddCommand(vcCmd)

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
			Namespace: "kube-system",
		},
		Type: "kubernetes.io/basic-auth",

		Data: map[string][]byte{
			"username": []byte(cred.username),
			"password": []byte(cred.password),
		},
	}

	_, err := clientset.CoreV1().Secrets("kube-system").Create(ctx, &secret, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return secret, nil
		}
	}
	return secret, err

}

func createVsphereCloudConfig(cl client.Client, ctx context.Context, cred credentials, secretName string, driver string) error {
	vcc := &v1alpha1.VsphereCloudConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", cred.vcIp, driver),
			Namespace: VdoNamespace,
		},
		Spec: v1alpha1.VsphereCloudConfigSpec{
			VcIP:        fmt.Sprintf(cred.vcIp),
			Insecure:    cred.insecure,
			Credentials: secretName,
			DataCenters: cred.datacenters,
		},
		Status: v1alpha1.VsphereCloudConfigStatus{},
	}

	err := cl.Create(ctx, vcc, &runtimeclient.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)

	}
	return err
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

func createVDOConfig(cl client.Client, ctx context.Context, vsphereCloudConfigList []string, cpi credentials, csi credentials) error {

	vdoConfig := &v1alpha1.VDOConfig{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vdo-config",
			Namespace: VdoNamespace,
		},
		Spec: v1alpha1.VDOConfigSpec{
			CloudProvider: v1alpha1.CloudProviderConfig{
				VsphereCloudConfigs: vsphereCloudConfigList,
				Topology:            cpi.topology,
			},
			StorageProvider: v1alpha1.StorageProviderConfig{
				VsphereCloudConfig:  csi.vcIp,
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

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
	//"github.com/go-logr/logr"
	//"log"
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

	"github.com/manifoldco/promptui"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type credentials struct {
	errorMsg    string
	username    string
	password    string
	vcIp        string
	datacenters string
}

type topology struct {
	zone   string
	region string
}

const (
	VdoNamespace = "vmware-system-vdo"
	GroupName    = "vdo.vmware.com"
	GroupVersion = "v1alpha1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
	insecure           = true
)

// vcCmd represents the vc command
var vcCmd = &cobra.Command{
	Use:   "vc",
	Short: "command to configure VC",
	Long: `This command helps to specify the details required to configure VDO for
a vcenter.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
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

		fmt.Println("Please provide the VC_IP")

		cred := credentials{
			errorMsg:    "unable to get VC creds",
			username:    "Username",
			password:    "Password",
			vcIp:        "VC_IP",
			datacenters: "Datacenters",
		}

		t := topology{
			zone:   "",
			region: "",
		}

		vcIp := promptGetVCIP(cred)

		res := promptGetSelect([]string{"Yes", "No"}, "Do you want to establish a secure connection")
		if res == "Yes" {
			insecure = false
		}

		if err != nil {
			panic(err)
		}

		isCPIRequired := promptGetSelect([]string{"Yes", "No"}, "Do you want this VC to be CPI configured?")

		var vsphereCloudConfigList []string

		if isCPIRequired == "Yes" {
			vsphereCloudConfigList = []string{vcIp}

		multivcloop:
			for {

				fmt.Println("Please provide the credentials for CPI")

				vcusr, vcpwd, dc := promptGetCredentials(cred)
				cpi := credentials{
					errorMsg: "",
					username: vcusr,
					password: vcpwd,
					vcIp:     vcIp,
				}

				secret, err := createSecret(clientset, ctx, cpi)
				if err != nil {
					panic(err)
				}

				err = createVsphereCloudConfig(cl, ctx, vcIp, dc, secret.Name, insecure)
				if err != nil {
					panic(err)
				}

				multiVC := promptGetSelect([]string{"Yes", "No"}, "Do you want configure another VC for CPI?")

				//fmt.Print(multiVC)

				if multiVC == "Yes" {
					fmt.Println("Please provide the VC_IP")
					vcIp = promptGetVCIP(cred)

					vsphereCloudConfigList = append(vsphereCloudConfigList, vcIp)
					res := promptGetSelect([]string{"Yes", "No"}, "Do you want to establish a secure connection")
					if res == "Yes" {
						insecure = false
					}
					continue multivcloop

				}

			}

			//zones, regions := promptGetCredentials(cred)

		}

		fmt.Println("Please provide the credentials for CSI")
		vcusr, vcpwd, dc := promptGetCredentials(cred)

		csi := credentials{
			errorMsg: "",
			username: vcusr,
			password: vcpwd,
			vcIp:     vcIp,
		}

		secret, err := createSecret(clientset, ctx, csi)

		if err != nil {
			panic(err)
		}

		err = createVsphereCloudConfig(cl, ctx, vcIp, dc, secret.Name, insecure)
		if err != nil {
			panic(err)
		}

		err = createVDOConfig(cl, ctx, vsphereCloudConfigList, t, vcIp)
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

func promptGetCredentials(cred credentials) (string, string, []string) {
	validate := func(input string) error {
		if len(input) <= 0 {
			return errors.New(cred.errorMsg)
		}
		return nil
	}

	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     cred.username,
		Templates: templates,
		Validate:  validate,
	}

	usr, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	prompt = promptui.Prompt{
		Label:     cred.password,
		Templates: templates,
		Validate:  validate,
		Mask:      '*',
	}

	pwd, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	prompt = promptui.Prompt{
		Label:     cred.datacenters,
		Templates: templates,
		Validate:  validate,
	}

	dc, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	dcList := strings.SplitAfter(dc, ",")

	return usr, pwd, dcList
}

func promptGetVCIP(cred credentials) string {
	validate := func(input string) error {
		if len(input) <= 0 {
			return errors.New(cred.errorMsg)
		}
		return nil
	}

	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     cred.vcIp,
		Templates: templates,
		Validate:  validate,
	}

	vcIp, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	return vcIp
}

func createSecret(clientset *kubernetes.Clientset, ctx context.Context, cpi credentials) (v1.Secret, error) {

	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-creds", cpi.vcIp),
			Namespace: "kube-system",
		},
		Type: "kubernetes.io/basic-auth",

		Data: map[string][]byte{
			"username": []byte(cpi.username),
			"password": []byte(cpi.password),
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

func createVsphereCloudConfig(cl client.Client, ctx context.Context, vcIP string, dc []string, creds string, insecure bool) error {
	vcc := &v1alpha1.VsphereCloudConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vcIP,
			Namespace: VdoNamespace,
		},
		Spec: v1alpha1.VsphereCloudConfigSpec{
			VcIP:        vcIP,
			Insecure:    insecure,
			Credentials: creds,
			DataCenters: dc,
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

func createVDOConfig(cl client.Client, ctx context.Context, vsphereCloudConfigList []string, t topology, vsphereCloudConfig string) error {
	vdoConfig := &v1alpha1.VDOConfig{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vdoconfig",
			Namespace: VdoNamespace,
		},
		Spec: v1alpha1.VDOConfigSpec{
			CloudProvider: v1alpha1.CloudProviderConfig{
				VsphereCloudConfigs: vsphereCloudConfigList,
				Topology: v1alpha1.TopologyInfo{
					Zone:   t.zone,
					Region: t.region,
				},
			},
			StorageProvider: v1alpha1.StorageProviderConfig{
				VsphereCloudConfig:  vsphereCloudConfig,
				ClusterDistribution: "",
				FileVolumes:         v1alpha1.FileVolume{},
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

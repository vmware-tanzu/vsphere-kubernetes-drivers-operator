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
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/viper"
)

const (
	VdoNamespace = "vmware-system-vdo"
	GroupName    = "vdo.vmware.com"
	GroupVersion = "v1alpha1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
	cfgFile            string
	kubeconfig         string
	K8sClientset       *kubernetes.Clientset
	K8sClient          client.Client
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "vdoctl",
	Short: "VDO Command Line",
	Long: `vdoctl is a command line interface for vSphere Kubernetes Drivers Operator.
vdoctl provides the user with basic set of commands required to install and configure VDO.
For example:
vdoctl deploy
vdoctl configure compat
vdoctl store creds
vdoctl configure vc
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func GenerateMarkdownDoc(docPath string) {
	rootCmd.DisableAutoGenTag = true
	err := doc.GenMarkdownTree(rootCmd, docPath)
	if err != nil {
		cobra.CheckErr(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.vdoctl.yaml)")

	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "points to the kubeconfig file of the target k8s cluster")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".vdoctl" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".vdoctl")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	if len(kubeconfig) <= 0 {
		kubeconfig = os.Getenv("KUBECONFIG")
		if len(kubeconfig) <= 0 {
			cobra.CheckErr(errors.New("could not detect a target kubernetes cluster. " +
				"Either use --kubeconfig flag or set KUBECONFIG environment variable"))
		}
	}

	err := generateK8sClient(kubeconfig)
	if err != nil {
		cobra.CheckErr(err)
	}

}

func generateK8sClient(kubeconfig string) error {
	clientConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return errors.New("Failed to generate client from provided kubeconfig")
	}

	K8sClientset, err = kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return err
	}

	K8sClient, _ = client.New(clientConfig, client.Options{
		Scheme: scheme.Scheme,
	})

	err = AddToScheme(scheme.Scheme)
	if err != nil {
		return err
	}

	return nil

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

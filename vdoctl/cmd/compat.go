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
	"fmt"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/vdoctl/pkg"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	CompatMatrixConfigMAp = "compat-matrix-config"
)

// compatCmd represents the compat command
var compatCmd = &cobra.Command{
	Use:   "compat",
	Short: "Compatibility matrix of VDO",
	Long:  `This command helps to configure compatiblity matrix for VDO`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("compat called")
		ctx := context.Background()

		config, err := buildConfig()
		if err != nil {
			panic(err)
		}

		client, err := runtimeclient.New(config, client.Options{
			Scheme: scheme.Scheme,
		})
		if err != nil {
			panic(err)
		}

		err = CreateNamespace(client, ctx)
		if err != nil {
			panic(err)
		}

		item := pkg.PromptGetSelect([]string{"local filepath", "fileURL"}, "Please select the mode for providing compat-matrix")

		flag := pkg.IsString
		if item == "fileURL" {
			flag = pkg.IsURL
		}
		filePath := pkg.PromptGetInput(item, errors.New("invalid input"), flag)

		err = CreateConfigMap(filePath, client, ctx)
		if err != nil {
			panic(err)
		}

	},
}

func init() {
	configureCmd.AddCommand(compatCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// compatCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// compatCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func CreateConfigMap(filepath string, clientset client.Client, ctx context.Context) error {

	configMapKey := types.NamespacedName{
		Namespace: VdoNamespace,
		Name:      CompatMatrixConfigMAp,
	}

	data := map[string]string{"versionConfigURL": filepath, "auto-upgrade": "disabled"}

	configMapObj := metav1.ObjectMeta{Name: configMapKey.Name, Namespace: configMapKey.Namespace}
	vsphereConfigMap := v1.ConfigMap{Data: data, ObjectMeta: configMapObj}

	err := clientset.Create(ctx, &vsphereConfigMap, &runtimeclient.CreateOptions{})

	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
	}
	return err
}

func CreateNamespace(clientset client.Client, ctx context.Context) error {

	nsSpec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: VdoNamespace,
		},
	}
	err := clientset.Create(ctx, nsSpec, &runtimeclient.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
	}
	return err
}

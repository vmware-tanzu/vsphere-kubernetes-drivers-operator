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

	vdoClient "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/client"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/vdoctl/pkg/utils"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	CompatMatrixConfigMAp = "compat-matrix-config"
	LocalFilepath         = "Local filepath"
	WebURL                = "Web URL"
)

// compatCmd represents the compat command
var compatCmd = &cobra.Command{
	Use:   "compat",
	Short: "Command to configure compatibility matrix of VDO",
	Long:  `This command helps to configure compatibility matrix for VDO`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		configKey := types.NamespacedName{
			Namespace: VdoNamespace,
			Name:      CompatMatrixConfigMap,
		}
		cm := v1.ConfigMap{}

		err := K8sClient.Get(ctx, configKey, &cm)
		if err == nil {
			cobra.CheckErr(errors.New("you have already configured the compatibility matrix. Check 'vdoctl update matrix' to update the existing one"))
		}

		item := utils.PromptGetSelect([]string{LocalFilepath, WebURL}, "Please select the mode for providing compat-matrix")

		flag := utils.IsString
		if item == WebURL {
			flag = utils.IsURL
		}
		filePath := utils.PromptGetInput(item, errors.New("invalid input"), flag)

		err = CreateNamespace(K8sClient, ctx)
		if err != nil {
			cobra.CheckErr(err)
		}

		err = CreateConfigMap(filePath, K8sClient, ctx, flag)
		if err != nil {
			cobra.CheckErr(err)
		}

	},
}

func init() {
	configureCmd.AddCommand(compatCmd)
}

func CreateConfigMap(filepath string, client runtimeclient.Client, ctx context.Context, flag utils.ValidationFlags) error {

	configMapKey := types.NamespacedName{
		Namespace: VdoNamespace,
		Name:      CompatMatrixConfigMAp,
	}
	var data map[string]string

	if flag == utils.IsURL {
		data = map[string]string{"versionConfigURL": filepath, "auto-upgrade": "disabled"}
	} else {
		fileBytes, err := vdoClient.GenerateYamlFromFilePath(filepath)
		if err != nil {
			cobra.CheckErr(fmt.Sprintf("unable to read the matrix from %s", filepath))
		}
		data = map[string]string{"versionConfigContent": string(fileBytes), "auto-upgrade": "disabled"}
	}

	configMapObj := metav1.ObjectMeta{Name: configMapKey.Name, Namespace: configMapKey.Namespace}
	vsphereConfigMap := v1.ConfigMap{Data: data, ObjectMeta: configMapObj}

	err := client.Create(ctx, &vsphereConfigMap, &runtimeclient.CreateOptions{})

	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = client.Update(ctx, &vsphereConfigMap)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return err
}

func CreateNamespace(client runtimeclient.Client, ctx context.Context) error {

	nsSpec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: VdoNamespace,
		},
	}

	err := client.Create(ctx, nsSpec, &runtimeclient.CreateOptions{})

	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
	}
	return err
}

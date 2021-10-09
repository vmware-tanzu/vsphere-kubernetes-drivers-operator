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
	"fmt"
	"github.com/spf13/cobra"
	vdov1alpha1 "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	v12 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const VDO_NOT_DEPLOYED = "VDO is not deployed. you can run `vdoctl deploy` command to deploy VDO"

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "command to get VDO status",
	Long:  "This command helps to get the status of the configurations created by VDO.\nIt includes brief detail about the status of CloudProvider and StorageProvider and Node details",
	Run: func(cmd *cobra.Command, args []string) {

		var vsphereCloudConfigList vdov1alpha1.VsphereCloudConfigList
		var vdoConfigList vdov1alpha1.VDOConfigList

		ctx := context.Background()

		err, _ := IsVDODeployed(ctx)
		if err != nil {
			if apierrors.IsNotFound(err) {
				fmt.Println(VDO_NOT_DEPLOYED)
				return
			} else {
				cobra.CheckErr(err)
			}
		}

		err = K8sClient.List(ctx, &vsphereCloudConfigList)
		if err != nil {
			cobra.CheckErr(err)
		}

		err = K8sClient.List(ctx, &vdoConfigList)
		if err != nil {
			cobra.CheckErr(err)
		}

		if len(vdoConfigList.Items) <= 0 {
			fmt.Println("VDO is not configured. you can use `vdoctl configure drivers` to configure VDO")
			return
		}

		// Fetch the first element from vdoConfigList, since we have a single vdoConfig
		vdoConfig := vdoConfigList.Items[0]

		// Display CloudProvider Details
		for _, vsphereCloudConfigName := range vdoConfig.Spec.CloudProvider.VsphereCloudConfigs {
			fmt.Printf("CloudProvider   : %s", vdoConfig.Status.CPIStatus.Phase)
			fetchVcenterIp(vsphereCloudConfigList, vsphereCloudConfigName)
		}

		if len(vdoConfig.Status.CPIStatus.NodeStatus) > 0 {
			fmt.Printf("\t Nodes : ")
		}

		for nodeName, status := range vdoConfig.Status.CPIStatus.NodeStatus {
			fmt.Printf("\n\t\t %s : %s ", nodeName, status)
		}

		// Display StorageProvider Details
		fmt.Printf("\nStorageProvider : %s", vdoConfig.Status.CSIStatus.Phase)
		fetchVcenterIp(vsphereCloudConfigList, vdoConfig.Spec.StorageProvider.VsphereCloudConfig)
	},
}

func IsVDODeployed(ctx context.Context) (error, *v12.Deployment) {
	deployment := &v12.Deployment{}
	ns := types.NamespacedName{Namespace: VdoNamespace, Name: VdoDeploymentName}
	err := K8sClient.Get(ctx, ns, deployment)
	return err, deployment
}

// Fetch VC IP of given VsphereCloudConfig
func fetchVcenterIp(vsphereCloudConfigList vdov1alpha1.VsphereCloudConfigList, configName string) {
	for _, vsphereCloudConfig := range vsphereCloudConfigList.Items {
		if configName == vsphereCloudConfig.Name {
			fmt.Printf("\n\t vCenter : ")
			if vsphereCloudConfig.Status.Config == vdov1alpha1.VsphereConfigVerified {
				fmt.Printf("\n\t\t%s  (%s)\n", vsphereCloudConfig.Spec.VcIP, "Credentials Verified")
			} else {
				fmt.Printf("\n\t\t%s  (%s)\n", vsphereCloudConfig.Spec.VcIP, vsphereCloudConfig.Status.Message)
			}
			break
		}
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

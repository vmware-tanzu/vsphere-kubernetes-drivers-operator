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
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "command to get VDO status",
	Long: `This command helps to get the status of the configurations created by VDO.
            It includes brief detail about the status of CloudProvider and StorageProvider
             and Node details`,
	Run: func(cmd *cobra.Command, args []string) {

		var vsphereCloudConfigList vdov1alpha1.VsphereCloudConfigList
		var vdoConfigList vdov1alpha1.VDOConfigList

		ctx := context.Background()

		err := K8sClient.List(ctx, &vsphereCloudConfigList)
		if err != nil {
			cobra.CheckErr(err)
		}

		err = K8sClient.List(ctx, &vdoConfigList)
		if err != nil {
			cobra.CheckErr(err)
		}

		// Fetch the first element from vdoConfigList, since we have a single vdoConfig
		vdoConfig := vdoConfigList.Items[0]

		// Display CloudProvider Details
		fmt.Printf("CloudProvider   : %s", vdoConfig.Status.CPIStatus.Phase)
		for _, vsphereCloudConfigName := range vdoConfig.Spec.CloudProvider.VsphereCloudConfigs {
			fetchVcenterIp(vsphereCloudConfigList, vsphereCloudConfigName)
		}

		fmt.Printf("\t Node : ")
		for nodeName, status := range vdoConfig.Status.CPIStatus.NodeStatus {
			fmt.Printf("\n\t\t %s : %s ", nodeName, status)
		}

		// Display StorageProvider Details
		fmt.Printf("\nStorageProvider : %s", vdoConfig.Status.CSIStatus.Phase)
		fetchVcenterIp(vsphereCloudConfigList, vdoConfig.Spec.StorageProvider.VsphereCloudConfig)
	},
}

// Fetch VC IP of given VsphereCloudConfig
func fetchVcenterIp(vsphereCloudConfigList vdov1alpha1.VsphereCloudConfigList, configName string) {
	for _, vsphereCloudConfig := range vsphereCloudConfigList.Items {
		fmt.Printf("\n\t vCenter : ")
		if configName == vsphereCloudConfig.Name {
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deployCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deployCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

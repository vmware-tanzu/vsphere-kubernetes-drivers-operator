/*
Copyright © 2021

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
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"github.com/spf13/cobra"
	vdoClient "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/client"
)

var CompatMatrixConfigMap = "compat-matrix-config"

// matrixUpdateCmd represents the driver command
var matrixUpdateCmd = &cobra.Command{
	Use:     "compatibility-matrix",
	Aliases: []string{"cm"},
	Short:   "Command to update the Compatibility matrix of Drivers",
	Long: `This command helps to update the Compatibility matrix of Drivers, 
which in turns help to upgrade/downgrade the versions of CSI & CPI drivers.`,
	Example: "vdoctl update compatibility-matrix https://github.com/demo/demo.yaml\nvdoctl update compatibility-matrix file://var/sample/sample.yaml",

	Run: func(cmd *cobra.Command, args []string) {
		cobra.MinimumNArgs(1)
		updateMatrix(args)
	},
}

// updateMatrix updates the ConfigMap containing the compatibility-matrix
func updateMatrix(args []string) {

	if len(args) < 1 {
		cobra.CheckErr("At-least one argument should be provided")
	}

	updatedMatrix := args[0]
	ctxNew := context.Background()

	// Check the vdoDeployment Namespace and confirm if VDO operator is running in the env
	err := getVdoNamespace(ctxNew)
	if err != nil {
		cobra.CheckErr(err)
	}

	// Check for volumes which have PWX or ROX access mode,
	// If any then manual steps are required before updating the driver
	volumeAttachmentList := storagev1.VolumeAttachmentList{}
	err = K8sClient.List(ctxNew, &volumeAttachmentList)
	if err != nil {
		cobra.CheckErr("unable to read the  volume list to do pre-check for upgrade")
	}
	var hasRWXROXVolumes bool
	var pvlistWithRWXROX []string
	for _, volumeAttachment := range volumeAttachmentList.Items {
		volumeSpecModeList := volumeAttachment.Spec.Source.InlineVolumeSpec.AccessModes

		for _, mode := range volumeSpecModeList {
			if mode == "ReadOnlyMany" || mode == "ReadWriteMany" {
				hasRWXROXVolumes = true
				pvlistWithRWXROX = append(pvlistWithRWXROX, *volumeAttachment.Spec.Source.PersistentVolumeName)
				break
			}
		}
	}
	if hasRWXROXVolumes {
		cobra.CheckErr(fmt.Sprintf("There are exisiting PV's attached with RWX | ROX mode %s"+
			"please follow CSI documentation to update the CSI https://vsphere-csi-driver.sigs.k8s.io/driver-deployment/upgrade.html ",
			pvlistWithRWXROX))
	}

	err = updateConfigMap(updatedMatrix, ctxNew)

	if err != nil {
		cobra.CheckErr(fmt.Sprintf("unable to read the updated matrix from %s", updatedMatrix))
	}

	fmt.Println("Compatibility matrix has been updated successfully.")
}

func updateConfigMap(filepath string, ctx context.Context) error {

	var err error
	var data map[string]string

	configMetaData := types.NamespacedName{
		Namespace: VdoCurrentNamespace,
		Name:      CompatMatrixConfigMap,
	}

	if strings.HasPrefix(filepath, "https://") || strings.HasPrefix(filepath, "http://") {
		data = map[string]string{"versionConfigURL": filepath, "auto-upgrade": "disabled"}
	} else {
		fileBytes, err := vdoClient.GenerateYamlFromFilePath(filepath)
		if err != nil {
			cobra.CheckErr(fmt.Sprintf("unable to read the updated matrix from %s", filepath))
		}
		data = map[string]string{"versionConfigContent": string(fileBytes), "auto-upgrade": "disabled"}
	}

	configMapObj := metav1.ObjectMeta{Name: configMetaData.Name, Namespace: configMetaData.Namespace}
	vsphereConfigMap := v1.ConfigMap{Data: data, ObjectMeta: configMapObj}

	err = K8sClient.Update(ctx, &vsphereConfigMap, &client.UpdateOptions{})

	if err != nil {
		cobra.CheckErr(fmt.Sprintf("Error received in updating config Map  %s", err))
	}
	return err
}

func init() {
	// Add the sub-command 'matrix' to update command
	updateCmd.AddCommand(matrixUpdateCmd)
}

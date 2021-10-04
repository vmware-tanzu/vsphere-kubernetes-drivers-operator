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
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	vdov1alpha1 "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/controllers"
	dynclient "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/client"
	vdocontext "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/context"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

const (
	VdoDeploymentName = "vdo-controller-manager"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "command to get VDO version",
	Long: `This command helps to get the version of the configurations created by VDO.
            It includes brief detail about the version of CloudProvider and StorageProvider
             along with VC and Kubernetes details`,
	Run: func(cmd *cobra.Command, args []string) {

		var vdoConfigList vdov1alpha1.VDOConfigList

		ctx := vdocontext.VDOContext{
			Context: context.Background(),
			Logger:  ctrllog.Log.WithName("vdoctl:version"),
		}

		err := K8sClient.List(ctx, &vdoConfigList)
		if err != nil {
			cobra.CheckErr(err)
		}

		// Fetch the first element from vdoConfigList, since we have a single vdoConfig
		vdoConfig := vdoConfigList.Items[0]

		s := scheme.Scheme
		s.AddKnownTypes(vdov1alpha1.GroupVersion, &vdov1alpha1.VDOConfig{})

		r := controllers.VDOConfigReconciler{
			Client:       K8sClient,
			Logger:       ctrllog.Log.WithName("vdoctl:version"),
			Scheme:       s,
			ClientConfig: ClientConfig,
		}

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      CompatMatrixConfigMAp,
				Namespace: VdoNamespace,
			},
		}

		deploymentReq := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      VdoDeploymentName,
				Namespace: VdoNamespace,
			},
		}

		// Fetch VDO deployment object
		deployment := &v12.Deployment{}
		err = K8sClient.Get(ctx, deploymentReq.NamespacedName, deployment)
		if err != nil {
			cobra.CheckErr(err)
		}

		var deploymentData string

		for _, v := range deployment.Annotations {
			if strings.Contains(v, "image") {
				deploymentData = v
				break
			}
		}

		var deploymentMap map[string]interface{}
		if err = json.Unmarshal([]byte(deploymentData), &deploymentMap); err != nil {
			cobra.CheckErr(err)
		}

		containerInfo := deploymentMap["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})

		var imageName interface{}
		for i, container := range containerInfo {
			if container.(map[string]interface{})["name"] == "manager" {
				imageName = containerInfo[i].(map[string]interface{})["image"]
				break
			}
		}

		vdoVersion := strings.Split(fmt.Sprint(imageName), ":")

		fmt.Printf("VDO Version        : %s", vdoVersion[len(vdoVersion)-1])

		configMap := &v1.ConfigMap{}
		err = K8sClient.Get(ctx, req.NamespacedName, configMap)
		if err != nil {
			cobra.CheckErr(err)
		}

		matrixConfig, err := dynclient.ParseMatrixYaml(configMap.Data["versionConfigURL"])
		if err != nil {
			cobra.CheckErr(err)
		}

		vSphereVersion, _ := r.FetchVsphereVersions(ctx, req, &vdoConfig)
		fmt.Printf("\nvSphere Versions   : %s", vSphereVersion)

		k8sVersion, _ := r.Fetchk8sVersions(ctx)
		fmt.Printf("\nkubernetes Version : %s", k8sVersion)

		err = r.FetchCsiDeploymentYamls(ctx, matrixConfig, vSphereVersion, k8sVersion)
		if err != nil {
			cobra.CheckErr(err)
		}
		fmt.Printf("\nCSI Version        : %s", r.CurrentCSIDeployedVersion)

		if len(vdoConfig.Spec.CloudProvider.VsphereCloudConfigs) > 0 {
			err = r.FetchCpiDeploymentYamls(ctx, matrixConfig, vSphereVersion, k8sVersion)
			if err != nil {
				cobra.CheckErr(err)
			}
			fmt.Printf("\nCPI Version        : %s", r.CurrentCPIDeployedVersion)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deployCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deployCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

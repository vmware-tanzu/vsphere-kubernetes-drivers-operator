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
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/models"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

var (
	vdoVersion     = "Not Configured"
	vsphereVersion []string
	k8sVersion     = "Not Configured"
	csiVersion     = "Not Configured"
	cpiVersion     = "Not Configured"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "command to get VDO version",
	Long:  "This command helps to get the version of the configurations created by VDO.\nIt includes brief detail about the version of CloudProvider and StorageProvider along with VC and Kubernetes details",
	Run: func(cmd *cobra.Command, args []string) {

		ctx := vdocontext.VDOContext{
			Context: context.Background(),
			Logger:  ctrllog.Log.WithName("vdoctl:version"),
		}

		// // Confirm if VDO operator is running in the env and get the vdoDeployment Namespace
		err, _ := IsVDODeployed(ctx)
		if err != nil {
			cobra.CheckErr(err)
		}

		k8sVersion = getK8sVersion()
		err, vdoDeployment := IsVDODeployed(ctx)
		if err != nil {
			if apierrors.IsNotFound(err) {
				showVersionInfo()
				return
			} else {
				cobra.CheckErr(err)
			}
		}

		vdoVersion = getVdoVersion(vdoDeployment)

		var vdoConfigList vdov1alpha1.VDOConfigList
		err = K8sClient.List(ctx, &vdoConfigList)
		if err != nil {
			cobra.CheckErr(err)
		}

		if len(vdoConfigList.Items) <= 0 {
			showVersionInfo()
			return
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
				Namespace: VdoCurrentNamespace,
			},
		}

		configMap := &v1.ConfigMap{}
		err = K8sClient.Get(ctx, req.NamespacedName, configMap)
		if err != nil {
			cobra.CheckErr(err)
		}

		var matrixConfig models.CompatMatrix
		if matrixConfigUrl, ok := configMap.Data["versionConfigURL"]; ok {
			matrixConfig, err = dynclient.ParseMatrixYaml(matrixConfigUrl)
		} else {
			err = json.Unmarshal([]byte(configMap.Data["versionConfigContent"]), &matrixConfig)
		}

		if err != nil {
			cobra.CheckErr(err)
		}

		vsphereVersion, _ = r.FetchVsphereVersions(ctx, req, &vdoConfig)

		err = r.FetchCsiDeploymentYamls(ctx, matrixConfig, vsphereVersion, k8sVersion)
		if err != nil {
			cobra.CheckErr(err)
		}
		csiVersion = r.CurrentCSIDeployedVersion

		if len(vdoConfig.Spec.CloudProvider.VsphereCloudConfigs) > 0 {
			err = r.FetchCpiDeploymentYamls(ctx, matrixConfig, vsphereVersion, k8sVersion)
			if err != nil {
				cobra.CheckErr(err)
			}
			cpiVersion = r.CurrentCPIDeployedVersion
		}
		showVersionInfo()
	},
}

func getK8sVersion() string {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(ClientConfig)
	if err != nil {
		cobra.CheckErr(err)
	}

	k8sServerVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		cobra.CheckErr(err)
	}

	return k8sServerVersion.Major + "." + k8sServerVersion.Minor
}

func getVdoVersion(vdoDeployment *v12.Deployment) string {
	containersList := vdoDeployment.Spec.Template.Spec.Containers
	var containerImage string
	for _, container := range containersList {
		if strings.Contains(container.Name, "manager") {
			containerImage = container.Image
			break
		}
	}
	if containerImage == "" {
		cobra.CheckErr("Unable to find the VDO manager container")
	}
	vdoVersionInfo := strings.Split(fmt.Sprint(containerImage), ":")
	return vdoVersionInfo[len(vdoVersionInfo)-1]
}

func showVersionInfo() {
	fmt.Printf("kubernetes Version : %s", k8sVersion)
	fmt.Printf("\nVDO Version        : %s", vdoVersion)
	fmt.Printf("\nvSphere Versions   : %s", vsphereVersion)
	fmt.Printf("\nCSI Version        : %s", csiVersion)
	fmt.Printf("\nCPI Version        : %s\n", cpiVersion)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

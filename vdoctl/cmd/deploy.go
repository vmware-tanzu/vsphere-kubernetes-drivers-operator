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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	dynclient "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/client"
	vdocontext "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/context"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

type platform string

const (
	openshift platform = "openshift"
)

var specfile string

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy --spec <path to spec file> (can be http or file based url's)",
	Short: "Deploy vSphere Kubernetes Driver Operator",
	Long: `This command helps to deploy VDO on the kubernetes cluster targeted by --kubeconfig flag or KUBECONFIG environment variable.
Currently the command supports deployment on vanilla k8s cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var fileBytes []byte

		ctx := vdocontext.VDOContext{
			Context: context.Background(),
			Logger:  ctrllog.Log.WithName("vdoctl:deploy"),
		}

		k8sPlatform := "" // promptGetSelect([]string{"vanilla", "openshift"}, "Please select the flavour of your k8s cluster")
		//todo need to add suppport for openshift clusters.
		if k8sPlatform == string(openshift) {
			panic(errors.New("Deploy command does not support openshift cluster at the moment"))
		}

		if strings.Contains(specfile, "file://") {
			fileBytes, err = dynclient.GenerateYamlFromFilePath(specfile)
		} else {
			fileBytes, err = dynclient.GenerateYamlFromUrl(specfile)
		}
		if err != nil {
			cobra.CheckErr(fmt.Sprintf("unable to read deployment spec from %s", specfile))
		}

		_, applyErr := dynclient.ParseAndProcessK8sObjects(ctx, K8sClient, fileBytes, "")
		if applyErr != nil {
			cobra.CheckErr(applyErr)
		}
	},
}

func init() {
	deployCmd.Flags().StringVar(&specfile, "spec", "", "url to vdo deployment spec file")
	rootCmd.AddCommand(deployCmd)
}

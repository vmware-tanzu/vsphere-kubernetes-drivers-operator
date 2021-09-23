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
	//"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	//"github.com/stretchr/testify/assert"

	"path/filepath"

	"os"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/manifoldco/promptui"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type compatMatrix struct {
	errorMsg string
	input    string
}

var kubeconfig *string

// compatCmd represents the compat command
var compatCmd = &cobra.Command{
	Use:   "compat",
	Short: "Compatibility matrix of VDO",
	Long:  `This command helps to configure compatiblity matrix for VDO`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("compat called")
		ctx := context.Background()

		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "absolute path to the kubeconfig file")
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

		err = CreateNamespace(clientset, ctx)
		if err != nil {
			panic(err)
		}

		item := promptGetSelect([]string{"local filepath", "fileURL"}, "Please select the mode for providing compat-matrix")

		fmt.Print(item)
		filePath := promptGetCompatMat(compatMatrix{
			errorMsg: fmt.Sprintf("unable to fetch compat-matrix %s", item),
			input:    item,
		})

		err = CreateConfigMap(filePath, clientset, ctx)
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

func promptGetSelect(items []string, label string) string {
	index := -1
	var result string
	var err error

	for index < 0 {
		prompt := promptui.SelectWithAdd{
			Label: label,
			Items: items,
		}

		index, result, err = prompt.Run()

		if index == -1 {
			items = append(items, result)
		}
	}

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	return result
}

func promptGetCompatMat(cm compatMatrix) string {
	validate := func(input string) error {
		if len(input) <= 0 {
			return errors.New(cm.errorMsg)
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
		Label:     cm.input,
		Templates: templates,
		Validate:  validate,
	}

	res, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	return res
}

func CreateConfigMap(filepath string, clientset *kubernetes.Clientset, ctx context.Context) error {

	configMapKey := types.NamespacedName{
		Namespace: VdoNamespace,
		Name:      "compat-matrix-config",
	}

	data := map[string]string{"versionConfigURL": filepath, "auto-upgrade": "disabled"}

	configMapObj := metav1.ObjectMeta{Name: configMapKey.Name, Namespace: configMapKey.Namespace}
	vsphereConfigMap := v1.ConfigMap{Data: data, ObjectMeta: configMapObj}

	_, err := clientset.CoreV1().ConfigMaps(VdoNamespace).Create(ctx, &vsphereConfigMap, metav1.CreateOptions{})

	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
	}
	return err
}

func CreateNamespace(clientset *kubernetes.Clientset, ctx context.Context) error {

	nsSpec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: VdoNamespace,
		},
	}
	_, err := clientset.CoreV1().Namespaces().Create(ctx, nsSpec, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
	}
	return err
}

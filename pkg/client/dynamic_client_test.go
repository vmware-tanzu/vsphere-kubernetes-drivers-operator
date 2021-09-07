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

package client

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	vdocontext "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/context"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var testEnv *envtest.Environment

func TestMethod(t *testing.T) {
	RegisterFailHandler(Fail)
	//defer GinkgoRecover()
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}
	cfg, _ := testEnv.Start()
	Expect(cfg).NotTo(BeNil())

	k8sClient, err := client.New(cfg, client.Options{})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	vdoctx := vdocontext.VDOContext{
		Context: context.Background(),
		Logger:  klogr.New(),
	}

	yamlBytes, err := GenerateYamlFromUrl("https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/master/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml")
	if err != nil {
		t.Fatalf("error occurred in test %v", err)
	}

	Expect(yamlBytes).ShouldNot(BeEmpty())

	_, err = ParseAndProcessK8sObjects(vdoctx, k8sClient, yamlBytes, "")
	Expect(err).To(BeNil())

	//vsphere-cloud-controller-manager-ds.yaml
	yamlBytes, err = GenerateYamlFromUrl("https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/master/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml")
	Expect(err).To(BeNil())
	Expect(len(yamlBytes)).NotTo(BeZero())

	_, err = ParseAndProcessK8sObjects(vdoctx, k8sClient, yamlBytes, "")
	Expect(err).To(BeNil())
}

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

package controllers

import (
	"context"
	"crypto/tls"

	vdov1alpha1 "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	"github.com/vmware/govmomi/simulator"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

var _ = Describe("VsphereCloudConfig controller", func() {

	Context("When creating vSphereCloudConfig resource", func() {
		var vc_ip, vc_user, vc_pwd string
		var s *simulator.Server

		BeforeEach(func() {
			//Setup VC SIM
			model := simulator.VPX()
			model.Host = 0 // ClusterHost only

			defer model.Remove()
			err := model.Create()
			if err != nil {
				Expect(err).NotTo(HaveOccurred())
			}
			model.Service.TLS = new(tls.Config)

			s = model.Service.NewServer()
			vc_pwd, _ = s.URL.User.Password()
			vc_user = s.URL.User.Username()
			vc_ip = s.URL.Host
		})

		AfterEach(func() {
			defer s.Close()
		})

		It("should create the resource", func() {
			ctx := context.Background()

			cloudConfig := v1alpha1.VsphereCloudConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource1",
					Namespace: "default",
				},
				Spec: v1alpha1.VsphereCloudConfigSpec{
					VcIP:        "1.1.1.1",
					Insecure:    true,
					Credentials: "secret-ref",
					DataCenters: []string{"datacenter-1"},
				},
				Status: v1alpha1.VsphereCloudConfigStatus{},
			}
			Expect(k8sClient.Create(ctx, &cloudConfig)).Should(Succeed())
			config := v1alpha1.VsphereCloudConfig{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: "test-resource1"}, &config)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(config.Spec.VcIP).Should(Equal("1.1.1.1"))
		})

		It("should call reconcile function", func() {
			ctx := context.Background()

			cloudConfig := &v1alpha1.VsphereCloudConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource1",
					Namespace: "default",
				},
				Spec: v1alpha1.VsphereCloudConfigSpec{
					VcIP:        vc_ip,
					Insecure:    true,
					Credentials: "secret-ref",
					DataCenters: []string{},
				},
				Status: v1alpha1.VsphereCloudConfigStatus{},
			}

			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-ref",
					Namespace: "kube-system",
				},
				Data: map[string][]byte{
					"username": []byte(vc_user),
					"password": []byte(vc_pwd),
				},
			}

			objs := []client.Object{cloudConfig}
			s := scheme.Scheme
			s.AddKnownTypes(v1alpha1.GroupVersion, cloudConfig)
			client := fake.NewClientBuilder().WithObjects(objs...).Build()
			r := &VsphereCloudConfigReconciler{
				Client: client,
				Scheme: s,
				Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			}

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-resource1",
					Namespace: "default",
				},
			}

			// When the secret is unknown
			_, error := r.Reconcile(ctx, req)
			Expect(error).To(HaveOccurred())
			tempConfig := &vdov1alpha1.VsphereCloudConfig{}
			r.Get(ctx, req.NamespacedName, tempConfig)
			// Let the URL parsing Fail
			tempConfig.Spec.VcIP = "DEL"
			tempConfig.Spec.Thumbprint = ""
			Expect(client.Create(ctx, secret)).Should(Succeed())
			_, error = r.reconcileVCCredentials(ctx, tempConfig)
			Expect(error).To(HaveOccurred())

			// When all configs are available
			//Expect(client.Create(ctx, secret)).Should(Succeed())
			_, error = r.Reconcile(ctx, req)
			Expect(error).ToNot(HaveOccurred())

			// When the namespace is unknown
			req.NamespacedName.Namespace = "nonexistent"
			_, error = r.Reconcile(ctx, req)
			Expect(error).To(HaveOccurred())

			config := v1alpha1.VsphereCloudConfig{}
			Eventually(func() v1alpha1.ConfigStatus {
				err := client.Get(ctx, types.NamespacedName{Namespace: "default", Name: "test-resource1"}, &config)
				if err != nil {
					return ""
				}
				if len(config.Status.Config) <= 0 {
					return ""
				}
				return config.Status.Config
			}, timeout, interval).Should(BeEquivalentTo(v1alpha1.VsphereConfigVerified))
		})
	})

})

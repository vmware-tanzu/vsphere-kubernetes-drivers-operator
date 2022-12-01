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
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	vdov1alpha1 "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	dynclient "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/client"
	vdocontext "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/context"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/drivers/cpi"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/models"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/session"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/simulator"
	v1 "k8s.io/api/apps/v1"

	v12 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"net/http"
	"net/http/httptest"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("TestReconcileCSIDeploymentStatus", func() {

	Context("When CSI Deployment succeeds", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		BeforeEach(func() {
			daemonSet := &v1.DaemonSet{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vsphere-csi-node",
					Namespace: "kube-system",
					Labels:    map[string]string{"app": "test-label"},
				},
				Spec: v1.DaemonSetSpec{},
				Status: v1.DaemonSetStatus{
					NumberUnavailable: 0,
				},
			}

			pod1 := &v12.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "kube-system",
					Labels:    map[string]string{"app": "test-label"},
				},
				Spec: v12.PodSpec{
					Containers: []v12.Container{
						{
							Name:            "nginx",
							Image:           "nginx",
							ImagePullPolicy: "Always",
						},
					},
				},
				Status: v12.PodStatus{Phase: v12.PodRunning},
			}

			node := &v12.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				Spec:       v12.NodeSpec{ProviderID: "vsphere://testid"},
			}

			csiNode := &storagev1.CSINode{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: node.APIVersion,
							Kind:       node.Kind,
							Name:       node.Name,
							UID:        node.UID,
						},
					},
				},
				Spec: storagev1.CSINodeSpec{
					Drivers: []storagev1.CSINodeDriver{},
				},
			}
			podInfo := true

			csiDriver := &storagev1.CSIDriver{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "csi-driver-test-1",
				},
				Spec: storagev1.CSIDriverSpec{PodInfoOnMount: &podInfo},
			}

			Expect(r.Create(ctx, daemonSet, &client.CreateOptions{})).NotTo(HaveOccurred())

			_, err := clientSet.CoreV1().Pods("kube-system").Create(ctx, pod1, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = clientSet.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = clientSet.StorageV1().CSINodes().Create(ctx, csiNode, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = clientSet.StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			err = r.createCSINamespace(vdoctx)
			Expect(err).NotTo(HaveOccurred())

			err = r.createCSINamespace(vdoctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reconcile deployment status without error", func() {
			Expect(r.reconcileCSIDeploymentStatus(vdoctx, clientSet)).NotTo(HaveOccurred())

			// Verify verifyCSINodeStatus all scenarios
			node := &v12.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node-2"},
				Spec:       v12.NodeSpec{ProviderID: "vsphere://testid2"},
			}
			_, err := clientSet.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			err = r.verifyCSINodeStatus(vdoctx, clientSet)
			Expect(err).To(HaveOccurred())
		})

	})

	Context("When deployment of CSI resource fails", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		BeforeEach(func() {
			daemonSet := &v1.DaemonSet{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vsphere-csi-node",
					Namespace: "kube-system",
					Labels:    map[string]string{"app": "test-label"},
				},
				Spec: v1.DaemonSetSpec{},
				Status: v1.DaemonSetStatus{
					NumberUnavailable: 1,
				},
			}

			pod1 := &v12.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "kube-system",
					Labels:    map[string]string{"app": "test-label"},
				},
				Spec: v12.PodSpec{
					Containers: []v12.Container{
						{
							Name:            "nginx",
							Image:           "nginx",
							ImagePullPolicy: "Always",
						},
					},
				},
				Status: v12.PodStatus{Phase: v12.PodFailed},
			}

			node := &v12.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				Spec:       v12.NodeSpec{ProviderID: "vsphere://testid"},
			}

			csiNode := &storagev1.CSINode{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: node.APIVersion,
							Kind:       node.Kind,
							Name:       node.Name,
							UID:        node.UID,
						},
					},
				},
				Spec: storagev1.CSINodeSpec{
					Drivers: []storagev1.CSINodeDriver{},
				},
			}
			podInfo := true

			csiDriver := &storagev1.CSIDriver{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "csi-driver-test-1",
				},
				Spec: storagev1.CSIDriverSpec{PodInfoOnMount: &podInfo},
			}

			Expect(r.Create(ctx, daemonSet, &client.CreateOptions{})).NotTo(HaveOccurred())

			_, err := clientSet.CoreV1().Pods("kube-system").Create(ctx, pod1, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = clientSet.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = clientSet.StorageV1().CSINodes().Create(ctx, csiNode, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = clientSet.StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reconcile deployment status with error", func() {

			Expect(r.reconcileCSIDeploymentStatus(vdoctx, clientSet)).To(HaveOccurred())
		})

	})
})

var _ = Describe("TestCPIReconcile", func() {
	Context("reconcileCPIConfiguration success..", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("reconcileCPIConfigurationTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		vdoConfig := initializeVDOConfig("kube-system")
		req := *new(ctrl.Request)
		req.Namespace = "kube-system"

		It("when cloudconfig is empty", func() {
			vdoConfigTemp := *vdoConfig
			vdoConfigTemp.Spec.CloudProvider.VsphereCloudConfigs = []string{}

			_, errcpi := r.reconcileCPIConfiguration(vdoctx, req, &vdoConfigTemp, clientSet)
			Expect(errcpi).NotTo(HaveOccurred())
		})

		It("should reconcile CPI configuration, configStatus Failed", func() {
			cloudConfigStatus := v1alpha1.VsphereCloudConfigStatus{}
			cloudConfigStatus.Config = "failed"
			cloudConfig := &v1alpha1.VsphereCloudConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: "kube-system",
				},
				Spec: v1alpha1.VsphereCloudConfigSpec{
					VcIP:        "1.1.1.1",
					Insecure:    true,
					Credentials: "secret-ref",
					DataCenters: []string{"datacenter-1"},
				},
				Status: cloudConfigStatus,
			}
			Expect(r.Create(ctx, vdoConfig)).Should(Succeed())
			Expect(r.Create(ctx, cloudConfig)).Should(Succeed())

			_, errcpi := r.reconcileCPIConfiguration(vdoctx, req, vdoConfig, clientSet)
			Expect(errcpi).To(HaveOccurred())
			Expect(r.Delete(ctx, cloudConfig)).Should(Succeed())
		})

		It("should reconcile CPI configuration, configStatus Not Verified", func() {
			cloudConfigStatus := v1alpha1.VsphereCloudConfigStatus{}
			cloudConfigStatus.Config = "unknown"
			cloudConfig := &v1alpha1.VsphereCloudConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: "kube-system",
				},
				Spec: v1alpha1.VsphereCloudConfigSpec{
					VcIP:        "1.1.1.1",
					Insecure:    true,
					Credentials: "secret-ref",
					DataCenters: []string{"datacenter-1"},
				},
				Status: cloudConfigStatus,
			}

			Expect(r.Create(ctx, cloudConfig)).Should(Succeed())
			_, errcpi := r.reconcileCPIConfiguration(vdoctx, req, vdoConfig, clientSet)
			Expect(errcpi).To(HaveOccurred())
			Expect(r.Delete(ctx, cloudConfig)).Should(Succeed())
		})

		It("should reconcile CPI configuration, configStatus Verified", func() {
			vc_user := "test_user"
			vc_pwd := "test_user_pwd"
			cloudConfigStatus := v1alpha1.VsphereCloudConfigStatus{}
			cloudConfigStatus.Config = "verified"
			cloudConfig := &v1alpha1.VsphereCloudConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: "kube-system",
				},
				Spec: v1alpha1.VsphereCloudConfigSpec{
					VcIP:        "1.1.1.1",
					Insecure:    true,
					Credentials: "secret-ref",
					DataCenters: []string{"datacenter-1"},
				},
				Status: cloudConfigStatus,
			}

			secret := &v12.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-ref",
					Namespace: "kube-system",
				},
				Data: map[string][]byte{
					"username": []byte(vc_user),
					"password": []byte(vc_pwd),
				},
			}

			daemonSet := &v1.DaemonSet{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vsphere-cloud-controller-manager",
					Namespace: "kube-system",
					Labels:    map[string]string{"app": "k8s-app"},
				},
				Spec: v1.DaemonSetSpec{},
				Status: v1.DaemonSetStatus{
					NumberUnavailable: 0,
				},
			}

			Expect(r.Create(ctx, daemonSet)).Should(Succeed())
			Expect(r.Create(ctx, secret)).Should(Succeed())
			Expect(r.Create(ctx, cloudConfig)).Should(Succeed())
			_, errcpi := r.reconcileCPIConfiguration(vdoctx, req, vdoConfig, clientSet)
			Expect(errcpi).NotTo(HaveOccurred())



			// update reconcileCPISecret
			secretCPI := &v12.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cpi-global-secret",
					Namespace: "kube-system",
				},
				Data: map[string][]byte{
					"username": []byte("test-user-2"),
					"password": []byte(vc_pwd),
				},
			}
			cpiSecretKey := types.NamespacedName{
				Namespace: VC_CREDS_SECRET_NS,
				Name:      SECRET_NAME,
			}
			Expect(r.Update(ctx, secretCPI)).Should(Succeed())
			vsphereCloudConfigItems, err := r.fetchVsphereCloudConfigItems(vdoctx, req, vdoConfig, vdoConfig.Spec.CloudProvider.VsphereCloudConfigs)
			Expect(err).NotTo(HaveOccurred())

			//Provide unknown cloudconfig
			req.NamespacedName.Namespace = "unknown"
			vsphereCloudConfigItems, err = r.fetchVsphereCloudConfigItems(vdoctx, req, vdoConfig, []string{"un-known"})
			Expect(err).To(HaveOccurred())

			_, err = r.reconcileCPISecret(vdoctx, vdoConfig, &vsphereCloudConfigItems, cpiSecretKey)
			Expect(err).NotTo(HaveOccurred())

			// updateVdoConfigWithNodeStatus failure
			nodeStatus := make(map[string]vdov1alpha1.NodeStatus)
			vdoConfig.Status.CPIStatus.Phase = "pending"
			nodeStatus["trial-fail"] = vdov1alpha1.NodeStatusPending
			vdoConfig.Status.CPIStatus.NodeStatus = nodeStatus
			err = r.updateVdoConfigWithNodeStatus(vdoctx, vdoConfig, vdoConfig.Status.CPIStatus.Phase, nodeStatus)
			Expect(err).NotTo(HaveOccurred())

			// Error CloudConfig
			cloudConfig2 := &v1alpha1.VsphereCloudConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource-2",
					Namespace: "kube-system",
				},
				Spec: v1alpha1.VsphereCloudConfigSpec{
					VcIP:        "1.1.1.1",
					Insecure:    true,
					Credentials: "secret-ref-2",
					DataCenters: []string{"datacenter-1"},
				},
				Status: cloudConfigStatus,
			}
			Expect(r.Create(ctx, cloudConfig2)).Should(Succeed())
			_, err = r.getVcSession(vdoctx, cloudConfig2)
			Expect(err).To(HaveOccurred())

			vdoConfig.Spec.CloudProvider.VsphereCloudConfigs = []string{"test-resource-2"}
			_, err = r.reconcileCPIConfiguration(vdoctx, req, vdoConfig, clientSet)
			Expect(err).To(HaveOccurred())

		})
	})
})

var _ = Describe("TestCSIConfigurationReconcile", func() {
	Context("reconcileCSIConfiguration all sceanrios", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("reconcileCPIConfigurationTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		vdoConfig := initializeVDOConfig("kube-system")
		req := *new(ctrl.Request)
		req.Namespace = "kube-system"

		It("should reconcileCSIConfiguration, configStatus Failed", func() {
			_, errcpi := r.reconcileCSIConfiguration(vdoctx, req, vdoConfig, clientSet)
			Expect(errcpi).To(HaveOccurred())
		})

		It("should reconcileCSIConfiguration, configStatus Found, Not Configured", func() {
			vc_user := "test_user"
			vc_pwd := "test_user_pwd"
			cloudConfigStatus := v1alpha1.VsphereCloudConfigStatus{}
			cloudConfigStatus.Config = "verified"
			cloudConfig := &v1alpha1.VsphereCloudConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: "kube-system",
				},
				Spec: v1alpha1.VsphereCloudConfigSpec{
					VcIP:        "1.1.1.1",
					Insecure:    true,
					Credentials: "secret-ref",
					DataCenters: []string{"datacenter-1"},
				},
				Status: cloudConfigStatus,
			}

			secret := &v12.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-ref",
					Namespace: "kube-system",
				},
				Data: map[string][]byte{
					"username": []byte(vc_user),
					"password": []byte(vc_pwd),
				},
			}

			daemonSet := &v1.DaemonSet{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vsphere-cloud-controller-manager",
					Namespace: "kube-system",
					Labels:    map[string]string{"app": "k8s-app"},
				},
				Spec: v1.DaemonSetSpec{},
				Status: v1.DaemonSetStatus{
					NumberUnavailable: 0,
				},
			}

			Expect(r.Create(ctx, daemonSet)).Should(Succeed())
			Expect(r.Create(ctx, secret)).Should(Succeed())
			Expect(r.Create(ctx, cloudConfig)).Should(Succeed())
			_, errcpi := r.reconcileCSIConfiguration(vdoctx, req, vdoConfig, clientSet)
			Expect(errcpi).To(HaveOccurred())
			Expect(r.Delete(ctx, daemonSet)).Should(Succeed())
			Expect(r.Delete(ctx, secret)).Should(Succeed())
			Expect(r.Delete(ctx, cloudConfig)).Should(Succeed())
		})

		It("should reconcileCSIConfiguration, configStatus Found, Configured", func() {
			vc_user := "test_user"
			vc_pwd := "test_user_pwd"
			cloudConfigStatus := v1alpha1.VsphereCloudConfigStatus{}
			cloudConfigStatus.Config = "verified"
			cloudConfig := &v1alpha1.VsphereCloudConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: "kube-system",
				},
				Spec: v1alpha1.VsphereCloudConfigSpec{
					VcIP:        "1.1.1.1",
					Insecure:    true,
					Credentials: "secret-ref",
					DataCenters: []string{"datacenter-1"},
				},
				Status: cloudConfigStatus,
			}

			secret := &v12.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-ref",
					Namespace: "kube-system",
				},
				Data: map[string][]byte{
					"username": []byte(vc_user),
					"password": []byte(vc_pwd),
				},
			}

			daemonSet := &v1.DaemonSet{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vsphere-cloud-controller-manager",
					Namespace: "kube-system",
					Labels:    map[string]string{"app": "k8s-app"},
				},
				Spec: v1.DaemonSetSpec{},
				Status: v1.DaemonSetStatus{
					NumberUnavailable: 0,
				},
			}

			configMap := &v12.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "internal-feature-states.csi.vsphere.vmware.com",
					Namespace: "kube-system",
				},
				Immutable: nil,
				Data: map[string]string{
					"use-csinode-id": "true",
				},
				BinaryData: nil,
			}

			csiDaemonSet := &v1.DaemonSet{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vsphere-csi-node",
					Namespace: "kube-system",
					Labels:    map[string]string{"app": "app"},
				},
				Spec: v1.DaemonSetSpec{},
				Status: v1.DaemonSetStatus{
					NumberUnavailable: 0,
				},
			}

			node := &v12.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				Spec:       v12.NodeSpec{ProviderID: "vsphere://testid"},
			}

			csiNode := &storagev1.CSINode{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: node.APIVersion,
							Kind:       node.Kind,
							Name:       node.Name,
							UID:        node.UID,
						},
					},
				},
				Spec: storagev1.CSINodeSpec{
					Drivers: []storagev1.CSINodeDriver{},
				},
			}
			r.CurrentCSIDeployedVersion = "2.6.0"

			Expect(r.Create(vdoctx, configMap)).Should(Succeed())
			Expect(r.Create(ctx, daemonSet)).Should(Succeed())
			Expect(r.Create(ctx, csiDaemonSet)).Should(Succeed())
			Expect(r.Create(ctx, secret)).Should(Succeed())
			Expect(r.Create(ctx, cloudConfig)).Should(Succeed())
			_, err := clientSet.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = clientSet.StorageV1().CSINodes().Create(ctx, csiNode, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			_, errcpi := r.reconcileCSIConfiguration(vdoctx, req, vdoConfig, clientSet)
			Expect(errcpi).To(HaveOccurred())
			_, errcpi = r.reconcileCSIConfiguration(vdoctx, req, vdoConfig, clientSet)
			Expect(errcpi).To(HaveOccurred())
			Expect(r.Delete(ctx, daemonSet)).Should(Succeed())
			Expect(r.Delete(ctx, secret)).Should(Succeed())
			Expect(r.Delete(ctx, cloudConfig)).Should(Succeed())
		})

	})
})

var _ = Describe("TestDeleteReconcile", func() {
	Context("When Upgrade scenario hits", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		r.CsiDeploymentYamls = append(r.CsiDeploymentYamls, "https://raw.githubusercontent.com/asifdxtreme/Docs/master/compat/test-file-vdo-test.yaml")
		r.CpiDeploymentYamls = append(r.CpiDeploymentYamls, "https://raw.githubusercontent.com/asifdxtreme/Docs/master/compat/test-file-vdo-test.yaml")

		_, err := r.reconcileCPIDeployment(vdoctx)
		Expect(err).NotTo(HaveOccurred())

		_, err = r.reconcileCSIDeployment(vdoctx)
		Expect(err).NotTo(HaveOccurred())

		r.CpiDeploymentYamls = append(r.CpiDeploymentYamls, "")
		r.CsiDeploymentYamls = append(r.CsiDeploymentYamls, "")
		_, err = r.reconcileCPIDeployment(vdoctx)
		Expect(err).To(HaveOccurred())

		_, err = r.reconcileCSIDeployment(vdoctx)
		Expect(err).To(HaveOccurred())

		_, err = r.applyYaml(r.CsiDeploymentYamls[0], vdoctx, false, dynclient.CREATE)
		Expect(err).NotTo(HaveOccurred())

		_, err = r.deleteCSIDeployment(vdoctx)
		Expect(err).NotTo(HaveOccurred())

		_, err = r.applyYaml(r.CpiDeploymentYamls[0], vdoctx, false, dynclient.CREATE)
		Expect(err).NotTo(HaveOccurred())

		_, err = r.deleteCPIDeployment(vdoctx)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("TestReconcileCSISecret", func() {

	Context("When Secret Creation succeeds", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		vc_user := "test_user"
		vc_pwd := "test_user_pwd"

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		cloudConfig := v1alpha1.VsphereCloudConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-resource",
				Namespace: "kube-system",
			},
			Spec: v1alpha1.VsphereCloudConfigSpec{
				VcIP:        "1.1.1.1",
				Insecure:    true,
				Credentials: "secret-ref",
				DataCenters: []string{"datacenter-1"},
			},
			Status: v1alpha1.VsphereCloudConfigStatus{},
		}

		vdoConfig := initializeVDOConfig("default")

		BeforeEach(func() {
			secret := &v12.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-ref",
					Namespace: "kube-system",
				},
				Data: map[string][]byte{
					"username": []byte(vc_user),
					"password": []byte(vc_pwd),
				},
			}

			Expect(r.Create(ctx, secret)).Should(Succeed())

			Expect(r.Create(ctx, vdoConfig)).Should(Succeed())
		})

		It("should reconcile CSI secret without error", func() {
			_, err := r.reconcileCSISecret(vdoctx, vdoConfig, &cloudConfig)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("TestReconcileNodeProviderID", func() {

	Context("When ProviderID is present on all the nodes", func() {
		RegisterFailHandler(Fail)

		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		cloudconfiglist := initializeVsphereConfigList()

		vdoConfig := initializeVDOConfig("default")
		Expect(r.Create(vdoctx, vdoConfig)).Should(Succeed())

		node1 := v12.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "test-node1"},
			Spec:       v12.NodeSpec{ProviderID: "vsphere://testid1"},
		}

		node2 := v12.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "test-node2"},
			Spec:       v12.NodeSpec{ProviderID: "vsphere://testid2"},
		}

		_, err := clientSet.CoreV1().Nodes().Create(vdoctx, &node1, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		_, err = clientSet.CoreV1().Nodes().Create(vdoctx, &node2, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		It("should reconcile providerID without error", func() {
			_, err := r.reconcileNodeProviderID(vdoctx, vdoConfig, clientSet, &cloudconfiglist)
			Expect(err).NotTo(HaveOccurred())

			nodes, err := clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())

			for _, node := range nodes.Items {
				val, ok := vdoConfig.Status.CPIStatus.NodeStatus[node.Name]
				Expect(ok).To(BeTrue())
				Expect(val).Should(BeEquivalentTo(v1alpha1.NodeStatusReady))
			}

		})

	})

	Context("When both ProviderID and taint are absent", func() {
		RegisterFailHandler(Fail)

		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		secret := v12.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-ref",
				Namespace: "kube-system",
			},
			Data: map[string][]byte{
				"username": []byte("vc_user"),
				"password": []byte("vc_pwd"),
			},
		}

		node4 := v12.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "test-node4"},
		}

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		cloudconfiglist := initializeVsphereConfigList()
		vdoConfig := initializeVDOConfig("default")

		SessionFn = func(ctx context.Context,
			server string, datacenters []string, username, password, thumbprint string) (*session.Session, error) {
			return &session.Session{}, nil
		}

		GetVMFn = func(ctx context.Context, ipAddy string, datacenters []*object.Datacenter) (*session.VirtualMachine, error) {
			return &session.VirtualMachine{}, nil
		}

		Expect(r.Client.Create(vdoctx, &secret)).Should(Succeed())

		_, err := clientSet.CoreV1().Nodes().Create(vdoctx, &node4, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		It("should reconcile providerID without error", func() {
			_, err := r.reconcileNodeProviderID(vdoctx, vdoConfig, clientSet, &cloudconfiglist)
			Expect(err).NotTo(HaveOccurred())
			_, ok := vdoConfig.Status.CPIStatus.NodeStatus[node4.Name]
			Expect(ok).To(BeFalse())

		})

	})
	Context("When ProviderID is absent while taint is present and the node's DC/VC is mentioned in the vsphereCloudConfig resource", func() {
		RegisterFailHandler(Fail)

		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		taint := v12.Taint{
			Key:       CLOUD_PROVIDER_INIT_TAINT_KEY,
			Value:     "true",
			Effect:    TAINT_NOSCHEDULE_KEY,
			TimeAdded: nil,
		}
		node5 := v12.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "test-node5"},
			Spec:       v12.NodeSpec{Taints: []v12.Taint{taint}},
		}

		secret := v12.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-ref",
				Namespace: "kube-system",
			},
			Data: map[string][]byte{
				"username": []byte("vc_user"),
				"password": []byte("vc_pwd"),
			},
		}

		Expect(r.Client.Create(vdoctx, &secret)).Should(Succeed())

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		cloudConfig := v1alpha1.VsphereCloudConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-resource",
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
		cloudconfiglist := []v1alpha1.VsphereCloudConfig{cloudConfig}

		vdoConfig := initializeVDOConfig("default")

		SessionFn = func(ctx context.Context,
			server string, datacenters []string, username, password, thumbprint string) (*session.Session, error) {
			return &session.Session{}, nil
		}

		_, err := clientSet.CoreV1().Nodes().Create(vdoctx, &node5, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		It("should reconcile providerID without error", func() {
			GetVMFn = func(ctx context.Context, ipAddy string, datacenters []*object.Datacenter) (*session.VirtualMachine, error) {
				return &session.VirtualMachine{}, nil
			}
			_, err := r.reconcileNodeProviderID(vdoctx, vdoConfig, clientSet, &cloudconfiglist)
			Expect(err).NotTo(HaveOccurred())
			val, ok := vdoConfig.Status.CPIStatus.NodeStatus[node5.Name]
			Expect(ok).To(BeTrue())
			Expect(val).Should(BeEquivalentTo(v1alpha1.NodeStatusPending))
		})

	})

	Context("When taint is present but the node is not mentioned in the vsphereCloudConfigResource", func() {
		RegisterFailHandler(Fail)

		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		taint := v12.Taint{
			Key:       CLOUD_PROVIDER_INIT_TAINT_KEY,
			Value:     "true",
			Effect:    TAINT_NOSCHEDULE_KEY,
			TimeAdded: nil,
		}

		node6 := v12.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "test-node6"},
			Spec:       v12.NodeSpec{Taints: []v12.Taint{taint}},
		}

		secret := v12.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-ref",
				Namespace: "kube-system",
			},
			Data: map[string][]byte{
				"username": []byte("vc_user"),
				"password": []byte("vc_pwd"),
			},
		}

		Expect(r.Client.Create(vdoctx, &secret)).Should(Succeed())
		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		cloudconfiglist := initializeVsphereConfigList()

		vdoConfig := initializeVDOConfig("default")

		SessionFn = func(ctx context.Context,
			server string, datacenters []string, username, password, thumbprint string) (*session.Session, error) {
			return &session.Session{}, nil
		}

		_, err := clientSet.CoreV1().Nodes().Create(vdoctx, &node6, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		It("should reconcile providerID with error", func() {
			GetVMFn = func(ctx context.Context, ipAddy string, datacenters []*object.Datacenter) (*session.VirtualMachine, error) {
				return nil, nil
			}
			_, err := r.reconcileNodeProviderID(vdoctx, vdoConfig, clientSet, &cloudconfiglist)
			Expect(err).To(HaveOccurred())
		})

	})

})

var _ = Describe("TestCompareVersions", func() {
	Context("Compare version with boundaries", func() {
		RegisterFailHandler(Fail)

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		result, err := r.compareVersions("1.2.0", "1.3.0", "1.4.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())

		result, err = r.compareVersions("1.2.0", "1.5.0", "1.4.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())

		result, err = r.compareVersions("1.2.0", "1.2.0", "1.2.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())

		result, err = r.compareVersions("1.2.0", "1.2.1", "1.2.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())

		result, err = r.compareVersions("1.2.0", "1.2+", "1.2.2")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())

		result, err = r.compareSkewVersions("1.2.0", "1.2.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())

		result, err = r.compareSkewVersions("1.2.0", "1.1.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())

		result, err = r.compareSkewVersions("1.2+", "1.2.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())

	})
})

var _ = Describe("TestReconcileNodeLabel", func() {
	node1 := v12.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node1"},
		Spec:       v12.NodeSpec{ProviderID: "vsphere://testid1"},
	}

	node2 := v12.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node2"},
	}

	Context("When reconciling node label succeeds", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-resource",
				Namespace: "default",
			},
		}

		vdoConfig := initializeVDOConfig("default")
		Expect(r.Create(vdoctx, vdoConfig)).Should(Succeed())

		vdoConfig.Status.CPIStatus.NodeStatus = map[string]v1alpha1.NodeStatus{node1.Name: v1alpha1.NodeStatusReady}

		_, err := clientSet.CoreV1().Nodes().Create(vdoctx, &node1, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		_, err = clientSet.CoreV1().Nodes().Create(vdoctx, &node2, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		It("should reconcile node label without error", func() {
			err := r.reconcileNodeLabel(vdoctx, req, clientSet, vdoConfig)
			Expect(err).NotTo(HaveOccurred())

			nodes, err := clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())

			for _, node := range nodes.Items {
				if node.Name == "test-node1" {
					Expect(node.Labels[VDO_NODE_LABEL_KEY]).Should(BeEquivalentTo(req.Name))
				}
			}
		})

		It("should not add label to the node", func() {

			nodes, err := clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())

			for _, node := range nodes.Items {
				if node.Name == "test-node2" {
					_, ok := node.Labels[VDO_NODE_LABEL_KEY]
					Expect(ok).To(BeFalse())
				}
			}

		})

	})
})

var _ = Describe("TestfetchDeploymentYamls", func() {

	Context("When fetching Deployment yamls succeeds", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		var matrix models.CompatMatrix

		matrixString := "{\n    \"CSI\" : {\n            \"2.2.1\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"min\": \"1.18\", \"max\": \"1.21\"},\n                    \"isCPIRequired\" : false,\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"]\n                    }\n        },\n    \"CPI\" : {\n            \"1.20.0\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"skewVersion\": \"1.21\"},\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"]\n                    }\n        }\n             \n}"
		err := json.Unmarshal([]byte(matrixString), &matrix)
		Expect(err).NotTo(HaveOccurred())

		It("should fetch CSI deployment yamls without error", func() {
			err = r.FetchCsiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.21")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fetch CPI deployment yamls without error", func() {
			err = r.FetchCpiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.21")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When version is mentioned in incorrect format", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		var matrix models.CompatMatrix

		matrixString := "{\n    \"CSI\" : {\n            \"2.2.1\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"min\": \"1.18\", \"max\": \"1.21.X\"},\n                    \"isCPIRequired\" : false,\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"]\n                    }\n        },\n    \"CPI\" : {\n            \"1.20.0\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"skewVersion\": \"1.21.X\"},\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"]\n                    }\n        }\n             \n}"
		err := json.Unmarshal([]byte(matrixString), &matrix)
		Expect(err).NotTo(HaveOccurred())

		It("should fetch CSI deployment yamls with error", func() {
			err = r.FetchCsiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.21")
			Expect(err).To(HaveOccurred())
		})

		It("should fetch CPI deployment yamls with error", func() {
			err = r.FetchCpiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.21")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When none of the versions matches", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		var matrix models.CompatMatrix

		matrixString := "{\n    \"CSI\" : {\n            \"2.2.1\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"min\": \"1.18\", \"max\": \"1.21\"},\n                    \"isCPIRequired\" : false,\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"]\n                    }\n        },\n    \"CPI\" : {\n            \"1.20.0\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"skewVersion\": \"1.21\"},\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"]\n                    }\n        }\n             \n}"
		err := json.Unmarshal([]byte(matrixString), &matrix)
		Expect(err).NotTo(HaveOccurred())

		It("should fetch CSI deployment yamls with error", func() {
			err = r.FetchCsiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.22")
			Expect(err).To(HaveOccurred())
		})

		It("should fetch CPI deployment yamls with error", func() {
			err = r.FetchCpiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.22")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When second latest version matches", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		var matrix models.CompatMatrix

		matrixString := "{\n\t\"CSI\": {\n\t\t\"2.2.1\": {\n\t\t\t\"vSphere\": {\n\t\t\t\t\"min\": \"6.7.0\",\n\t\t\t\t\"max\": \"7.0.7\"\n\t\t\t},\n\t\t\t\"k8s\": {\n\t\t\t\t\"min\": \"1.18\",\n\t\t\t\t\"max\": \"1.21\"\n\t\t\t},\n\t\t\t\"isCPIRequired\": false,\n\t\t\t\"deploymentPath\": [\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"\n\t\t\t]\n\t\t},\n\t\t\"2.2.0\": {\n\t\t\t\"vSphere\": {\n\t\t\t\t\"min\": \"6.5\",\n\t\t\t\t\"max\": \"6.7\"\n\t\t\t},\n\t\t\t\"k8s\": {\n\t\t\t\t\"min\": \"1.16\",\n\t\t\t\t\"max\": \"1.17\"\n\t\t\t},\n\t\t\t\"isCPIRequired\": false,\n\t\t\t\"deploymentPath\": [\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"\n\t\t\t]\n\t\t}\n\t},\n\t\"CPI\": {\n\t\t\"1.20.0\": {\n\t\t\t\"vSphere\": {\n\t\t\t\t\"min\": \"6.7\",\n\t\t\t\t\"max\": \"7.0\"\n\t\t\t},\n\t\t\t\"k8s\": {\n\t\t\t\t\"skewVersion\": \"1.21\"\n\t\t\t},\n\t\t\t\"deploymentPath\": [\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"\n\t\t\t]\n\t\t},\n\t\t\"1.19.3\": {\n\t\t\t\"vSphere\": {\n\t\t\t\t\"min\": \"6.5\",\n\t\t\t\t\"max\": \"6.7\"\n\t\t\t},\n\t\t\t\"k8s\": {\n\t\t\t\t\"skewVersion\": \"1.17\"\n\t\t\t},\n\t\t\t\"deploymentPath\": [\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n\t\t\t\t\"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"\n\t\t\t]\n\t\t}\n\t}\n\n}"
		err := json.Unmarshal([]byte(matrixString), &matrix)
		Expect(err).NotTo(HaveOccurred())

		It("should fetch CSI deployment yamls without error", func() {
			err = r.FetchCsiDeploymentYamls(vdoctx, matrix, []string{"6.5.3"}, "1.17")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fetch CPI deployment yamls without error", func() {
			err = r.FetchCpiDeploymentYamls(vdoctx, matrix, []string{"6.5.3"}, "1.17")
			Expect(err).NotTo(HaveOccurred())
		})

	})

})

var _ = Describe("TestApplyYaml", func() {

	Context("When yaml gets applied successfully", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		DEPLOYMENT_YAML_URL := "https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml"
		FILE_PATH := "/tmp/test_deployment.yaml"

		BeforeEach(func() {
			fileContents := "kind: Deployment\napiVersion: apps/v1\nmetadata:\n  name: vsphere-csi-controller\n  namespace: kube-system\nspec:\n  replicas: 1\n  selector:\n    matchLabels:\n      app: vsphere-csi-controller\n  template:\n    metadata:\n      labels:\n        app: vsphere-csi-controller\n        role: vsphere-csi\n    spec:\n      serviceAccountName: vsphere-csi-controller\n      nodeSelector:\n        node-role.kubernetes.io/master: \"\"\n      tolerations:\n        - key: node-role.kubernetes.io/master\n          operator: Exists\n          effect: NoSchedule\n        # uncomment below toleration if you need an aggressive pod eviction in case when\n        # node becomes not-ready or unreachable. Default is 300 seconds if not specified.\n        #- key: node.kubernetes.io/not-ready\n        #  operator: Exists\n        #  effect: NoExecute\n        #  tolerationSeconds: 30\n        #- key: node.kubernetes.io/unreachable\n        #  operator: Exists\n        #  effect: NoExecute\n        #  tolerationSeconds: 30\n      dnsPolicy: \"Default\"\n      containers:\n        - name: csi-attacher\n          image: quay.io/k8scsi/csi-attacher:v3.1.0\n          args:\n            - \"--v=4\"\n            - \"--timeout=300s\"\n            - \"--csi-address=$(ADDRESS)\"\n            - \"--leader-election\"\n          env:\n            - name: ADDRESS\n              value: /csi/csi.sock\n          csiVolumeMounts:\n            - mountPath: /csi\n              name: socket-dir\n        - name: csi-resizer\n          image: quay.io/k8scsi/csi-resizer:v1.1.0\n          args:\n            - \"--v=4\"\n            - \"--timeout=300s\"\n            - \"--handle-volume-inuse-error=false\"\n            - \"--csi-address=$(ADDRESS)\"\n            - \"--kube-api-qps=100\"\n            - \"--kube-api-burst=100\"\n            - \"--leader-election\"\n          env:\n            - name: ADDRESS\n              value: /csi/csi.sock\n          csiVolumeMounts:\n            - mountPath: /csi\n              name: socket-dir\n        - name: vsphere-csi-controller\n          image: gcr.io/cloud-provider-vsphere/csi/release/driver:v2.2.1\n          args:\n            - \"--fss-name=internal-feature-states.csi.vsphere.vmware.com\"\n            - \"--fss-namespace=$(CSI_NAMESPACE)\"\n          imagePullPolicy: \"Always\"\n          env:\n            - name: CSI_ENDPOINT\n              value: unix:///csi/csi.sock\n            - name: X_CSI_MODE\n              value: \"controller\"\n            - name: VSPHERE_CSI_CONFIG\n              value: \"/etc/cloud/csi-vsphere.conf\"\n            - name: LOGGER_LEVEL\n              value: \"PRODUCTION\" # Options: DEVELOPMENT, PRODUCTION\n            - name: INCLUSTER_CLIENT_QPS\n              value: \"100\"\n            - name: INCLUSTER_CLIENT_BURST\n              value: \"100\"\n            - name: CSI_NAMESPACE\n              valueFrom:\n                fieldRef:\n                  fieldPath: metadata.namespace\n            - name: X_CSI_SERIAL_VOL_ACCESS_TIMEOUT\n              value: 3m\n          csiVolumeMounts:\n            - mountPath: /etc/cloud\n              name: vsphere-config-volume\n              readOnly: true\n            - mountPath: /csi\n              name: socket-dir\n          ports:\n            - name: healthz\n              containerPort: 9808\n              protocol: TCP\n            - name: prometheus\n              containerPort: 2112\n              protocol: TCP\n          livenessProbe:\n            httpGet:\n              path: /healthz\n              port: healthz\n            initialDelaySeconds: 10\n            timeoutSeconds: 3\n            periodSeconds: 5\n            failureThreshold: 3\n        - name: liveness-probe\n          image: quay.io/k8scsi/livenessprobe:v2.2.0\n          args:\n            - \"--v=4\"\n            - \"--csi-address=/csi/csi.sock\"\n          csiVolumeMounts:\n            - name: socket-dir\n              mountPath: /csi\n        - name: vsphere-syncer\n          image: gcr.io/cloud-provider-vsphere/csi/release/syncer:v2.2.1\n          args:\n            - \"--leader-election\"\n            - \"--fss-name=internal-feature-states.csi.vsphere.vmware.com\"\n            - \"--fss-namespace=$(CSI_NAMESPACE)\"\n          imagePullPolicy: \"Always\"\n          ports:\n            - containerPort: 2113\n              name: prometheus\n              protocol: TCP\n          env:\n            - name: FULL_SYNC_INTERVAL_MINUTES\n              value: \"30\"\n            - name: VSPHERE_CSI_CONFIG\n              value: \"/etc/cloud/csi-vsphere.conf\"\n            - name: LOGGER_LEVEL\n              value: \"PRODUCTION\" # Options: DEVELOPMENT, PRODUCTION\n            - name: INCLUSTER_CLIENT_QPS\n              value: \"100\"\n            - name: INCLUSTER_CLIENT_BURST\n              value: \"100\"\n            - name: CSI_NAMESPACE\n              valueFrom:\n                fieldRef:\n                  fieldPath: metadata.namespace\n          csiVolumeMounts:\n            - mountPath: /etc/cloud\n              name: vsphere-config-volume\n              readOnly: true\n        - name: csi-provisioner\n          image: quay.io/k8scsi/csi-provisioner:v2.1.0\n          args:\n            - \"--v=4\"\n            - \"--timeout=300s\"\n            - \"--csi-address=$(ADDRESS)\"\n            - \"--kube-api-qps=100\"\n            - \"--kube-api-burst=100\"\n            - \"--leader-election\"\n            - \"--default-fstype=ext4\"\n            # needed only for topology aware setup\n            #- \"--feature-gates=Topology=true\"\n            #- \"--strict-topology\"\n          env:\n            - name: ADDRESS\n              value: /csi/csi.sock\n          csiVolumeMounts:\n            - mountPath: /csi\n              name: socket-dir\n      volumes:\n      - name: vsphere-config-volume\n        secret:\n          secretName: vsphere-config-secret\n      - name: socket-dir\n        emptyDir: {}\n---\napiVersion: v1\ndata:\n  \"csi-migration\": \"false\"\n  \"csi-auth-check\": \"true\"\n  \"online-volume-extend\": \"true\"\nkind: ConfigMap\nmetadata:\n  name: internal-feature-states.csi.vsphere.vmware.com\n  namespace: kube-system\n---\napiVersion: storage.k8s.io/v1 # For k8s 1.17 use storage.k8s.io/v1beta1\nkind: CSIDriver\nmetadata:\n  name: csi.vsphere.vmware.com\nspec:\n  attachRequired: true\n  podInfoOnMount: false\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: vsphere-csi-controller\n  namespace: kube-system\n  labels:\n    app: vsphere-csi-controller\nspec:\n  ports:\n    - name: ctlr\n      port: 2112\n      targetPort: 2112\n      protocol: TCP\n    - name: syncer\n      port: 2113\n      targetPort: 2113\n      protocol: TCP\n  selector:\n    app: vsphere-csi-controller"

			err := createConfigFile(FILE_PATH, fileContents)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should apply the File path yaml without error", func() {
			_, err := r.applyYaml("file:/"+FILE_PATH, vdoctx, false, dynclient.CREATE)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should apply the Network Path yaml without error", func() {
			_, err := r.applyYaml(DEPLOYMENT_YAML_URL, vdoctx, false, dynclient.CREATE)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When apply yaml throws error", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		DEPLOYMENT_YAML_URL := "https://raw.githubusercontent.com/kubernetes-sigs/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml"
		FILE_PATH := "/tmp/test_deployment.yaml"

		BeforeEach(func() {
			fileContents := "kind: Deployment\nmetadata:\n  name: vsphere-csi-controller\n  namespace: kube-system\nspec:\n  replicas: 1\n  selector:\n    matchLabels:\n      app: vsphere-csi-controller\n  template:\n    metadata:\n      labels:\n        app: vsphere-csi-controller\n        role: vsphere-csi\n    spec:\n      serviceAccountName: vsphere-csi-controller\n      nodeSelector:\n        node-role.kubernetes.io/master: \"\"\n      tolerations:\n        - key: node-role.kubernetes.io/master\n          operator: Exists\n          effect: NoSchedule\n        # uncomment below toleration if you need an aggressive pod eviction in case when\n        # node becomes not-ready or unreachable. Default is 300 seconds if not specified.\n        #- key: node.kubernetes.io/not-ready\n        #  operator: Exists\n        #  effect: NoExecute\n        #  tolerationSeconds: 30\n        #- key: node.kubernetes.io/unreachable\n        #  operator: Exists\n        #  effect: NoExecute\n        #  tolerationSeconds: 30\n      dnsPolicy: \"Default\"\n      containers:\n        - name: csi-attacher\n          image: quay.io/k8scsi/csi-attacher:v3.1.0\n          args:\n            - \"--v=4\"\n            - \"--timeout=300s\"\n            - \"--csi-address=$(ADDRESS)\"\n            - \"--leader-election\"\n          env:\n            - name: ADDRESS\n              value: /csi/csi.sock\n          csiVolumeMounts:\n            - mountPath: /csi\n              name: socket-dir\n        - name: csi-resizer\n          image: quay.io/k8scsi/csi-resizer:v1.1.0\n          args:\n            - \"--v=4\"\n            - \"--timeout=300s\"\n            - \"--handle-volume-inuse-error=false\"\n            - \"--csi-address=$(ADDRESS)\"\n            - \"--kube-api-qps=100\"\n            - \"--kube-api-burst=100\"\n            - \"--leader-election\"\n          env:\n            - name: ADDRESS\n              value: /csi/csi.sock\n          csiVolumeMounts:\n            - mountPath: /csi\n              name: socket-dir\n        - name: vsphere-csi-controller\n          image: gcr.io/cloud-provider-vsphere/csi/release/driver:v2.2.1\n          args:\n            - \"--fss-name=internal-feature-states.csi.vsphere.vmware.com\"\n            - \"--fss-namespace=$(CSI_NAMESPACE)\"\n          imagePullPolicy: \"Always\"\n          env:\n            - name: CSI_ENDPOINT\n              value: unix:///csi/csi.sock\n            - name: X_CSI_MODE\n              value: \"controller\"\n            - name: VSPHERE_CSI_CONFIG\n              value: \"/etc/cloud/csi-vsphere.conf\"\n            - name: LOGGER_LEVEL\n              value: \"PRODUCTION\" # Options: DEVELOPMENT, PRODUCTION\n            - name: INCLUSTER_CLIENT_QPS\n              value: \"100\"\n            - name: INCLUSTER_CLIENT_BURST\n              value: \"100\"\n            - name: CSI_NAMESPACE\n              valueFrom:\n                fieldRef:\n                  fieldPath: metadata.namespace\n            - name: X_CSI_SERIAL_VOL_ACCESS_TIMEOUT\n              value: 3m\n          csiVolumeMounts:\n            - mountPath: /etc/cloud\n              name: vsphere-config-volume\n              readOnly: true\n            - mountPath: /csi\n              name: socket-dir\n          ports:\n            - name: healthz\n              containerPort: 9808\n              protocol: TCP\n            - name: prometheus\n              containerPort: 2112\n              protocol: TCP\n          livenessProbe:\n            httpGet:\n              path: /healthz\n              port: healthz\n            initialDelaySeconds: 10\n            timeoutSeconds: 3\n            periodSeconds: 5\n            failureThreshold: 3\n        - name: liveness-probe\n          image: quay.io/k8scsi/livenessprobe:v2.2.0\n          args:\n            - \"--v=4\"\n            - \"--csi-address=/csi/csi.sock\"\n          csiVolumeMounts:\n            - name: socket-dir\n              mountPath: /csi\n        - name: vsphere-syncer\n          image: gcr.io/cloud-provider-vsphere/csi/release/syncer:v2.2.1\n          args:\n            - \"--leader-election\"\n            - \"--fss-name=internal-feature-states.csi.vsphere.vmware.com\"\n            - \"--fss-namespace=$(CSI_NAMESPACE)\"\n          imagePullPolicy: \"Always\"\n          ports:\n            - containerPort: 2113\n              name: prometheus\n              protocol: TCP\n          env:\n            - name: FULL_SYNC_INTERVAL_MINUTES\n              value: \"30\"\n            - name: VSPHERE_CSI_CONFIG\n              value: \"/etc/cloud/csi-vsphere.conf\"\n            - name: LOGGER_LEVEL\n              value: \"PRODUCTION\" # Options: DEVELOPMENT, PRODUCTION\n            - name: INCLUSTER_CLIENT_QPS\n              value: \"100\"\n            - name: INCLUSTER_CLIENT_BURST\n              value: \"100\"\n            - name: CSI_NAMESPACE\n              valueFrom:\n                fieldRef:\n                  fieldPath: metadata.namespace\n          csiVolumeMounts:\n            - mountPath: /etc/cloud\n              name: vsphere-config-volume\n              readOnly: true\n        - name: csi-provisioner\n          image: quay.io/k8scsi/csi-provisioner:v2.1.0\n          args:\n            - \"--v=4\"\n            - \"--timeout=300s\"\n            - \"--csi-address=$(ADDRESS)\"\n            - \"--kube-api-qps=100\"\n            - \"--kube-api-burst=100\"\n            - \"--leader-election\"\n            - \"--default-fstype=ext4\"\n            # needed only for topology aware setup\n            #- \"--feature-gates=Topology=true\"\n            #- \"--strict-topology\"\n          env:\n            - name: ADDRESS\n              value: /csi/csi.sock\n          csiVolumeMounts:\n            - mountPath: /csi\n              name: socket-dir\n      volumes:\n      - name: vsphere-config-volume\n        secret:\n          secretName: vsphere-config-secret\n      - name: socket-dir\n        emptyDir: {}\n---\napiVersion: v1\ndata:\n  \"csi-migration\": \"false\"\n  \"csi-auth-check\": \"true\"\n  \"online-volume-extend\": \"true\"\nkind: ConfigMap\nmetadata:\n  name: internal-feature-states.csi.vsphere.vmware.com\n  namespace: kube-system\n---\napiVersion: storage.k8s.io/v1 # For k8s 1.17 use storage.k8s.io/v1beta1\nkind: CSIDriver\nmetadata:\n  name: csi.vsphere.vmware.com\nspec:\n  attachRequired: true\n  podInfoOnMount: false\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: vsphere-csi-controller\n  namespace: kube-system\n  labels:\n    app: vsphere-csi-controller\nspec:\n  ports:\n    - name: ctlr\n      port: 2112\n      targetPort: 2112\n      protocol: TCP\n    - name: syncer\n      port: 2113\n      targetPort: 2113\n      protocol: TCP\n  selector:\n    app: vsphere-csi-controller"

			err := createConfigFile(FILE_PATH, fileContents)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should throw error while applying the Network path yaml", func() {
			_, err := r.applyYaml("file:/"+FILE_PATH, vdoctx, false, dynclient.CREATE)
			Expect(err).To(HaveOccurred())
		})

		It("should throw error while applying the File Path yaml", func() {
			_, err := r.applyYaml(DEPLOYMENT_YAML_URL, vdoctx, false, dynclient.CREATE)
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("TestParseMatrixYaml", func() {

	Context("When matrix yaml is parsed successfully successfully", func() {
		RegisterFailHandler(Fail)

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		MATRIX_YAML_URL := "https://raw.githubusercontent.com/ridaz/Sample-files/main/sample.yaml"
		FILE_PATH := "/tmp/test_matrix.yaml"

		BeforeEach(func() {
			fileContents := "{\n    \"CSI\" : {\n            \"2.2.1\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.1\", \"max\": \"7.0.4\"},\n                    \"k8s\" : {\"min\": \"1.18\", \"max\": \"1.22\"},\n                    \"isCPIRequired\" : false,\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"]\n                    }\n        },\n    \"CPI\" : {\n            \"1.20.0\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.1\", \"max\": \"7.0.4\"},\n                    \"k8s\" : {\"skewVersion\": \"1.22\"},\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"]\n                    }\n        }\n             \n}"

			err := createConfigFile(FILE_PATH, fileContents)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should parse the Network path yaml without error", func() {
			_, err := dynclient.ParseMatrixYaml(MATRIX_YAML_URL)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should parse the File Path yaml without error", func() {
			_, err := dynclient.ParseMatrixYaml("file:/" + FILE_PATH)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("TestGetMatrixConfig", func() {

	Context("When Compat matrix configmap contains expected data", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		configMap := &v12.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "compat-matrix-config",
				Namespace: "default",
			},
			Immutable: nil,
			Data: map[string]string{
				"auto-upgrade":         "disabled",
				"versionConfigContent": "{\n    \"CSI\" : {\n            \"2.2.1\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.1\", \"max\": \"7.0.4\"},\n                    \"k8s\" : {\"min\": \"1.18\", \"max\": \"1.22\"},\n                    \"isCPIRequired\" : false,\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"]\n                    }\n        },\n    \"CPI\" : {\n            \"1.20.0\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.1\", \"max\": \"7.0.4\"},\n                    \"k8s\" : {\"skewVersion\": \"1.22\"},\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"]\n                    }\n        }\n             \n}",
			},
			BinaryData: nil,
		}
		Expect(r.Create(vdoctx, configMap)).Should(Succeed())

		It("should fetch the matrix config without error", func() {
			_, err := r.getMatrixConfig(configMap.Data["versionConfigURL"], configMap.Data["versionConfigContent"])
			Expect(err).NotTo(HaveOccurred())
		})

	})

	Context("When Compat matrix configmap doesn't contain expected data", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		configMap := &v12.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "compat-matrix-config",
				Namespace: "default",
			},
			Immutable: nil,
			Data: map[string]string{
				"auto-upgrade":         "disabled",
				"versionConfigContent": "{\n    \"CSI\" : {\n            \"2.2.1\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.1\", \"max\": \"7.0.4\"},\n                    \"k8s\" : {\"min\": \"1.18\", \"max\": \"1.22\"},\n                    \"isCPIRequired\" : false,\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"]\n                    }\n        },\n    \"CPI\" : {\n            \"1.20.0\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.1\", \"max\": \"7.0.4\"},\n                    \"k8s\" : {\"skewVersion\": \"1.22\"},\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"]\n                    }\n        }\n             \n}",
				"versionConfigURL":     "https://raw.githubusercontent.com/asifdxtreme/Docs/master/sample/matrix/matrix.yaml",
			},
			BinaryData: nil,
		}
		Expect(r.Create(vdoctx, configMap)).Should(Succeed())

		It("should fetch the matrix config with error", func() {
			_, err := r.getMatrixConfig(configMap.Data["versionConfigURL"], configMap.Data["versionConfigContent"])
			Expect(err).To(HaveOccurred())
		})

	})
})

var _ = Describe("TestupdateMatrixInfo", func() {

	Context("When creating new Matrix", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()
		VDO_NAMESPACE = "vmware-system-vdo"

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		expect := version.Info{
			Major:     "1",
			Minor:     "21",
			GitCommit: "v1.21.1",
		}
		// get server object with expected version info
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			output, err := json.Marshal(expect)
			Expect(err).NotTo(HaveOccurred())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err = w.Write(output)
			Expect(err).NotTo(HaveOccurred())
		}))

		r := VDOConfigReconciler{
			Client:       fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger:       ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme:       s,
			ClientConfig: &restclient.Config{Host: server.URL},
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      CM_NAME,
				Namespace: VDO_NAMESPACE,
			},
		}

		SessionFn = func(ctx context.Context,
			server string, datacenters []string, username, password, thumbprint string) (*session.Session, error) {
			return &session.Session{
				Client:         nil,
				Datacenters:    nil,
				VsphereVersion: "7.0.3",
			}, nil

		}
		vdoConfig := initializeVDOConfig("default")
		Expect(r.Create(vdoctx, vdoConfig)).Should(Succeed())

		It("should create the resources without error", func() {

			secret := &v12.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-ref",
					Namespace: "kube-system",
				},
				Data: map[string][]byte{
					"username": []byte("test_user"),
					"password": []byte("test_password"),
				},
			}
			Expect(r.Create(ctx, secret)).Should(Succeed())

			configMapObject := &v12.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      CM_NAME,
					Namespace: VDO_NAMESPACE,
				},
				Data: map[string]string{
					"auto-upgrade":     "disabled",
					"versionConfigURL": "https://raw.githubusercontent.com/asifdxtreme/Docs/master/sample/matrix/matrix.yaml",
				},
			}

			Expect(r.Create(ctx, configMapObject)).Should(Succeed())
		})

		It("Should set the env variables", func() {
			err := r.updateMatrixInfo(vdoctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Getenv(COMPAT_MATRIX_CONFIG_URL)).Should(Equal("https://raw.githubusercontent.com/asifdxtreme/Docs/master/sample/matrix/matrix.yaml"))
			defer server.Close()
		})

		It("Should not give error if VDOConfig not available", func() {
			err := r.updateMatrixInfo(vdoctx, req)
			Expect(err).ToNot(HaveOccurred())
			defer server.Close()
		})

		It("Should unset env variables when Configmap is deleted", func() {
			configMapObject := &v12.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      CM_NAME,
					Namespace: VDO_NAMESPACE,
				},
				Data: map[string]string{
					"auto-upgrade":     "disabled",
					"versionConfigURL": "https://raw.githubusercontent.com/asifdxtreme/Docs/master/sample/matrix/matrix.yaml",
				},
			}
			Expect(r.Delete(ctx, configMapObject)).Should(Succeed())
			err := r.updateMatrixInfo(vdoctx, req)
			Expect(os.Getenv(COMPAT_MATRIX_CONFIG_URL)).Should(Equal(""))
			Expect(err).To(HaveOccurred())
			defer server.Close()
		})
	})
})

var _ = Describe("TestReconcile", func() {
	Context("when reconcile is queued", func() {

		os.Setenv("VDO_NAMESPACE", "vmware-system-vdo")
		ctx := context.Background()
		defer GinkgoRecover()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfigList{})
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VsphereCloudConfig{})

		It("should fail loading compatibility matrix", func() {
			r := VDOConfigReconciler{
				Client:       fake2.NewClientBuilder().WithRuntimeObjects().Build(),
				Logger:       ctrllog.Log.WithName("VDOConfigControllerTest"),
				Scheme:       s,
				ClientConfig: testRestConfig,
			}
			vdoctx := vdocontext.VDOContext{
				Context: ctx,
				Logger:  r.Logger,
			}

			secret := &v12.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-ref",
					Namespace: "kube-system",
				},
				Data: map[string][]byte{
					"username": []byte("vc_user"),
					"password": []byte("vc_pwd"),
				},
			}

			cloudConfig := &v1alpha1.VsphereCloudConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
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

			vdoConfig := initializeVDOConfigWithStatus("default")

			Expect(r.Create(vdoctx, secret)).Should(Succeed())
			Expect(r.Create(vdoctx, cloudConfig)).Should(Succeed())
			Expect(r.Create(vdoctx, vdoConfig)).Should(Succeed())

			ns := types.NamespacedName{Name: "vdo-sample",
				Namespace: "default"}
			req := ctrl.Request{NamespacedName: ns}
			_, err := r.Reconcile(ctx, req)
			Expect(err.Error()).To(BeEquivalentTo("Matrix Config URL/Content not provided in proper format"))

			vdoNamespace := os.Getenv("VDO_NAMESPACE")
			os.Unsetenv("VDO_NAMESPACE")
			_, err = r.Reconcile(ctx, req)
			Expect(err.Error()).To(BeEquivalentTo("Unable to determine operator namespace"))
			os.Setenv("VDO_NAMESPACE", vdoNamespace)

			ns2 := types.NamespacedName{Name: "vdo-sample:21",
				Namespace: "default"}
			req2 := ctrl.Request{NamespacedName: ns2}
			os.Setenv("MATRIX_CONFIG_URL", "https://raw.githubusercontent.com/asifdxtreme/Docs/master/sample/matrix/matrix.yaml")
			_, err = r.Reconcile(ctx, req2)
			Expect(err).NotTo(HaveOccurred())
			req.Name = "vdo-sample"
			_, err = r.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("TestReconcileConfigMap", func() {

	Context("When Configmap Creation succeeds", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		cloudconfiglist := initializeVsphereConfigList()

		vdoConfig := initializeVDOConfig("default")

		Expect(r.Create(ctx, vdoConfig)).Should(Succeed())

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		secretTestKey := types.NamespacedName{
			Name:      "test-secret",
			Namespace: "kube-system",
		}

		cpi.CPI_VSPHERE_CONF_FILE = "test_config.conf"
		It("should reconcile configmap without error", func() {
			_, err := r.reconcileConfigMap(vdoctx, vdoConfig, &cloudconfiglist, secretTestKey)
			Expect(err).NotTo(HaveOccurred())
		})

		It("when cloud-config map exist", func() {
			configMapKey := types.NamespacedName{
				Namespace: VC_CREDS_SECRET_NS,
				Name:      CONFIGMAP_NAME,
			}
			_, err := r.reconcileConfigMap(vdoctx, vdoConfig, &cloudconfiglist, configMapKey)
			Expect(err).NotTo(HaveOccurred())

			// When configMapIsSame is true
			_, err = r.reconcileConfigMap(vdoctx, vdoConfig, &cloudconfiglist, configMapKey)
			Expect(err).NotTo(HaveOccurred())
		})

	})
})

func initializeVsphereConfigList() []v1alpha1.VsphereCloudConfig {
	cloudConfig1 := v1alpha1.VsphereCloudConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-resource",
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
	cloudConfig2 := v1alpha1.VsphereCloudConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		},
		Spec: v1alpha1.VsphereCloudConfigSpec{
			VcIP:        "2.2.2.2",
			Insecure:    true,
			Credentials: "secret-ref",
			DataCenters: []string{"datacenter-1"},
		},
		Status: v1alpha1.VsphereCloudConfigStatus{},
	}
	var cloudconfiglist []v1alpha1.VsphereCloudConfig
	cloudconfiglist = append(cloudconfiglist, cloudConfig1, cloudConfig2)
	return cloudconfiglist
}

func initializeVDOConfig(namespace string) *v1alpha1.VDOConfig {
	vdoConfig := &v1alpha1.VDOConfig{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vdo-sample",
			Namespace: namespace,
		},
		Spec: v1alpha1.VDOConfigSpec{
			CloudProvider: v1alpha1.CloudProviderConfig{
				VsphereCloudConfigs: []string{"test-resource"},
				Topology: v1alpha1.TopologyInfo{
					Zone:   "k8s-zone-A",
					Region: "k8s-region-A",
				},
			},
			StorageProvider: v1alpha1.StorageProviderConfig{
				VsphereCloudConfig:  "test-resource",
				ClusterDistribution: "",
				FileVolumes:         v1alpha1.FileVolume{},
			},
		},
		Status: v1alpha1.VDOConfigStatus{
			CPIStatus: v1alpha1.CPIStatus{},
			CSIStatus: v1alpha1.CSIStatus{},
		},
	}
	return vdoConfig
}
func initializeVDOConfigWithStatus(namespace string) *v1alpha1.VDOConfig {
	cpiStatus := v1alpha1.CPIStatus{
		Phase:      "Deploying",
		StatusMsg:  "",
		NodeStatus: map[string]vdov1alpha1.NodeStatus{"21": "ready"},
	}
	vdoConfig := &v1alpha1.VDOConfig{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vdo-sample",
			Namespace: namespace,
		},
		Spec: v1alpha1.VDOConfigSpec{
			CloudProvider: v1alpha1.CloudProviderConfig{
				VsphereCloudConfigs: []string{"test-resource"},
				Topology: v1alpha1.TopologyInfo{
					Zone:   "k8s-zone-A",
					Region: "k8s-region-A",
				},
			},
			StorageProvider: v1alpha1.StorageProviderConfig{
				VsphereCloudConfig:  "test-resource",
				ClusterDistribution: "",
				FileVolumes:         v1alpha1.FileVolume{},
			},
		},
		Status: v1alpha1.VDOConfigStatus{
			CPIStatus: cpiStatus,
			CSIStatus: v1alpha1.CSIStatus{},
		},
	}
	return vdoConfig
}

func createConfigFile(filePath, fileContents string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	Expect(err).NotTo(HaveOccurred())

	err = os.Truncate(filePath, 0)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	_, err = file.Write([]byte(fileContents))

	return err
}

var _ = Describe("TestUpdatingKubeletPath", func() {
	Context("when custom Kubelet Path is provided", func() {
		ctx := context.Background()
		defer GinkgoRecover()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		daemonSet := &v1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vsphere-csi-node",
				Namespace: "kube-system",
				Labels:    map[string]string{"app": "test-label"},
			},
			Spec: v1.DaemonSetSpec{
				Template: v12.PodTemplateSpec{
					Spec: v12.PodSpec{
						Volumes: []v12.Volume{
							{
								Name: string(podVolName),
								VolumeSource: v12.VolumeSource{
									HostPath: &v12.HostPathVolumeSource{
										Path: "/var/lib/kubelet",
										Type: nil,
									},
								},
							},
						},
						Containers: []v12.Container{
							{
								Name: CSI_DAEMONSET_NAME,
								VolumeMounts: []v12.VolumeMount{
									{
										Name:      string(podVolName),
										MountPath: "/var/lib/kubelet",
									},
								},
							},
							{
								Name: string(csiNodeDriverRegName),
								Env: []v12.EnvVar{
									{
										Name:  CSI_DRIVER_REG_PATH,
										Value: "/var/lib/kubelet/plugins/csi.vsphere.vmware.com/csi.sock",
									},
									{
										Name:  "ADDRESS",
										Value: "/csi/csi.sock",
									},
								},
								LivenessProbe: &v12.Probe{
									Handler: v12.Handler{
										Exec: &v12.ExecAction{
											Command: []string{
												"/csi-node-driver-registrar",
												"--kubelet-registration-path=/var/lib/kubelet/plugins/csi.vsphere.vmware.com/csi.sock",
												"--mode=kubelet-registration-probe",
											},
										},
									},
								},
							},
						}}}},
			Status: v1.DaemonSetStatus{
				NumberUnavailable: 1,
			},
		}

		Expect(r.Create(vdoctx, daemonSet, &client.CreateOptions{})).NotTo(HaveOccurred())

		It("Should update DaemonSet without error", func() {
			Expect(r.updateCSIDaemonSet(vdoctx, "/var/data/kubelet")).Should(Succeed())
			key := types.NamespacedName{
				Namespace: DEPLOYMENT_NS,
				Name:      CSI_DAEMONSET_NAME,
			}
			ds := v1.DaemonSet{}
			Expect(r.Get(ctx, key, &ds)).Should(Succeed())

			Expect(ds.Spec.Template.Spec.Volumes[0].HostPath.Path).Should(BeEquivalentTo("/var/data/kubelet"))
			Expect(ds.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath).Should(BeEquivalentTo("/var/data/kubelet"))
			Expect(ds.Spec.Template.Spec.Containers[1].LivenessProbe.Exec.Command[1]).Should(BeEquivalentTo("--kubelet-registration-path=/var/data/kubelet/plugins/csi.vsphere.vmware.com/csi.sock"))
		})

	})
})

var _ = Describe("TestUpdatingCSIConfigmap", func() {
	Context("when feature state is updated successfully", func() {
		ctx := context.Background()
		defer GinkgoRecover()

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		r := VDOConfigReconciler{
			Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger: ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme: s,
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		configMap := &v12.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "internal-feature-states.csi.vsphere.vmware.com",
				Namespace: "kube-system",
			},
			Immutable: nil,
			Data: map[string]string{
				"improved-csi-idempotency": "false",
				"improved-volume-topology": "false",
				"online-volume-extend":     "true",
				"trigger-csi-fullsync:":    "false",
				"async-query-volume":       "false",
				"csi-auth-check":           "true",
				"csi-migration":            "false",
			},
			BinaryData: nil,
		}

		Expect(r.Create(vdoctx, configMap, &client.CreateOptions{})).NotTo(HaveOccurred())

		It("Should update Configmap without error", func() {
			Expect(r.updateCSIConfigmap(vdoctx)).Should(Succeed())
			Expect(configMap.Data[CSI_NODE_ID]).ShouldNot(BeNil())
		})

	})
})

var _ = Describe("TestCheckCompatAndRetrieveSpec", func() {

	Context("When fetching deployment yamls", func() {
		RegisterFailHandler(Fail)
		ctx := context.Background()

		var sim *simulator.Server
		var vc_user, vc_pwd string

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		expect := version.Info{
			Major:     "1",
			Minor:     "21",
			GitCommit: "v1.21.1",
		}
		// get server object with expected version info
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			output, err := json.Marshal(expect)
			Expect(err).NotTo(HaveOccurred())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err = w.Write(output)
			Expect(err).NotTo(HaveOccurred())
		}))

		r := VDOConfigReconciler{
			Client:       fake2.NewClientBuilder().WithRuntimeObjects().Build(),
			Logger:       ctrllog.Log.WithName("VDOConfigControllerTest"),
			Scheme:       s,
			ClientConfig: &restclient.Config{Host: server.URL},
		}

		vdoctx := vdocontext.VDOContext{
			Context: ctx,
			Logger:  r.Logger,
		}

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-resource",
				Namespace: "default",
			},
		}

		SessionFn = func(ctx context.Context,
			server string, datacenters []string, username, password, thumbprint string) (*session.Session, error) {
			return &session.Session{
				Client:         nil,
				Datacenters:    nil,
				VsphereVersion: "7.0.3",
			}, nil

		}

		matrixString := "{\n    \"CSI\" : {\n            \"2.2.1\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"min\": \"1.18\", \"max\": \"1.21\"},\n                    \"isCPIRequired\" : false,\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"]\n                    }\n        },\n    \"CPI\" : {\n            \"1.20.0\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"skewVersion\": \"1.21\"},\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"]\n                    }\n        }\n             \n}"

		vdoConfig := initializeVDOConfig("default")
		Expect(r.Create(vdoctx, vdoConfig)).Should(Succeed())

		//Setup VC SIM
		model := simulator.VPX()
		model.Host = 0 // ClusterHost only

		defer model.Remove()
		err := model.Create()
		if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		model.Service.TLS = new(tls.Config)

		sim = model.Service.NewServer()
		vc_pwd, _ = sim.URL.User.Password()
		vc_user = sim.URL.User.Username()

		It("should create the resources without error", func() {

			secret := &v12.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-ref",
					Namespace: "kube-system",
				},
				Data: map[string][]byte{
					"username": []byte(vc_user),
					"password": []byte(vc_pwd),
				},
			}
			Expect(r.Create(ctx, secret)).Should(Succeed())

			cloudConfig := &v1alpha1.VsphereCloudConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
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
			Expect(r.Create(ctx, cloudConfig)).Should(Succeed())
			defer sim.Close()
		})

		It("Test Config URL", func() {
			matrixConfig := "https://raw.githubusercontent.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/main/artifacts/compatibility-yaml/compatibility-v0.2.1.yaml"
			os.Setenv(COMPAT_MATRIX_CONFIG_URL, matrixConfig)
			err := r.CheckCompatAndRetrieveSpec(vdoctx, req, vdoConfig, matrixConfig)
			Expect(err).NotTo(HaveOccurred())
			os.Setenv(COMPAT_MATRIX_CONFIG_URL, "")
		})

		It("Should fetch deployment yamls without error", func() {
			err := r.CheckCompatAndRetrieveSpec(vdoctx, req, vdoConfig, matrixString)
			Expect(err).NotTo(HaveOccurred())
		})

		matrixStringIncompatibleCSI := "{\n    \"CSI\" : {\n            \"2.2.1\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"min\": \"1.23\", \"max\": \"1.24\"},\n                    \"isCPIRequired\" : false,\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"]\n                    }\n        },\n    \"CPI\" : {\n            \"1.20.0\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"skewVersion\": \"1.21\"},\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"]\n                    }\n        }\n             \n}"
		It("Should fail with CSI Version not available error when CSI/CPI is configured", func() {
			err := r.CheckCompatAndRetrieveSpec(vdoctx, req, vdoConfig, matrixStringIncompatibleCSI)
			Expect(err.Error()).Should(Equal("could not fetch compatible CSI version for vSphere version and k8s version "))
		})

		matrixStringIncompatibleCPI := "{\n    \"CSI\" : {\n            \"2.2.1\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"min\": \"1.18\", \"max\": \"1.21\"},\n                    \"isCPIRequired\" : false,\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"]\n                    }\n        },\n    \"CPI\" : {\n            \"1.20.0\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.0\", \"max\": \"7.0.7\"},\n                    \"k8s\" : {\"skewVersion\": \"1.22\"},\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"]\n                    }\n        }\n             \n}"
		It("Should fail with CPI Version not available error when CSI/CPI is configured", func() {
			err := r.CheckCompatAndRetrieveSpec(vdoctx, req, vdoConfig, matrixStringIncompatibleCPI)
			Expect(err.Error()).Should(Equal("could not fetch compatible CPI version for vSphere version and k8s version "))
		})

		vdoConfigWithoutCpi := initializeVDOConfig("default")
		vdoConfigWithoutCpi.Spec.CloudProvider = v1alpha1.CloudProviderConfig{}
		It("Should fetch deployment yamls without errors if only CSI is configured", func() {
			err := r.CheckCompatAndRetrieveSpec(vdoctx, req, vdoConfigWithoutCpi, matrixStringIncompatibleCPI)
			Expect(err).NotTo(HaveOccurred())
			defer server.Close()
		})
	})
})

var _ = Describe("Test SetupWithMgs", func() {
	Context("SettingUpWithMgs", func() {
		RegisterFailHandler(Fail)

		s := scheme.Scheme
		s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.VDOConfig{})

		It("When SettingUp with Mgr success", func() {
			mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
			Expect(err).NotTo(HaveOccurred())
			err = (&VDOConfigReconciler{
				Client: fake2.NewClientBuilder().WithRuntimeObjects().Build(),
				Logger: ctrllog.Log.WithName("reconcileCPIConfigurationTest"),
				Scheme: s,
			}).SetupWithManager(mgr)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

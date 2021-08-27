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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	vdocontext "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/context"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
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
		})

		It("should reconcile deployment status without error", func() {

			Expect(r.reconcileCSIDeploymentStatus(vdoctx, clientSet)).NotTo(HaveOccurred())
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

		vdoConfig := &v1alpha1.VDOConfig{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vdo-sample",
				Namespace: "default",
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

		Expect(r.Create(ctx, vdoConfig)).Should(Succeed())

		clientSet := fake.NewSimpleClientset()
		Expect(clientSet).NotTo(BeNil())

		secretTestKey := types.NamespacedName{
			Name:      "test-secret",
			Namespace: "kube-system",
		}

		It("should reconcile configmap without error", func() {
			_, err := r.reconcileConfigMap(vdoctx, vdoConfig, &cloudconfiglist, secretTestKey)
			Expect(err).NotTo(HaveOccurred())
		})
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

		vdoConfig := &v1alpha1.VDOConfig{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vdo-sample",
				Namespace: "default",
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

		It("should reconcile configmap without error", func() {
			_, err := r.reconcileCSISecret(vdoctx, vdoConfig, &cloudConfig)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

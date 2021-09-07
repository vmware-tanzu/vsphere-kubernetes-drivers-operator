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
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	vdocontext "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/context"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/drivers/cpi"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/models"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/session"
	"github.com/vmware/govmomi/object"
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

		cloudconfiglist := initializeVsphereConfigList()

		vdoConfig := initializeVDOConfig()

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

		vdoConfig := initializeVDOConfig()

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

		vdoConfig := initializeVDOConfig()
		Expect(r.Create(vdoctx, vdoConfig)).Should(Succeed())

		node1 := v12.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "test-node1"},
			Spec:       v12.NodeSpec{ProviderID: "vsphere://testid1"},
		}

		node2 := v12.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "test-node2"},
			Spec:       v12.NodeSpec{ProviderID: "vsphere://testid2"},
		}

		It("should create the nodes without error", func() {
			_, err := clientSet.CoreV1().Nodes().Create(vdoctx, &node1, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = clientSet.CoreV1().Nodes().Create(vdoctx, &node2, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

		})

		It("should reconcile providerID without error", func() {
			_, nodelist, err := r.reconcileNodeProviderID(vdoctx, vdoConfig, clientSet, &cloudconfiglist)
			Expect(err).NotTo(HaveOccurred())
			val, ok := nodelist[node1.Name]
			Expect(ok).To(BeTrue())
			Expect(val).To(BeTrue())

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
		vdoConfig := initializeVDOConfig()

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
			_, nodelist, err := r.reconcileNodeProviderID(vdoctx, vdoConfig, clientSet, &cloudconfiglist)
			Expect(err).NotTo(HaveOccurred())
			_, ok := nodelist[node4.Name]
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

		vdoConfig := initializeVDOConfig()

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
			_, nodelist, err := r.reconcileNodeProviderID(vdoctx, vdoConfig, clientSet, &cloudconfiglist)
			Expect(err).NotTo(HaveOccurred())
			val, ok := nodelist[node5.Name]
			Expect(ok).To(BeTrue())
			Expect(val).To(BeTrue())

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

		vdoConfig := initializeVDOConfig()

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
			_, _, err := r.reconcileNodeProviderID(vdoctx, vdoConfig, clientSet, &cloudconfiglist)
			Expect(err).To(HaveOccurred())

		})

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

	testnodelist := map[string]bool{
		node1.Name: true,
		node2.Name: false,
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

		vdoConfig := initializeVDOConfig()
		Expect(r.Create(vdoctx, vdoConfig)).Should(Succeed())

		_, err := clientSet.CoreV1().Nodes().Create(vdoctx, &node1, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		_, err = clientSet.CoreV1().Nodes().Create(vdoctx, &node2, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		It("should reconcile node label without error", func() {
			err := r.reconcileNodeLabel(vdoctx, req, clientSet, testnodelist)
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
			err = r.fetchCsiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.21")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fetch CPI deployment yamls without error", func() {
			err = r.fetchCpiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.21")
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
			err = r.fetchCsiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.21")
			Expect(err).To(HaveOccurred())
		})

		It("should fetch CPI deployment yamls with error", func() {
			err = r.fetchCpiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.21")
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
			err = r.fetchCsiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.22")
			Expect(err).To(HaveOccurred())
		})

		It("should fetch CPI deployment yamls with error", func() {
			err = r.fetchCpiDeploymentYamls(vdoctx, matrix, []string{"7.0.3"}, "1.22")
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
			err = r.fetchCsiDeploymentYamls(vdoctx, matrix, []string{"6.5.3"}, "1.17")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fetch CPI deployment yamls without error", func() {
			err = r.fetchCpiDeploymentYamls(vdoctx, matrix, []string{"6.5.3"}, "1.17")
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

func initializeVDOConfig() *v1alpha1.VDOConfig {
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
	return vdoConfig
}

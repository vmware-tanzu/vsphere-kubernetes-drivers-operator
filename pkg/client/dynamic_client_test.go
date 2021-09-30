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
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	vdocontext "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/context"
	"k8s.io/klog/v2/klogr"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var _ = Describe("Dynamic Client Tests", func() {

	var (
		testEnv   *envtest.Environment
		k8sClient client.Client
		vdoctx    = vdocontext.VDOContext{
			Context: context.Background(),
			Logger:  klogr.New(),
		}
		specFilePath       = "/tmp/vdo-spec.yaml"
		compMatrixFilePath = "/tmp/matrix.yaml"
	)

	BeforeEach(func() {
		testEnv = &envtest.Environment{
			CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
			ErrorIfCRDPathMissing: true,
		}
		cfg, _ := testEnv.Start()
		Expect(cfg).NotTo(BeNil())

		var err error
		k8sClient, err = client.New(cfg, client.Options{})
		Expect(err).NotTo(HaveOccurred())
		Expect(k8sClient).NotTo(BeNil())
	})

	AfterEach(func() {
		Expect(testEnv.Stop()).NotTo(HaveOccurred())
	})

	Context("when applying yaml spec file from URL", func() {
		It("should parse the spec file", func() {
			yamlBytes, err := GenerateYamlFromUrl("https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/master/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml")
			Expect(err).NotTo(HaveOccurred())

			Expect(yamlBytes).ShouldNot(BeEmpty())

			_, err = ParseAndProcessK8sObjects(vdoctx, k8sClient, yamlBytes, "", "CREATE")
			Expect(err).NotTo(HaveOccurred())

			yamlBytes, err = GenerateYamlFromUrl("https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/master/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(yamlBytes)).NotTo(BeZero())

			_, err = ParseAndProcessK8sObjects(vdoctx, k8sClient, yamlBytes, "", "CREATE")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when applying yaml spec file from file", func() {
		BeforeEach(func() {
			fileContents := "kind: Deployment\napiVersion: apps/v1\nmetadata:\n  name: vsphere-csi-controller\n  namespace: kube-system\nspec:\n  replicas: 1\n  selector:\n    matchLabels:\n      app: vsphere-csi-controller\n  template:\n    metadata:\n      labels:\n        app: vsphere-csi-controller\n        role: vsphere-csi\n    spec:\n      serviceAccountName: vsphere-csi-controller\n      nodeSelector:\n        node-role.kubernetes.io/master: \"\"\n      tolerations:\n        - key: node-role.kubernetes.io/master\n          operator: Exists\n          effect: NoSchedule\n        # uncomment below toleration if you need an aggressive pod eviction in case when\n        # node becomes not-ready or unreachable. Default is 300 seconds if not specified.\n        #- key: node.kubernetes.io/not-ready\n        #  operator: Exists\n        #  effect: NoExecute\n        #  tolerationSeconds: 30\n        #- key: node.kubernetes.io/unreachable\n        #  operator: Exists\n        #  effect: NoExecute\n        #  tolerationSeconds: 30\n      dnsPolicy: \"Default\"\n      containers:\n        - name: csi-attacher\n          image: quay.io/k8scsi/csi-attacher:v3.1.0\n          args:\n            - \"--v=4\"\n            - \"--timeout=300s\"\n            - \"--csi-address=$(ADDRESS)\"\n            - \"--leader-election\"\n          env:\n            - name: ADDRESS\n              value: /csi/csi.sock\n          volumeMounts:\n            - mountPath: /csi\n              name: socket-dir\n        - name: csi-resizer\n          image: quay.io/k8scsi/csi-resizer:v1.1.0\n          args:\n            - \"--v=4\"\n            - \"--timeout=300s\"\n            - \"--handle-volume-inuse-error=false\"\n            - \"--csi-address=$(ADDRESS)\"\n            - \"--kube-api-qps=100\"\n            - \"--kube-api-burst=100\"\n            - \"--leader-election\"\n          env:\n            - name: ADDRESS\n              value: /csi/csi.sock\n          volumeMounts:\n            - mountPath: /csi\n              name: socket-dir\n        - name: vsphere-csi-controller\n          image: gcr.io/cloud-provider-vsphere/csi/release/driver:v2.2.1\n          args:\n            - \"--fss-name=internal-feature-states.csi.vsphere.vmware.com\"\n            - \"--fss-namespace=$(CSI_NAMESPACE)\"\n          imagePullPolicy: \"Always\"\n          env:\n            - name: CSI_ENDPOINT\n              value: unix:///csi/csi.sock\n            - name: X_CSI_MODE\n              value: \"controller\"\n            - name: VSPHERE_CSI_CONFIG\n              value: \"/etc/cloud/csi-vsphere.conf\"\n            - name: LOGGER_LEVEL\n              value: \"PRODUCTION\" # Options: DEVELOPMENT, PRODUCTION\n            - name: INCLUSTER_CLIENT_QPS\n              value: \"100\"\n            - name: INCLUSTER_CLIENT_BURST\n              value: \"100\"\n            - name: CSI_NAMESPACE\n              valueFrom:\n                fieldRef:\n                  fieldPath: metadata.namespace\n            - name: X_CSI_SERIAL_VOL_ACCESS_TIMEOUT\n              value: 3m\n          volumeMounts:\n            - mountPath: /etc/cloud\n              name: vsphere-config-volume\n              readOnly: true\n            - mountPath: /csi\n              name: socket-dir\n          ports:\n            - name: healthz\n              containerPort: 9808\n              protocol: TCP\n            - name: prometheus\n              containerPort: 2112\n              protocol: TCP\n          livenessProbe:\n            httpGet:\n              path: /healthz\n              port: healthz\n            initialDelaySeconds: 10\n            timeoutSeconds: 3\n            periodSeconds: 5\n            failureThreshold: 3\n        - name: liveness-probe\n          image: quay.io/k8scsi/livenessprobe:v2.2.0\n          args:\n            - \"--v=4\"\n            - \"--csi-address=/csi/csi.sock\"\n          volumeMounts:\n            - name: socket-dir\n              mountPath: /csi\n        - name: vsphere-syncer\n          image: gcr.io/cloud-provider-vsphere/csi/release/syncer:v2.2.1\n          args:\n            - \"--leader-election\"\n            - \"--fss-name=internal-feature-states.csi.vsphere.vmware.com\"\n            - \"--fss-namespace=$(CSI_NAMESPACE)\"\n          imagePullPolicy: \"Always\"\n          ports:\n            - containerPort: 2113\n              name: prometheus\n              protocol: TCP\n          env:\n            - name: FULL_SYNC_INTERVAL_MINUTES\n              value: \"30\"\n            - name: VSPHERE_CSI_CONFIG\n              value: \"/etc/cloud/csi-vsphere.conf\"\n            - name: LOGGER_LEVEL\n              value: \"PRODUCTION\" # Options: DEVELOPMENT, PRODUCTION\n            - name: INCLUSTER_CLIENT_QPS\n              value: \"100\"\n            - name: INCLUSTER_CLIENT_BURST\n              value: \"100\"\n            - name: CSI_NAMESPACE\n              valueFrom:\n                fieldRef:\n                  fieldPath: metadata.namespace\n          volumeMounts:\n            - mountPath: /etc/cloud\n              name: vsphere-config-volume\n              readOnly: true\n        - name: csi-provisioner\n          image: quay.io/k8scsi/csi-provisioner:v2.1.0\n          args:\n            - \"--v=4\"\n            - \"--timeout=300s\"\n            - \"--csi-address=$(ADDRESS)\"\n            - \"--kube-api-qps=100\"\n            - \"--kube-api-burst=100\"\n            - \"--leader-election\"\n            - \"--default-fstype=ext4\"\n            # needed only for topology aware setup\n            #- \"--feature-gates=Topology=true\"\n            #- \"--strict-topology\"\n          env:\n            - name: ADDRESS\n              value: /csi/csi.sock\n          volumeMounts:\n            - mountPath: /csi\n              name: socket-dir\n      volumes:\n      - name: vsphere-config-volume\n        secret:\n          secretName: vsphere-config-secret\n      - name: socket-dir\n        emptyDir: {}\n---\napiVersion: v1\ndata:\n  \"csi-migration\": \"false\"\n  \"csi-auth-check\": \"true\"\n  \"online-volume-extend\": \"true\"\nkind: ConfigMap\nmetadata:\n  name: internal-feature-states.csi.vsphere.vmware.com\n  namespace: kube-system\n---\napiVersion: storage.k8s.io/v1 # For k8s 1.17 use storage.k8s.io/v1beta1\nkind: CSIDriver\nmetadata:\n  name: csi.vsphere.vmware.com\nspec:\n  attachRequired: true\n  podInfoOnMount: false\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: vsphere-csi-controller\n  namespace: kube-system\n  labels:\n    app: vsphere-csi-controller\nspec:\n  ports:\n    - name: ctlr\n      port: 2112\n      targetPort: 2112\n      protocol: TCP\n    - name: syncer\n      port: 2113\n      targetPort: 2113\n      protocol: TCP\n  selector:\n    app: vsphere-csi-controller"
			compMatrixContent := "{\n    \"CSI\" : {\n            \"2.2.1\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.1\", \"max\": \"7.0.4\"},\n                    \"k8s\" : {\"min\": \"1.18\", \"max\": \"1.22\"},\n                    \"isCPIRequired\" : false,\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-controller-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/rbac/vsphere-csi-node-rbac.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-controller-deployment.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes-sigs/vsphere-csi-driver/v2.2.1/manifests/v2.2.1/deploy/vsphere-csi-node-ds.yaml\"]\n                    }\n        },\n    \"CPI\" : {\n            \"1.20.0\" : {\n                    \"vSphere\" : { \"min\" : \"6.7.1\", \"max\": \"7.0.4\"},\n                    \"k8s\" : {\"skewVersion\": \"1.22\"},\n                    \"deploymentPath\": [\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-roles.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/cloud-controller-manager-role-bindings.yaml\",\n                    \"https://raw.githubusercontent.com/kubernetes/cloud-provider-vsphere/v1.20.0/manifests/controller-manager/vsphere-cloud-controller-manager-ds.yaml\"]\n                    }\n        }\n             \n}"

			err := createFileWithContents(specFilePath, fileContents)
			Expect(err).NotTo(HaveOccurred())
			err = createFileWithContents(compMatrixFilePath, compMatrixContent)
			Expect(err).NotTo(HaveOccurred())
		})
		It("should parse the spec file", func() {
			yamlBytes, err := GenerateYamlFromFilePath(fmt.Sprintf("file:/%s", specFilePath))
			Expect(err).NotTo(HaveOccurred())
			Expect(yamlBytes).ShouldNot(BeEmpty())

			matrix, err := ParseMatrixYaml(fmt.Sprintf("file:/%s", compMatrixFilePath))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(matrix.CPISpecList)).ShouldNot(BeZero())
		})
	})
})

func createFileWithContents(filePath, fileContents string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	Expect(err).NotTo(HaveOccurred())

	err = os.Truncate(filePath, 0)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	_, err = file.Write([]byte(fileContents))

	return err
}

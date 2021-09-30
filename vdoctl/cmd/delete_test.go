package cmd

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var _ = Describe("vdoctl delete", func() {
	Context("when delete command is invoked", func() {
		It("should delete all vdo resources as expected", func() {
			testEnv := &envtest.Environment{
				CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
				ErrorIfCRDPathMissing: true,
			}
			cfg, _ := testEnv.Start()
			Expect(cfg).NotTo(BeNil())

			ctx := context.Background()

			testK8sClient, err := client.New(cfg, client.Options{})
			Expect(err).NotTo(HaveOccurred())
			Expect(testK8sClient).NotTo(BeNil())

			K8sClient = testK8sClient

			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "vmware-system-vdo"},
			}
			testK8sClient.Create(ctx, ns, &client.CreateOptions{})

			cr := &v12.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "vdo-cr-role",
					Labels: map[string]string{"managedby": "vdo"},
				},
				Rules: []v12.PolicyRule{},
			}
			testK8sClient.Create(ctx, cr, &client.CreateOptions{})

			crb := &v12.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "vdo-cr-role-cb",
					Labels: map[string]string{"managedby": "vdo"},
				},
				RoleRef: v12.RoleRef{Name: cr.Name},
			}
			testK8sClient.Create(ctx, crb, &client.CreateOptions{})

			deleteCmd.Run(&cobra.Command{}, []string{})

			Expect(testK8sClient.Delete(ctx, ns, &client.DeleteOptions{})).ShouldNot(Succeed())
			Expect(testK8sClient.Delete(ctx, cr, &client.DeleteOptions{})).ShouldNot(Succeed())
		})
	})
})

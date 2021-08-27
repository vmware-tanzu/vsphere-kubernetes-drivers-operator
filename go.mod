module github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator

go 1.16

require (
	github.com/go-logr/logr v0.3.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/pkg/errors v0.9.1
	github.com/vmware/govmomi v0.26.0
	golang.org/x/tools v0.1.4 // indirect
	gopkg.in/ini.v1 v1.51.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/klog/v2 v2.4.0
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)

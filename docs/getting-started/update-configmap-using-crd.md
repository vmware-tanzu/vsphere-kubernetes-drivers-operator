## CompatibilityConfig

The newly introduced CRD gives user a simpler way to update the compatibility matrix
without using the `vdoctl`.

To use the new CRD you can create the object referred below
```shell
apiVersion: vdo.vmware.com/v1alpha1
kind: CompatibilityConfig
metadata:
  name: compat-matrix-config
  namespace: vmware-system-vdo
spec:
  matrixURL: "https://github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/releases/download/0.3.0-rc/compatibility.yaml"
```

You can update the matrixURL as per your requirement.

**Note** : Make sure you keep the name of the CompatibilityConfig(`compat-matrix-config`) unchanged. The namespace can be updated as per usage.
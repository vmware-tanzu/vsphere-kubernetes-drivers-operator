## Getting Started

VDO(vSphere Kubernetes Driver Operator) is built out of operator-sdk.
The operator is configured to run on master node, with a single replica deployment.

VDO operator is built to run on vanilla k8s cluster as well Openshift clusters

## Pre-requisite

Please ensure the pre-requisites for [CSI](https://docs.vmware.com/en/VMware-vSphere-Container-Storage-Plug-in/2.0/vmware-vsphere-csp-getting-started/GUID-0AB6E692-AA47-4B6A-8CEA-38B754E16567.html) and [CPI](https://cloud-provider-vsphere.sigs.k8s.io/tutorials/kubernetes-on-vsphere-with-kubeadm.html#:~:text=Check%20that%20all%20nodes%20are%20tainted) are met so that the CloudProvider and StorageProvider function properly

## Choose your Deployment Schemes

VDO can be deployed and configured in multiple ways depending on the environment you are working on.
Choose the desrired deployment scheme and follow the relevant guide to bring up and configure VDO.  

| Deployment Method| Links | Remarks |
| ----------- | ----------- | ---------|
| Deployment via source code |  [Guide](getting-started/getting-started-via-code.md)       |          |
| Deployment via Releases   | [Guide](getting-started/getting-started-from-release.md)        |          |
| Deployment via Operator Hub | [Guide](getting-started/getting-started-from-operator-hub.md) | Recommended|

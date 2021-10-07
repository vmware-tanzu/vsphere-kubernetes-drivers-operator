## Getting Started

VDO(vSphere Kubernetes Driver Operator) is built out of operator-sdk.
The operator is configured to run on master node, with a single replica deployment.

VDO operator is built to run on vanilla k8s cluster as well Openshift clusters


### Pre-requisite
It is always recommended using the operator from our [release](https://github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/releases) page, but if you want to use our
latest and the greatest one then you can follow the below steps. This projects assumes you have the following setup.
- go version 1.16 and above
- docker

### Build

There are mainly two components which gets shipped with this project:  
 - VDO Operator
 - VDOCTL (CLI tool to help configure and deploy VDO)

#### VDO Operator
You can build the operator using below command.
```shell
cd $GOPATH
mkdir -p vmware-tanzu
cd vmware-tanzu

git clone https://github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator.git

cd vsphere-kubernetes-drivers-operator
make build
```

#### VDOCTL
To build the cli you can run below commands
```shell
# for Linux env
make build-vdoctl

# for Mac env
make build-vdoctl-mac
```

#### Tests
To run the test case you need to ensure that you have kind cluster installed in your env.
```shell
make test
```

### Deploy 
For Deploying the Operator on vanilla K8s cluster we have the following options:

- On local kind cluster
```shell
make deploy
```
- On live vanilla k8s cluster 
```shell
export K8S_MASTER_IP=YOUR-K8S_MASTER_IP
export K8S_MASTER_SSH_USER=USERNAME
export K8S_MASTER_SSH_PWD=PASSWORD

make deploy-k8s-cluster
```

Refer the [MakeFile](../Makefile) for more details.


### Deploying Drivers

Once the VDO is deployed you need to configure the compatibility-matrix 
and CSI/CPI drivers.
Before starting check whether the VDO is deployed, you will notice that 
the vdo pods are in `ConfigError` state.
```shell
kubectl get pods -n vmware-system-vdo
vmware-system-vdo    vdo-controller-manager-66758456d8-mnqgv      1/2     CreateContainerConfigError   0          11s
```

#### Configure Compatibility Matrix
So to bring the VDO in running state we need to first configure the 
compatibility-matrix using `vdoctl` command line tool. You can either use the 
self made binary of vdoctl from the above steps or you can download the 
vdoctl binary from our release page and place the binary in your system path.

```shell
vdoctl configure compatibility-matrix
✔ Web URL
Web URL https://raw.githubusercontent.com/asifdxtreme/Docs/master/sample/matrix/matrix.yaml
```
Note: You can either use this sample url or create your own matrix.

Generally with each new release a New Compatibility Matrix will be released, 
you can get more details from [here](https://github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/releases).

Once the compatibility matrix is configured, you can re-check the vdo operator running status
```shell
kubectl get pods -n vmware-system-vdo
vmware-system-vdo    vdo-controller-manager-66758456d8-mnqgv      2/2     Running   0          99s
```

#### Configure Drivers
To configure the drivers we need to provide the basic details of your vSphere Platform 
like vc IP, vc Credentials, Datacenter's. You can look into [User Guide](User-Guide.md) to have
more details explanation of how to configure the drivers.



## Getting Started

VDO operator is built out of operator-sdk.
The operator is configured to run on master node, with a single replica deployment.

VDO operator is built to run on vanilla k8s cluster as well Openshift clusters

### Build

There are two components packaged in this repo
1. VDO Operator
2. VDOCTL (CLI tool to help configure and deploy VDO)

#### VDO Operator
Run `make build` to build the VDO operator

#### VDOCTL
Run `make build-vdoctl` to build vdoctl for linux flavour

Run `make build-vdoctl-mac` to build vdoctl for mac flavour

### Tests
Run `make test` to run the test associated with the operator

### Deploy on vanilla K8s cluster

Refer the [MakeFile](Makefile) to build and deploy the operator.

Run `make generate` to generate the scaffolding code from the provided base types

Run `make manifests` to generate the spec files required to deploy the operator.
The spec file to deploy the operator will be available at [vdo-spec.yaml](../artifacts/vanilla/vdo-spec.yaml)

The operator can be deployed

1. locally on a kind cluster `make deploy`

   Run `make deploy` target to `generate`, `build` and `deploy`
   the operator in local kind cluster

3. remotely on a live vanilla k8s cluster `make deploy-k8s-cluster`

   Run `make deploy-k8s-cluster` target to build container and
   deploy the container image & apply the operator spec on the given k8s cluster
   
   Following environment variables need to be set before invoking the target

      1. K8S_MASTER_IP - IP of K8s Master

      2. K8S_MASTER_SSH_USER - username to ssh into k8smaster

      3. K8S_MASTER_SSH_PWD - password to ssh into k8smaster


### Prerequisites

Following are the prerequisites for deploying operator

1. Create Configmap
2. Configuration of VDO

The following steps help in configuring VDO to install/configure the drivers

1. configure compatibility

    - `cat <<EOF | sudo tee comp-matrix-config.yaml
      apiVersion: v1
      kind: ConfigMap
      metadata:
      name: comp-matrix-config
      namespace: vmware-system-vdo
      data:
      versionConfigURL: "matrix-url"
      auto-upgrade: "disabled"`

      `kubectl apply -f comp-matrix-config.yaml`

2. create secret

    - `cat <<EOF | sudo tee secret.yaml
      apiVersion: v1
      kind: Secret
      metadata:
      name: vc-name-creds
      namespace: kube-system
      type: kubernetes.io/basic-auth
      stringData:
      username: "vc-username"
      password: "vc-password"`

      `kubectl apply -f secret.yaml`

3. create VsphereCloudConfig resource
    - credentials field in the resource refers to the name of the secret
4. create VDOconfig resource
    - Cloud Provider can take multiple instances of VsphereCloudConfig resource
    - Storage provider takes a single VsphereCLoudConfig resource


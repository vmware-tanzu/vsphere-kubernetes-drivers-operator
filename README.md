# vSphere Kubernetes Drivers Operator  

[![Build](https://github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/actions/workflows/BuildAndDeploy.yml/badge.svg)](https://github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/actions/workflows/BuildAndDeploy.yml)

## Overview

The vSphere Kubernetes Drivers Operator project is designed to simplify and
automate the lifecycle management of CNI, CSI and CPI drivers in a Kubernetes
cluster running on vSphere. It will do this by following the Kubernetes operator
model in which desired state and configuration is defined in a custom resource
and the operator(s) running in Kubernetes will attempt to reconcile the drivers
to satisfy the desired state.

## Goals

Our goal is to make vSphere the best place to run any Kubernetes and to vastly
simplify the experience of deploying Kubernetes to vSphere.

### Phase 1

- Improve the resiliance, error reporting and state transition status of the
  existing drivers
- Provide a single point of reference to the information necessary to perform
  a manual install of the vSphere drivers
- Ensure that the existing documentation for a manual install is up to date
  and correct
- Ensure that common error conditions are tested, feedback from the driver
  is clear and remediation information is available
- Document how to contribute

### Phase 2

- Design a consolidated configuration format - the Spec of the custom
  resource(s) - that covers the config needed for all of the drivers
- Come up with a design spec for the operator, including but not exclusive to:
  - The topology - should it be a single operator with a controller per
    driver or multiple operators?
  - Build depedencies - are the existing drivers statically linked
    into the new operator or does it load them dynamically?
  - How do the drivers consume the new CRD format? Should it be cloned?
    Should the config format be pluggable?
  - Control flow for each driver. What are the preconditions that have
    to be met before it can be installed?
  - Interdependencies between drivers. How are state transitions managed?
  - What Status would users expect to be able to see from the drivers operator?
  - What CNI implementations are we supporting?
  - How should we define health and liveness?
  - What metrics might we want to export for monitoring?
  - How is compatibility defined and enforced? Between the drivers themselves,
    the K8S version in use and the target vSphere
  - Define common expected failure conditions and how these should be
    reported to the user
  - Ensure that requirements for specific distributions such as OpenShift
    are properly captured
- Define basic driver framework, including makefiles, build and test.
  Include proposed framework(s) Eg. Kubebuilder
- Create controllers with stubs that assert the defined state transitions,
  status reporting and basic lifecycle functions work
- Build out controllers for each of the drivers incrementally ensuring
  integration tests are added where necessary
- End goal of Phase 2 is to have a functioning drivers operator for
  upstream K8S that installs and deletes drivers

### Phase 3

- Deliver any additional requirements for specific distributions, such as OpenShift

vSphere Kubernetes Drivers Operator (VDO) is a kubernetes operator
responsible for installing CPI and CSI vSphere drivers
enabling k8s workloads to run on vSphere

## Prerequisites

VDO can run on vanilla as well as OpenShift k8s clusters
It is expected that the kubernetes master/worker vms are setup to work on vSphere

## Getting Started

VDO operator is built out of operator-sdk.
The operator is configured to run on master node, with a single replica deployment.

Refer the [MakeFile](Makefile) to build and deploy the operator.
The operator can be deployed

1. locally on a kind cluster `make deploy`

2.remotely on a live k8s cluster `make deploy-k8s-cluster`

Run `make generate` to generate the scaffolding code from the provided base types

Run `make manifests` to generate the spec files required to deploy the operator

### Configuration

Pre-requisite for deploying operator

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

### Deploy to kind cluster

Run `make deploy` target to `generate`, `build` and `deploy`
the operator in local kind cluster

### Deploy to K8S cluster

Following environment variables need to be set before invoking the target

1. K8S_MASTER_IP - IP of K8s Master

2. K8S_MASTER_SSH_USER - username to ssh into k8smaster

3. K8S_MASTER_SSH_PWD - password to ssh into k8smaster

Run `make deploy-k8s-cluster` target to build container and
deploy the container image in the given k8s cluster

## Community

All of the exising CPI, CSI and CNI driver projects are maintained in GitHub

- [vSphere Cloud Provider](https://github.com/kubernetes/cloud-provider-vsphere)
- [vSphere CSI Storage Driver](https://github.com/kubernetes-sigs/vsphere-csi-driver)
- [vSphere Antrea CNI Driver](https://github.com/vmware-tanzu/antrea)

We really value the community of developers and vSphere users who run Kubernetes
on vSphere and our goal is to ensure that the vSphere Drivers Operator is designed
and developed 100% in the open. As such, we will be using GitHub issues for
tracking all of our work and GitHub markdown for our designs. We will be starting
a regular call where design decisions are discussed and we commit to ensuring
that decisions are well documented.

If you have an interest in contributing or submitting requirements,
we'd love to hear from you!

## Contributing

### Project Scope

This is a brand new project we are launching on GitHub, developed and designed upstream.
We will be setting up a slack channel, regular public developer meetings
and design discussions in the next few weeks.
Please watch this space.

### How to Contribute

The vSphere Kubernetes Drivers Opearator project team
welcomes contributions from the community.
If you wish to contribute code and
you have not signed our contributor license agreement (CLA),
our bot will update the issue when you open a Pull Request.
For any questions about the CLA process,
please refer to our FAQ

Please get in touch with us via Slack or
come to one of our meetings if you want to get involved.

## License

VDO is licensed under the Apache License, version 2.0

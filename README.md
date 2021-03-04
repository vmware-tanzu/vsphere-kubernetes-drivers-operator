# vSphere Kubernetes Drivers Operator

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

## License

Antrea is licensed under the [Apache License, version 2.0](LICENSE)
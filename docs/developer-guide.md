## Developer Guide

This document helps you get started using the Volcano code base. If you follow this guide and find some problem, please take a few minutes to update this file.

- [Setting up the developer env](#setting-up-the-developer-env)
- [Building the code from source](#building-the-code-from-source)
- [Running the Test Cases](#running-the-test-cases)
- [Raising the PR and doing the pre-checks](#Raising-the-PR-and-doing-the-prechecks)


## Setting up the developer env

You will need to fork the repository `vsphere-kubernetes-drivers-operator` into your own workspace and clone the main branch to `$GOPATH/src/vsphere-kubernetes-drivers-operator` for the code to work correctly.

```bash
git clone https://github.com/$USER/vsphere-kubernetes-drivers-operator.git
```

## Building the code from source

There are mainly two components which gets shipped with this project:
- VDO Operator
- VDOCTL (CLI tool to help configure and deploy VDO)

### VDO Operator

To build VDO for the host architecture, go to the source root and run:

```bash
make build
```

#### Deploy VDO

To deploy the operator on vanilla k8s cluster, we have the following options:

- On local kind cluster
```shell
make deploy
```
- On live vanilla k8s cluster
```shell
# Set up environment variables by
export K8S_MASTER_IP=YOUR-K8S_MASTER_IP
export K8S_MASTER_SSH_USER=USERNAME
export K8S_MASTER_SSH_PWD=PASSWORD
```

```shell
make deploy-k8s-cluster
```

### VDOCTL

To build the CLI `vdoctl`, run the below commands:

- for Linux env
```shell
make build-vdoctl
```

- for Mac env
```shell
make build-vdoctl-mac
```

## Running the Test Cases

### Unit Tests

All the available unit tests can be with:

```shell
make test
```

## Raising the PR and doing the prechecks

Before sending pull requests you should at least make sure your changes have passed the unit tests and verified the changes on a vanilla k8s cluster. We only merge pull requests when **all** the checks are passing.

- All the packages and files added should have Unit tests corresponding to it.
- Unit tests are written using the standard Go testing package.
- [Code Coverage](https://codecov.io/gh/vmware-tanzu/vsphere-kubernetes-drivers-operator) report should not have any decrease in the code coverage percentage with respect to the Unit Tests.
- Concurrent unit test runs must pass.
- At least **two** reviewers should approve the pull request before merging.
- [Build and Deploy](https://github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/actions/workflows/BuildAndDeploy.yml) pipeline check should pass for the corresponding request.
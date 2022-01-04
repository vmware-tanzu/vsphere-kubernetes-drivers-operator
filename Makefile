# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.1.0

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# build variables
KIND_CLUSTER_NAME 	?= kind
KUBECONFIG       	?= $$($(KIND) get kubeconfig --name $(KIND_CLUSTER_NAME))
BINARY_NAME			= vspheredriver-operator
IMAGE_TAG         	?= latest
BUILD_NUMBER 	  	?= 00000000 # from gobuild
BUILD_VERSION 		?= $(shell git describe --always 2>/dev/null)
ARTIFACTS_DIR		?= artifacts
CRC					?= crc
SPEC_FILE			?= vdo-spec.yaml
CRC					?= crc
OC_CERTIFIED_LATEST_VERSION ?= 0.1.0

# Configure the golangci-lint timeout if an environment variable exists
ifneq ($(origin LINT_TIMEOUT), undefined)
GOLANGCI_LINT_TIMEOUT = ${LINT_TIMEOUT}
else
# Set the default timeout to 2m
GOLANGCI_LINT_TIMEOUT = 2m
endif

# if kind node image is passed use that instead of the default one used by kind
ifneq ($(origin KIND_NODE_IMAGE), undefined)
LOCAL_KIND_NODE_IMAGE = --image ${KIND_NODE_IMAGE}
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# vmware.com/vspheredriveroperator-bundle:$VERSION and vmware.com/vspheredriveroperator-catalog:$VERSION.
# default-route-openshift-image-registry.apps-crc.testing/vmware-system-vdo
# for openshift crc clusters pass the IMAGE_TAG_BASE=<private-registry> along with make manifests-openshift
IMAGE_TAG_BASE ?= vmware.com/vdo

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# Image URL to use all building/pushing image targets
IMG ?= ${IMAGE_TAG_BASE}:${BUILD_VERSION}
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Get the golang image path
ifneq ($(origin DOCKER_PROXY), undefined)
DOCKER_IMAGE_PROXY = --build-arg DOCKER_PROXY=${DOCKER_PROXY}
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen kustomize ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) output:rbac:dir=./config/rbac rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	@mkdir -p $(ARTIFACTS_DIR)/vanilla
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/vanilla_k8s > $(ARTIFACTS_DIR)/vanilla/vdo-spec.yaml

manifests-openshift: kustomize
	@echo "** Making manifest based on the latest oc certified version $(OC_CERTIFIED_LATEST_VERSION)"
	@mkdir -p $(ARTIFACTS_DIR)/staging-openshift
	@cp artifacts/oc-certified/$(OC_CERTIFIED_LATEST_VERSION)/manifests/vsphere-kubernetes-drivers-operator.clusterserviceversion.yaml $(ARTIFACTS_DIR)/staging-openshift/
	@cp config/openshift/crd/vdoconfigs.vdo.vmware.com-crd.yaml $(ARTIFACTS_DIR)/staging-openshift/
	@cp config/openshift/crd/vspherecloudconfigs.vdo.vmware.com-crd.yaml $(ARTIFACTS_DIR)/staging-openshift/
	@cp config/openshift/rbac/vdo-controller-manager-metrics-service.yaml $(ARTIFACTS_DIR)/staging-openshift/
	@echo "** Staging manifest has been created in artifacts/openshift"


generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.xml

##@ Build

build: generate fmt vet lint ## Build manager binary.
	go build -o bin/manager main.go

build-vdoctl: generate fmt vet lint ## Build manager binary.
	GOOS=linux GOARCH=amd64 go build -o bin/vdoctl vdoctl/main.go

build-vdoctl-mac: generate fmt vet lint
	go build -o bin/vdoctl vdoctl/main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} ${DOCKER_IMAGE_PROXY} --rm .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

deploy-crc: build ## Builds and deploys the operator to a CRC cluster. Pass the url of internal registry as varaible IMAGE_TAG_BASE=<url>
	@echo "**** Ensure you have setup crc cluster and internal registry ***** "
	$(eval $(crc oc-env))
	oc new-project vmware-system-vdo
	@echo "**** Completed Setup of new project ***** "
	$(MAKE) docker-build
	$(MAKE) docker-push
	@echo "**** Built and pushed operator image ***** "
	$(MAKE) manifests-openshift
	@echo "**** Generated manifests required to deploy operator using OLM ***** "
	$(MAKE) bundle-build
	$(MAKE) bundle-push
	@echo "**** Built and pushed bundle image to repo ***** "
	$(MAKE) catalog-build
	$(MAKE) catalog-push
	@echo "**** Built and pushed catalog image to repo ***** "
	$(MAKE) index-build
	$(MAKE) index-push
	@echo "**** Built and pushed index image to repo ***** "
	oc apply -f artifacts/openshift/crd.yaml
	oc apply -f artifacts/openshift/rbac.yaml
	@echo "**** Applied CRD and RBAC manifests ***** "
	oc adm policy add-scc-to-user privileged -z vdo-controller-manager -n vmware-system-vdo
	oc apply -f config/openshift/misc/operator-group.yaml
	oc apply -f config/openshift/misc/catalog_source.yaml
	oc apply -f artifacts/openshift/vsphere-kubernetes-drivers-operator.clusterserviceversion.yaml
	@echo "**** Applied misc and CSV manifests ***** "

deploy: manifests kustomize deploy-local-kind ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KIND) get kubeconfig > kind-kubeconfig
	kubectl --kubeconfig ./kind-kubeconfig apply -f $(ARTIFACTS_DIR)/vanilla/vdo-spec.yaml

# build and deploy container in k8s cluster
# this target expects the following environment variables to be set
# K8S_MASTER_IP
# K8S_MASTER_SSH_USER
# K8S_MASTER_SSH_PWD
.PHONY: deploy-k8s-cluster
deploy-k8s-cluster: manifests kustomize build ## Build manager and Deploy the deployment to kind cluster.
	mkdir -p export
	docker build -t ${IMG} ${DOCKER_IMAGE_PROXY} --output type=tar,dest=export/vdo.tar .
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	./hack/deploy-vdo-cluster.sh ${IMG}

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# build and deploy container in kind cluster
.PHONY: deploy-local-kind
deploy-local-kind: kind-cluster build docker-build  ## Build manager and Deploy the deployment to kind cluster.
	@echo "Kubeconfig file: ${KUBECONFIG}"
	$(KIND) load docker-image $(IMG) --name ${KIND_CLUSTER_NAME} -v 3

# Create a kind cluster
.PHONY: kind-cluster
kind-cluster: kind ## Create a kind cluster if one does not exist
	@if $(KIND) get clusters | grep -qE "^${KIND_CLUSTER_NAME}$$"  >& /dev/null ; then \
		echo "Using existing kind cluster ${KIND_CLUSTER_NAME}" ; \
	else \
		echo "Creating cluster with kind..." ; \
		$(KIND) create cluster --name ${KIND_CLUSTER_NAME} ${LOCAL_KIND_NODE_IMAGE} -v 3  ; \
	fi

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	$(MAKE) add-bundle-image-artifacts
	$(MAKE) add-bundle-image-labels
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: bundle ## Build the bundle image for OLM
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) ${DOCKER_IMAGE_PROXY}.

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

add-bundle-image-artifacts: ## Add the Required labels for bundle image
	echo "COPY artifacts /manifests/" >> bundle.Dockerfile

add-bundle-image-labels: ## Add the Required labels for bundle image
	echo "LABEL com.redhat.openshift.versions=\"v4.8\"" >> bundle.Dockerfile
	echo "LABEL com.redhat.delivery.operator.bundle=true" >> bundle.Dockerfile
	echo "LABEL com.redhat.deliver.backport=false" >> bundle.Dockerfile

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.3/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

INDEX_IMG ?= $(IMAGE_TAG_BASE)-index:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image for OLM
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

.PHONY: index-build
index-build: opm ## Build a index image for OLM
	$(OPM) index add --container-tool docker --mode semver --tag $(INDEX_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

.PHONY: index-push
index-push: ## Push a index image.
	$(MAKE) docker-push IMG=$(INDEX_IMG)

## --------------------------------------
## Linting and fixing linter errors
## --------------------------------------

.PHONY: lint
lint: ## Run all the lint targets
	$(MAKE) lint-go-full
	#$(MAKE) lint-markdown

GOLANGCI_LINT_FLAGS ?= --fast=true --skip-dirs=docs
.PHONY: lint-go
lint-go: golangci_lint ## Lint codebase
	$(GOLANGCI_LINT) run -v $(GOLANGCI_LINT_FLAGS)

.PHONY: lint-go-full
lint-go-full: GOLANGCI_LINT_FLAGS = --fast=false --timeout ${GOLANGCI_LINT_TIMEOUT}
lint-go-full: lint-go ## Run slower linters to detect possible issues

.PHONY: lint-markdown
lint-markdown: ## Lint the project's markdown
	docker run --rm -v "$$(pwd)":/build$(DOCKER_VOL_OPTS) gcr.io/cluster-api-provider-vsphere/extra/mdlint:0.23.2 -- /md/lint -i vendor -i docs/ .

.PHONY: vdoctl-docgen
vdoctl-docgen: build-vdoctl
	@mkdir -p "docs/vdoctl"
	-rm -f docs/vdoctl/*.md
	bin/vdoctl generate-doc 'docs/vdoctl'

.PHONY: fix
fix: GOLANGCI_LINT_FLAGS = --fast=false --fix
fix: lint-go ## Tries to fix errors reported by lint-go-full target

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
golangci_lint: ## Download golangci-lint locally if necessary.
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint)

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

KIND = $(shell pwd)/bin/kind
kind:
	$(call go-get-tool,$(KIND),sigs.k8s.io/kind@v0.11.1)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

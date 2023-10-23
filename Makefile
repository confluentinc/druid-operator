# IMG TAG
IMG_TAG ?= "latest"
TEST_IMG_TAG ?= "test"
# Image URL to use all building/pushing image targets
IMG ?= "datainfrahq/druid-operator"
# Local Image URL to be pushed to kind registery
IMG_KIND ?= "localhost:5001/druid-operator"
# NAMESPACE for druid operator e2e
NAMESPACE_DRUID_OPERATOR ?= "druid-operator"
# NAMESPACE for zk operator e2e
NAMESPACE_ZK_OPERATOR ?= "zk-operator"
# NAMESPACE for zk operator e2e
NAMESPACE_MINIO_OPERATOR ?= "minio-operator"
# NAMESPACE for druid app e2e
NAMESPACE_DRUID ?= "druid"

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.26.0

GO_FIPS_ENV_VARS ?=
GOEXPERIMENTS ?=

#FIPS
# The build options that need to be passed to compile in boring-crypto includes GOOS=linux. This causes compilation errors when building the binary on Macbooks. Hence making it conditional to enable boring-crypto only when compiled in linux env (ex: our CI).
HOST_OS := $(shell uname | tr A-Z a-z)
ARCH := $(shell uname -m)
ifneq ($(HOST_OS),darwin)
ifneq (,$(filter $(ARCH),amd64 x86_64))
GO_FIPS_ENV_VARS = CGO_ENABLED=1 GOOS=linux
GO_EXPERIMENTS += boringcrypto
endif
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build test lint template docker-build

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

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(if $(GO_EXPERIMENTS),GOEXPERIMENT=$(subst $(_space),$(_comma),$(GO_EXPERIMENTS))) \
	$(GO_FIPS_ENV_VARS) $(CONTROLLER_GEN) rbac:roleName=manager-role crd:maxDescLen=0 webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(GO_FIPS_ENV_VARS) $(CONTROLLER_GEN) crd:generateEmbeddedObjectMeta=true paths="./..." output:crd:artifacts:config=deploy/crds
	$(GO_FIPS_ENV_VARS) $(CONTROLLER_GEN) crd:generateEmbeddedObjectMeta=true paths="./..." output:crd:artifacts:config=chart/templates/crds/

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(if $(GO_EXPERIMENTS),GOEXPERIMENT=$(subst $(_space),$(_comma),$(GO_EXPERIMENTS))) \
	$(GO_FIPS_ENV_VARS) go test ./... -coverprofile cover.out

.PHONY: e2e
e2e: ## Runs e2e tests
	e2e/e2e.sh

.PHONY: kind
kind: ## Bootstrap Kind Locally
	sh e2e/kind.sh

## Make Docker build for kind registery
.PHONY: docker-build-local
docker-build-local: ## Build docker image with the manager.
	docker build -t ${IMG_KIND}:${IMG_TAG} .

## Make Docker push locally to kind registery
.PHONY: docker-push-local
docker-push-local: ## Build docker image with the manager.
	docker push ${IMG_KIND}:${IMG_TAG}

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	$(if $(GO_EXPERIMENTS),GOEXPERIMENT=$(subst $(_space),$(_comma),$(GO_EXPERIMENTS))) \
	$(GO_FIPS_ENV_VARS) go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	$(if $(GO_EXPERIMENTS),GOEXPERIMENT=$(subst $(_space),$(_comma),$(GO_EXPERIMENTS))) \
	$(GO_FIPS_ENV_VARS) go run ./main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG}:${IMG_TAG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}:${IMG_TAG}

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have enable BuildKit, More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: test ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG}:${IMG_TAG} -f Dockerfile.cross .
	- docker buildx rm project-v3-builder
	rm Dockerfile.cross

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}:${IMG_TAG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Helm
.PHONY: helm-generate
helm-generate: ## Generate the Helm chart directory
	$(KUSTOMIZE) build config/crd > chart/templates/crds/druid.apache.org_druids.yaml

.PHONY: helm-lint
helm-lint: ## Lint Helm chart.
	helm lint ./chart

.PHONY: helm-template
helm-template: ## Run Helm template.
	helm -n druid-operator-system template --create-namespace ${NAMESPACE_DRUID_OPERATOR} ./chart --debug

.PHONY: helm-install-druid-operator
helm-install-druid-operator: ## helm upgrade/install.
	helm upgrade --install \
	--namespace ${NAMESPACE_DRUID_OPERATOR} \
	--create-namespace \
	${NAMESPACE_DRUID_OPERATOR} chart/ \
	--set image.repository=${IMG_KIND} \
	--set image.tag=${IMG_TAG}

.PHONY: helm-minio-install
helm-minio-install: ## Helm deploy minio operator and minio
	helm repo add minio https://operator.min.io/
	helm repo update minio
	helm upgrade --install \
	--namespace ${NAMESPACE_MINIO_OPERATOR} \
	--create-namespace \
	 ${NAMESPACE_MINIO_OPERATOR} minio/operator \
	-f e2e/configs/minio-operator-override.yaml
	helm upgrade --install \
	--namespace ${NAMESPACE_DRUID} \
	--create-namespace \
  	${NAMESPACE_DRUID}-minio minio/tenant \
	-f e2e/configs/minio-tenant-override.yaml

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.11.2

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

## e2e deployment
.PHONY: e2e
e2e: 
	e2e/e2e.sh

## Build Kind
.PHONY: kind 
kind: ## Bootstrap Kind Locally
	sh e2e/kind.sh

## Make Docker build for kind registery
.PHONY: docker-build-local
docker-build-local: ## Build docker image with the manager.
	docker build -t ${IMG_KIND}:${IMG_TAG} .

## Make Docker push locally to kind registery
.PHONY: docker-push-local
docker-push-local: ## Build docker image with the manager.
	docker push ${IMG_KIND}:${IMG_TAG}

## Make Docker build for test image
.PHONY: docker-build-local-test
docker-build-local: ## Build docker image with the manager.
	docker build -t ${IMG_KIND}:${TEST_IMG_TAG} -f e2e/Dockerfile-testpod .

## Make Docker push  locally to kind registery
.PHONY: docker-push-local-test
docker-push-local: ## Build docker image with the manager.
	docker push ${IMG_KIND}:${TEST_IMG_TAG}

## Helm install to deploy the druid operator
.PHONY: helm-install-druid-operator
helm-install-druid-operator: ## helm upgrade/install
	helm upgrade --install \
	--namespace ${NAMESPACE_DRUID_OPERATOR} \
	--create-namespace \
	${NAMESPACE_DRUID_OPERATOR} chart/ \
	--set image.repository=${IMG_KIND} \
	--set image.tag=${IMG_TAG}

## Helm deploy minio operator and minio
.PHONY: helm-minio-install
helm-minio-install:
	helm repo add minio https://operator.min.io/
	helm repo update minio
	helm upgrade --install \
	--namespace ${NAMESPACE_MINIO_OPERATOR} \
	--create-namespace \
	 ${NAMESPACE_MINIO_OPERATOR} minio/operator \
	-f e2e/configs/minio-operator-override.yaml
	helm upgrade --install \
	--namespace ${NAMESPACE_DRUID} \
	--create-namespace \
  	${NAMESPACE_DRUID}-minio minio/tenant \
	-f e2e/configs/minio-tenant-override.yaml

## Run the test pod
.PHONY: deploy-testjob
deploy-testjob:
	kubectl create job wiki-test --image=${IMG_KIND}:${TEST_IMG_TAG}  -- sh /wikipedia-test.sh


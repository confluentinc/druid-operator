
# Image URL to use all building/pushing image targets
IMG ?= "druid-operator:latest"
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:maxDescLen=0,trivialVersions=true,generateEmbeddedObjectMeta=true"

GO_FIPS_ENV_VARS ?=
GOEXPERIMENTS ?=

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

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

all: manager

# Run tests
test: generate fmt vet manifests
	$(if $(GO_EXPERIMENTS),GOEXPERIMENT=$(subst $(_space),$(_comma),$(GO_EXPERIMENTS))) \
	$(GO_FIPS_ENV_VARS) go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	$(if $(GO_EXPERIMENTS),GOEXPERIMENT=$(subst $(_space),$(_comma),$(GO_EXPERIMENTS))) \
	$(GO_FIPS_ENV_VARS) go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	$(if $(GO_EXPERIMENTS),GOEXPERIMENT=$(subst $(_space),$(_comma),$(GO_EXPERIMENTS))) \
	$(GO_FIPS_ENV_VARS) go run ./main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f deploy/crds/druid.apache.org_druids.yaml

# Uninstall CRDs from a cluster
uninstall: manifests
	kubectl delete -f deploy/crds/druid.apache.org_druids.yaml

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f deploy/service_account.yaml
	kubectl apply -f deploy/role.yaml
	kubectl apply -f deploy/role_binding.yaml
	kustomize build deploy/ | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=deploy/crds/

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	$(if $(GO_EXPERIMENTS),GOEXPERIMENT=$(subst $(_space),$(_comma),$(GO_EXPERIMENTS))) \
	$(GO_FIPS_ENV_VARS) go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: generate manifests
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.10.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

init-ci:
	# Install kubebuilder
	curl -L -O "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_${KUBEBUILDER_VERSION}_linux_${OS_ARCH}.tar.gz"
	sudo mkdir -p /usr/local/kubebuilder && \
		sudo tar -zxvf kubebuilder_${KUBEBUILDER_VERSION}_linux_${OS_ARCH}.tar.gz --strip-components=1 -C /usr/local/kubebuilder/
	export PATH=$PATH:/usr/local/kubebuilder/bin

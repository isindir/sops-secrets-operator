GO := GOPROXY=https://proxy.golang.org go
SOPS_SEC_OPERATOR_VERSION := 0.2.2

# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_GEN_VERSION := "v0.4.1"
KUSTOMIZE_VERSION := "v4.1.2"

# Use existing cluster instead of starting processes
USE_EXISTING_CLUSTER ?= true
# Image URL to use all building/pushing image targets
IMG_NAME ?= isindir/sops-secrets-operator
IMG ?= ${IMG_NAME}:${SOPS_SEC_OPERATOR_VERSION}
IMG_LATEST ?= ${IMG_NAME}:latest
IMG_CACHE ?= ${IMG_NAME}:cache
BUILDX_PLATFORMS ?= linux/amd64,linux/arm64
# Produce CRDs that work back to Kubernetes 1.16
CRD_OPTIONS ?= crd:crdVersions=v1

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
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

clean: ## Cleans dependency directories.
	rm -fr $$( which controller-gen )
	rm -fr ./vendor

tidy: ## Fetches all go dependencies.
	$(GO) mod tidy
	$(GO) mod vendor

pre-commit: ## Update and runs pre-commit.
	pre-commit install
	pre-commit autoupdate
	pre-commit run -a

##@ Helm

package-helm: ## Repackages helm chart.
	@{ \
		( cd docs; \
			helm package ../chart/helm3/sops-secrets-operator ; \
			helm repo index . --url https://isindir.github.io/sops-secrets-operator ) ; \
	}

test-helm: ## Tests helm chart.
	@{ \
		$(MAKE) -C chart/helm3/sops-secrets-operator all ; \
	}

##@ Development

manifests: tidy controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen tidy ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	@echo
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

#test: controller-gen generate fmt vet manifests ## Run tests.
#	USE_EXISTING_CLUSTER=${USE_EXISTING_CLUSTER} go test ./... -coverprofile cover.out

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-login: ## Performs logging to dockerhub using DOCKERHUB_USERNAME and DOCKERHUB_PASS environment variables.
	echo "${DOCKERHUB_PASS}" | base64 -d | docker login -u "${DOCKERHUB_USERNAME}" --password-stdin
	docker buildx create --name mybuilder --use

docker-cross-build: ## Build multi-arch docker image.
	docker buildx build --quiet --cache-from=${IMG_CACHE} --cache-to=${IMG_CACHE} --platform ${BUILDX_PLATFORMS} -t ${IMG} .

docker-build-dont-test: generate fmt vet manifests ## Build the docker image without running tests.
	docker build . -t ${IMG}
	docker tag ${IMG} ${IMG_LATEST}

docker-build: test ## Build docker image with the manager.
	docker build . -t ${IMG}
	docker tag ${IMG} ${IMG_LATEST}

docker-push: ## Push docker image with the manager.
	docker push ${IMG}
	docker push ${IMG_LATEST}

##@ Deployment

# TODO: re-tag with crane image to latest
release: controller-gen generate fmt vet manifests ## Creates github release and pushes docker image to dockerhub.
	@{ \
		set +e ; \
		git tag "${SOPS_SEC_OPERATOR_VERSION}" ; \
		tagResult=$$? ; \
		if [[ $$tagResult -ne 0 ]]; then \
			echo "Release '${SOPS_SEC_OPERATOR_VERSION}' exists - skipping" ; \
		else \
			set -e ; \
			git-chglog "${SOPS_SEC_OPERATOR_VERSION}" > chglog.tmp ; \
			hub release create -F chglog.tmp "${SOPS_SEC_OPERATOR_VERSION}" ; \
			docker buildx build --push --quiet --cache-from=${IMG_CACHE} --cache-to=${IMG_CACHE} --platform ${BUILDX_PLATFORMS} -t ${IMG} . ; \
		fi ; \
	}

inspect: ## Inspects remote docker 'image tag' - target fails if it does find existing tag.
	@echo "Inspect remote image"
	@! DOCKER_CLI_EXPERIMENTAL="enabled" docker manifest inspect ${IMG} >/dev/null \
		|| { echo "Image already exists"; exit 1; }

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION})

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@${KUSTOMIZE_VERSION})

# go-get-tool will 'go get' any package $2 and install it to $1
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

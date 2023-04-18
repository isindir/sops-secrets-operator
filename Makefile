# UPDATE_HERE
GO := GOPROXY=https://proxy.golang.org go
SOPS_SEC_OPERATOR_VERSION := 0.8.2

# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION ?= v0.11.3
#CONTROLLER_GEN_VERSION := "v0.11.3"
# https://github.com/kubernetes-sigs/controller-runtime/releases
CONTROLLER_RUNTIME_VERSION := "v0.14.6"
# https://github.com/kubernetes-sigs/kustomize/releases
KUSTOMIZE_VERSION := "v5.0.1"
# use `setup-envtest list` to obtain the list of available versions
# until fixed, can't use newer version, see:
#   https://github.com/kubernetes-sigs/controller-runtime/issues/1571
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
# https://storage.googleapis.com/kubebuilder-tools
ENVTEST_K8S_VERSION := "1.26.1"

# Use existing cluster instead of starting processes
USE_EXISTING_CLUSTER ?= true
# Image URL to use all building/pushing image targets
IMG_NAME ?= isindir/sops-secrets-operator
IMG ?= ${IMG_NAME}:${SOPS_SEC_OPERATOR_VERSION}
IMG_LATEST ?= ${IMG_NAME}:latest
IMG_CACHE ?= ${IMG_NAME}:cache
BUILDX_PLATFORMS ?= linux/amd64,linux/arm64
# Produce CRDs are backwards compatible up to Kubernetes 1.16
CRD_OPTIONS ?= crd:crdVersions=v1

TMP_COVER_FILE="cover.out"
TMP_COVER_HTML_FILE="index.html"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell $(GO) env GOBIN))
GOBIN=$(shell $(GO) env GOPATH)/bin
else
GOBIN=$(shell $(GO) env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest' in the test target.
# for more information about setup-envtest refer to
#     https://github.com/kubernetes-sigs/controller-runtime/tree/v0.9.6/tools/setup-envtest
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: clean build

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

.PHONY: clean
clean: ## Cleans dependency directories.
	rm -fr ./vendor
	rm -fr ./testbin
	rm -fr ./bin
	rm -f $(TMP_COVER_HTML_FILE) $(TMP_COVER_FILE)

.PHONY: tidy
tidy: ## Fetches all go dependencies.
	$(GO) mod tidy
	$(GO) mod vendor

.PHONY: pre-commit
pre-commit: ## Update and runs pre-commit.
	pre-commit install
	pre-commit autoupdate
	pre-commit run -a

##@ Helm

.PHONY: package-helm
package-helm: ## Repackages helm chart.
	@{ \
		( cd docs; \
			helm package ../chart/helm3/sops-secrets-operator ; \
			helm repo index . --url https://isindir.github.io/sops-secrets-operator ) ; \
	}

.PHONY: test-helm
test-helm: ## Tests helm chart.
	@{ \
		$(MAKE) -C chart/helm3/sops-secrets-operator all ; \
	}

##@ Development

.PHONY: update-here
update-here: ## Helper target to start editing all occurances with UPDATE_HERE.
	@echo "Update following files for release:"
	@grep --color -nHR UPDATE_HERE .

.PHONY: manifests
manifests: tidy controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: manifests ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	@echo
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	$(GO) fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	$(GO) vet ./...

.PHONY: test
test: clean generate fmt vet envtest ## Run tests.
	SOPS_AGE_RECIPIENTS="age1pnmp2nq5qx9z4lpmachyn2ld07xjumn98hpeq77e4glddu96zvms9nn7c8" SOPS_AGE_KEY_FILE="${PWD}/config/age-test-key/key-file.txt" KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path --force)" $(GO) test ./... -coverpkg=./internal/controllers/... -coverprofile=$(TMP_COVER_FILE)

cover: test ## Run tests with coverage.
	$(GO) tool cover -func=$(TMP_COVER_FILE)
	$(GO) tool cover -o $(TMP_COVER_HTML_FILE) -html=$(TMP_COVER_FILE)

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	$(GO) build -o bin/manager cmd/main.go

.PHONY: run
run: generate fmt vet ## Run a controller from your host.
	$(GO) run ./cmd/main.go

docker-login: ## Performs logging to dockerhub using DOCKERHUB_USERNAME and DOCKERHUB_PASS environment variables.
	echo "${DOCKERHUB_PASS}" | base64 -d | docker login -u "${DOCKERHUB_USERNAME}" --password-stdin
	docker buildx create --name mybuilder --use

docker-cross-build: ## Build multi-arch docker image.
	docker buildx build --cache-from=${IMG_CACHE} --cache-to=${IMG_CACHE} --platform ${BUILDX_PLATFORMS} -t ${IMG} .

docker-build-dont-test: generate fmt vet ## Build the docker image without running tests.
	docker build -t ${IMG} .
	docker tag ${IMG} ${IMG_LATEST}

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .
	docker tag ${IMG} ${IMG_LATEST}

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}
	docker push ${IMG_LATEST}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

# TODO: re-tag with crane image to latest
#       https://michaelsauter.github.io/crane/docs.html
.PHONY: release
release: generate fmt vet ## Creates github release and pushes docker image to dockerhub.
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

.PHONY: inspect
inspect: ## Inspects remote docker 'image tag' - target fails if it does find existing tag.
	@echo "Inspect remote image"
	@! DOCKER_CLI_EXPERIMENTAL="enabled" docker manifest inspect ${IMG} >/dev/null \
		|| { echo "Image already exists"; exit 1; }

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Misc

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)
#controller-gen: ## Download controller-gen locally if necessary.
#	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION})

KUSTOMIZE ?= $(LOCALBIN)/kustomize
KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) --output install_kustomize.sh && bash install_kustomize.sh $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); rm install_kustomize.sh; }
#.PHONY: kustomize
#kustomize: ## Download kustomize locally if necessary.
#	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5@${KUSTOMIZE_VERSION})

ENVTEST ?= $(LOCALBIN)/setup-envtest
.PHONY: envtest
$(ENVTEST): $(LOCALBIN)
envtest: ## Download setup-envtest locally if necessary.
	echo $(LOCALBIN)/setup-envtest
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
#	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

.PHONY: envtest-list
envtest-list: envtest ## List of the available setup-envtest versions.
	$(ENVTEST) list

#GINKGO = $(shell pwd)/ginkgo
#setup-ginkgo: ## Download ginkgo locally
#	$(call go-install-tool,$(GINKGO),github.com/onsi/ginkgo/ginkgo)

## go-install-tool will 'go install' any package $2 and install it to $1
#PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
#define go-install-tool
#@[ -f $(1) ] || { \
#set -e ;\
#TMP_DIR=$$(mktemp -d) ;\
#cd $$TMP_DIR ;\
#$(GO) version ;\
#$(GO) mod init tmp ;\
#echo "Downloading $(2)" ;\
#GOBIN=$(PROJECT_DIR)/bin $(GO) install $(2) ;\
#rm -rf $$TMP_DIR ;\
#}
#endef
#
## go-get-tool will 'go get' any package $2 and install it to $1.
#PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
#define go-get-tool
#@[ -f $(1) ] || { \
#set -e ;\
#TMP_DIR=$$(mktemp -d) ;\
#cd $$TMP_DIR ;\
#go mod init tmp ;\
#echo "Downloading $(2)" ;\
#GOBIN=$(PROJECT_DIR)/bin $(GO) get $(2) ;\
#rm -rf $$TMP_DIR ;\
#}
#endef

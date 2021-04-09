SHELL := /bin/bash
GO := GO15VENDOREXPERIMENT=1 GO111MODULE=on GOPROXY=https://proxy.golang.org go
SOPS_SEC_OPERATOR_VERSION := 0.1.13

# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION := "v0.3.0"

# Use existing cluster instead of starting processes
USE_EXISTING_CLUSTER ?= true
# Image URL to use all building/pushing image targets
IMG_NAME ?= isindir/sops-secrets-operator
IMG ?= ${IMG_NAME}:${SOPS_SEC_OPERATOR_VERSION}
IMG_LATEST ?= ${IMG_NAME}:latest
IMG_CHACHE ?= ${IMG_NAME}:cache
BUILDX_PLATFORMS ?= linux/amd64,linux/arm64
# Produce CRDs that work back to Kubernetes 1.16
CRD_OPTIONS ?= crd:crdVersions=v1

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

## clean: cleans dependency directories
clean:
	rm -fr $$( which controller-gen )
	rm -fr ./vendor

## package-helm: repackage helm charts
package-helm:
	@{ \
		( cd docs; \
			helm package ../chart/helm3/sops-secrets-operator ; \
			helm repo index . --url https://isindir.github.io/sops-secrets-operator ) ; \
	}

## test-helm: test helm charts
test-helm:
	@{ \
		$(MAKE) -C chart/helm3/sops-secrets-operator all ; \
	}

## test: Run tests
test: generate fmt vet manifests
	USE_EXISTING_CLUSTER=${USE_EXISTING_CLUSTER} go test ./... -coverprofile cover.out

## manager: Build manager binary
manager: generate fmt vet manifests
	go build -o bin/manager main.go

## run: Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

## install: Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

## uninstall: Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

## deploy: Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

## manifests: Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

## fmt: Run go fmt against code
fmt:
	go fmt ./...

## vet: Run go vet against code
vet:
	go vet ./...

## tidy: Fetch all needed go packages
tidy:
	$(GO) mod tidy
	$(GO) mod vendor

## generate: Generate code
generate: controller-gen tidy
	@echo
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

## docker-login: performs logging to dockerhub using DOCKERHUB_USERNAME and DOCKERHUB_PASS environment variables
docker-login:
	echo "${DOCKERHUB_PASS}" | base64 -d | docker login -u "${DOCKERHUB_USERNAME}" --password-stdin
	docker buildx create --name mybuilder --use

## docker-cross-build: Build multi-arch docker image
docker-cross-build:
	docker buildx build --quiet --cache-from=${IMG_CHACHE} --cache-to=${IMG_CHACHE} --platform ${BUILDX_PLATFORMS} -t ${IMG} .

## docker-cross-build-dont-test: Build the docker image without running tests
docker-cross-build-dont-test: generate fmt vet manifests
	docker buildx build --push --quiet --cache-from=${IMG_CHACHE} --cache-to=${IMG_CHACHE} --platform ${BUILDX_PLATFORMS} -t ${IMG} .

## docker-build: Build the docker image
docker-build: test
	docker build . -t ${IMG}
	docker tag ${IMG} ${IMG_LATEST}

## docker-build-dont-test: Build the docker image without running tests
docker-build-dont-test: generate fmt vet manifests
	docker build . -t ${IMG}
	docker tag ${IMG} ${IMG_LATEST}

## docker-push: Push the docker image
docker-push:
	docker push ${IMG}
	docker push ${IMG_LATEST}

## release: creates github release and pushes docker image to dockerhub
release: docker-cross-build-dont-test
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
			echo "${DOCKERHUB_PASS}" | base64 -d | docker login -u "${DOCKERHUB_USERNAME}" --password-stdin ; \
			# TODO: re-tag with crane image to latest
		fi ; \
	}

## inspect: inspects remote docker 'image tag' - target fails if it does
inspect:
	@echo "Inspect remote image"
	@! DOCKER_CLI_EXPERIMENTAL="enabled" docker manifest inspect ${IMG} >/dev/null \
		|| { echo "Image already exists"; exit 1; }

## controller-gen: find or download controller-gen - download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_TOOLS_VERSION} ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

## pre-commit: update pre-commit
pre-commit:
	pre-commit install
	pre-commit autoupdate
	pre-commit run -a

.PHONY: help
## help: prints this help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

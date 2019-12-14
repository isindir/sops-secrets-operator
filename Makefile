SHELL := /bin/bash
GO 		:= GO15VENDOREXPERIMENT=1 GO111MODULE=on GOPROXY=https://proxy.golang.org go

IMAGE_NAME?="isindir/sops-secrets-operator"
SDK_IMAGE_NAME?="isindir/sdk"
VERSION?=$(shell awk 'BEGIN { FS=" = " } $$0~/Version = / \
				 { gsub(/"/, ""); print $$2; }' version/version.go)
BUILD:=`git rev-parse HEAD`
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

all: clean gen mod fmt check test inspect build

.PHONY: repo-tag
## repo-tag: tags git repository with latest version
repo-tag:
	@git tag -a ${VERSION} -m "sops-secrets-operator ${VERSION}"

.PHONY: release
## release: builds operator docker image and pushes it to docker repository
release: build push

.PHONY: mod
## mod: fetches dependencies
mod:
	@echo "Go Mod Vendor"
	$(GO) mod tidy
	$(GO) mod vendor
	@echo

.PHONY: echo
## echo: prints image name and version of the operator
echo:
	@echo "${IMAGE_NAME}:${VERSION}"
	@echo "${BUILD}"

.PHONY: clean
## clean: removes build artifacts from source code
clean:
	@echo "Cleaning"
	@rm -fr build/_output
	@rm -fr vendor
	@echo

.PHONY: inspect
## inspect: inspects remote docker 'image tag' - target fails if it does
inspect:
	@echo "Inspect remote image"
	@! DOCKER_CLI_EXPERIMENTAL="enabled" docker manifest inspect ${IMAGE_NAME}:${VERSION} >/dev/null \
		|| { echo "Image already exists"; exit 1; }

.PHONY: build
## build: builds operator docker image
build:
	@echo "Building"
	@operator-sdk build "${IMAGE_NAME}:${VERSION}"
	@docker tag "${IMAGE_NAME}:${VERSION}" "${IMAGE_NAME}:latest"
	@echo

.PHONY: build/sdk
## build/sdk: builds sdk docker image (not used)
build/sdk:
	@echo "Building sdk image"
	@docker build .circleci -t "${SDK_IMAGE_NAME}"
	@echo

.PHONY: push
## push: pushes operator docker image to repository
push:
	@echo "Pushing"
	@docker push "${IMAGE_NAME}:latest"
	@docker push "${IMAGE_NAME}:${VERSION}"
	@echo

.PHONY: push/sdk
## push/sdk: pushes sdk docker image to repository
push/sdk:
	@echo "Pushing"
	@docker push "${SDK_IMAGE_NAME}"
	@echo

.PHONY: gen
## gen: generates automated code
gen:
	@echo "Generating"
	@operator-sdk generate k8s
	@echo

.PHONY: fmt
## fmt: formats go code
fmt:
	@echo "Formatting"
	@gofmt -l -w $(SRC)
	@echo

.PHONY: check
## check: runs linting
check:
	@echo "Linting"
	@for d in $$(go list ./... | grep -v /vendor/); do golint $${d}; done
	@echo

.PHONY: test/unit
## test/unit: runs unit tests
test/unit:
	@echo "Running unit tests"
	$(GO) test -count=1 -short ./pkg/controller/...
	@echo

.PHONY: test/e2e
## test/e2e: runs e2e tests
test/e2e:
	@echo "Running e2e tests"
	@operator-sdk test local ./test/e2e --up-local --namespace sops
	@echo

.PHONY: test/operator
## test/operator: runs following make targets - fmt check test/unit test/e2e
test/operator: fmt check test/unit test/e2e

.PHONY: test
## test: placeholder to run unit and e2e tests
test: test/operator
	@echo "TODO: Write some usefull unit and e2e tests"
	@echo

.PHONY: run/local
## run/local: runs operator in local mode
run/local:
	@OPERATOR_NAME=sops-secrets-operator operator-sdk up local --namespace=sops

.PHONY: run/sdk
## run/sdk: runs sdk docker image
run/sdk:
	@docker run -v ~/.gitconfig:/root/.gitconfig \
		-v ~/.gnupg:/root/.gnupg \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v ${PWD}:/go/src/github.com/isindir/sops-secrets-operator \
		-ti "${SDK_IMAGE_NAME}" bash

.PHONY: help
## help: prints this help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

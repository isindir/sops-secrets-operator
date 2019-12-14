SHELL := /bin/bash
GO 		:= GO15VENDOREXPERIMENT=1 GO111MODULE=on GOPROXY=https://proxy.golang.org go

.PHONY: repo-tag release mod echo clean build push gen fmt check test/unit test/e2e test/operator test local/run

IMAGE_NAME?="isindir/sops-secrets-operator"
SDK_IMAGE_NAME?="isindir/sdk"
VERSION?=$(shell awk 'BEGIN { FS=" = " } $$0~/Version = / \
				 { gsub(/"/, ""); print $$2; }' version/version.go)
BUILD:=`git rev-parse HEAD`
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

#LDFLAGS=-ldflags "-X=version.Version=$(VERSION) -X=version.Build=$(BUILD)"

all: clean gen mod fmt check test inspect build

repo-tag:
	@git tag -a ${VERSION} -m "sops-secrets-operator ${VERSION}"

release: build push

mod:
	@echo "Go Mod Vendor"
	$(GO) mod tidy
	$(GO) mod vendor
	@echo

echo:
	@echo "${IMAGE_NAME}:${VERSION}"
	@echo "${BUILD}"
	@#echo "${LDFLAGS}"

clean:
	@echo "Cleaning"
	@rm -fr build/_output
	@echo

inspect:
	@echo "Inspect remote image"
	@! DOCKER_CLI_EXPERIMENTAL="enabled" docker manifest inspect ${IMAGE_NAME}:${VERSION} >/dev/null \
		|| { echo "Image already exists"; exit 1; }

build:
	@echo "Building"
	@operator-sdk build "${IMAGE_NAME}:${VERSION}"
	@docker tag "${IMAGE_NAME}:${VERSION}" "${IMAGE_NAME}:latest"
	@echo

build/sdk:
	@echo "Building sdk image"
	@docker build .circleci -t "${SDK_IMAGE_NAME}"
	@echo

push:
	@echo "Pushing"
	@docker push "${IMAGE_NAME}:latest"
	@docker push "${IMAGE_NAME}:${VERSION}"
	@echo

push/sdk:
	@echo "Pushing"
	@docker push "${SDK_IMAGE_NAME}"
	@echo

gen:
	@echo "Generating"
	@operator-sdk generate k8s
	@echo

fmt:
	@echo "Formatting"
	@gofmt -l -w $(SRC)
	@echo

check:
	@echo "Linting"
	@for d in $$(go list ./... | grep -v /vendor/); do golint $${d}; done
	@echo
	@#go vet ${SRC}

test/unit:
	@echo "Running unit tests"
	$(GO) test -count=1 -short ./pkg/controller/...
	@echo

test/e2e:
	@echo "Running e2e tests"
	@operator-sdk test local ./test/e2e --up-local --namespace sops
	@echo

test/operator: fmt check test/unit test/e2e

test: test/operator
	@echo "TODO: Write some usefull unit and e2e tests"
	@echo

run/local:
	@OPERATOR_NAME=sops-secrets-operator operator-sdk up local --namespace=sops

run/sdk:
	@docker run -v ~/.gitconfig:/root/.gitconfig \
		-v ~/.gnupg:/root/.gnupg \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v ${PWD}:/go/src/github.com/isindir/sops-secrets-operator \
		-ti "${SDK_IMAGE_NAME}" bash

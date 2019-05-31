SHELL := /bin/bash

.PHONY: repo-tag release mod echo clean build push gen fmt check test local/run

IMAGE_NAME?="isindir/sops-secrets-operator"
VERSION?=$(shell awk 'BEGIN { FS=" = " } $$0~/Version = / \
				 { gsub(/"/, ""); print $$2; }' version/version.go)
BUILD:=`git rev-parse HEAD`
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

#LDFLAGS=-ldflags "-X=version.Version=$(VERSION) -X=version.Build=$(BUILD)"

all: clean gen mod fmt check test build

repo-tag:
	@git tag -a ${VERSION} -m "sops-secrets-operator ${VERSION}"

release: build push

mod:
	@echo "Go Mod Vendor"
	@go mod vendor
	@echo

echo:
	@echo "${IMAGE_NAME}:${VERSION}"
	@echo "${BUILD}"
	@#echo "${LDFLAGS}"

clean:
	@echo "Cleaning"
	@rm -fr build/_output
	@echo

build:
	@echo "Building"
	@operator-sdk build "${IMAGE_NAME}:${VERSION}"
	@docker tag "${IMAGE_NAME}:${VERSION}" "${IMAGE_NAME}:latest"
	@echo

push:
	@echo "Pushing"
	@docker push "${IMAGE_NAME}:latest"
	@docker push "${IMAGE_NAME}:${VERSION}"
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

test:
	@echo "TODO: Testing"
	@echo

local/run:
	@OPERATOR_NAME=sops-secrets-operator operator-sdk up local --namespace=sops

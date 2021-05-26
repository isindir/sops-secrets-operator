.PHONY: all dep list test

CHART_NAME?=$(shell cat Chart.yaml | awk 'BEGIN { FS=": " } $$0~/^name:/ { gsub(/['\'',]/, ""); print $$2; }')
VERSION_TAG?=$(shell cat Chart.yaml | awk 'BEGIN { FS=": " } $$0~/^version/ { gsub(/['\'',]/, ""); print $$2; }')

SHELL=/bin/bash

##@ General

all: echo lint test validate ## run all test targets

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

versions: ## shows currently installed tool versions
	helm version ; echo
	helm plugin list | grep unittest ; echo
	@echo '--------------------'
	@asdf current kubebuilder
	@asdf current golang
	@asdf current sops
	@asdf current kustomize
	@asdf current k3d
	@asdf current kubectl
	@asdf current helm
	@asdf current kubeval
	@echo '--------------------'

echo: ## prints chart information
	@echo '-=-=-=-=-=-=-=-=-=-=- "${CHART_NAME}" version: "${VERSION_TAG}" -=-=-=-=-=-=-=-=-=-=-'

test: ## runs unittests
	helm unittest --helm3 --color .
	@echo '--------------------'

lint: ## runs helm chart linting
	helm lint .
	@echo '--------------------'

validate: ## validates rendered chart templates using 'kubeval'
	helm template . --set securityContextenabled=true | kubeval --ignore-missing-schemas --force-color -
	@echo '--------------------'

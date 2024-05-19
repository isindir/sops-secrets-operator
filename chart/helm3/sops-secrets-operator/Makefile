.PHONY: all dep list test

CHART_NAME?=$(shell cat Chart.yaml | awk 'BEGIN { FS=": " } $$0~/^name:/ { gsub(/['\'',]/, ""); print $$2; }')
VERSION_TAG?=$(shell cat Chart.yaml | awk 'BEGIN { FS=": " } $$0~/^version/ { gsub(/['\'',]/, ""); print $$2; }')

# UPDATE_HERE
K8S_VERSION := "1.30.1"

SHELL=/bin/bash

##@ General

.PHONY: all
all: echo lint test validate ## run all test targets

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: versions
versions: ## shows currently installed tool versions
	helm version ; echo
	helm plugin list | grep unittest ; echo
	@echo '--------------------'
	@asdf current 2>/dev/null | grep sops-secrets-operator
	@echo '--------------------'

.PHONY: echo
echo: ## prints chart information
	@echo '-=-=-=-=-=-=-=-=-=-=- "${CHART_NAME}" version: "${VERSION_TAG}" -=-=-=-=-=-=-=-=-=-=-'

.PHONY: test
test: ## runs unittests
	helm unittest --color .
	@echo '--------------------'

.PHONY: lint
lint: ## runs helm chart linting
	helm lint .
	@echo '--------------------'

.PHONY: validate
validate: ## validates rendered chart templates using 'kubeconform'
	helm template . \
		--set securityContextenabled=true \
		--set metrics.enabled=true \
		--set gcp.enabled=true \
		--set azure.enabled=true \
		| kubeconform  -summary \
				-verbose \
				-strict \
				-output pretty \
				-schema-location https://raw.githubusercontent.com/Onemind-Services-LLC/kubernetes-json-schema/master/schema \
				-schema-location https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/monitoring.coreos.com/servicemonitor_v1.json \
				-kubernetes-version $(K8S_VERSION) -
	@echo '--------------------'

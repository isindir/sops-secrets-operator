.PHONY: all dep list test

CHART_NAME?=$(shell cat Chart.yaml | awk 'BEGIN { FS=": " } $$0~/^name:/ { gsub(/['\'',]/, ""); print $$2; }')
VERSION_TAG?=$(shell cat Chart.yaml | awk 'BEGIN { FS=": " } $$0~/^version/ { gsub(/['\'',]/, ""); print $$2; }')

# UPDATE_HERE
K8S_VERSION := "1.29.1"

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
	@asdf current kubebuilder
	@asdf current golang
	@asdf current sops
	@asdf current kustomize
	@asdf current k3d
	@asdf current kubectl
	@asdf current helm
	@asdf current kubeval
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
validate: ## validates rendered chart templates using 'kubeval'
	helm template . --set securityContextenabled=true \
		| kubeval --force-color \
				--strict \
				--schema-location https://raw.githubusercontent.com/Onemind-Services-LLC/kubernetes-json-schema/master/schema \
				--kubernetes-version $(K8S_VERSION) -
	@echo '--------------------'

.PHONY: conform
conform: ## validates rendered chart templates using 'kubeconform'
	helm template . --set securityContextenabled=true \
		| kubeconform  -summary \
				-verbose \
				-strict \
				-schema-location https://raw.githubusercontent.com/Onemind-Services-LLC/kubernetes-json-schema/master/schema \
				-kubernetes-version $(K8S_VERSION) -
	@echo '--------------------'

#Usage: /Users/erikszelenka/.asdf/installs/kubeconform/0.6.4/bin/kubeconform [OPTION]... [FILE OR FOLDER]...
#  -cache string
#        cache schemas downloaded via HTTP to this folder
#  -debug
#        print debug information
#  -exit-on-error
#        immediately stop execution when the first error is encountered
#  -h    show help information
#  -ignore-filename-pattern value
#        regular expression specifying paths to ignore (can be specified multiple times)
#  -ignore-missing-schemas
#        skip files with missing schemas instead of failing
#  -insecure-skip-tls-verify
#        disable verification of the server's SSL certificate. This will make your HTTPS connections insecure
#  -kubernetes-version string
#        version of Kubernetes to validate against, e.g.: 1.18.0 (default "master")
#  -n int
#        number of goroutines to run concurrently (default 4)
#  -output string
#        output format - json, junit, pretty, tap, text (default "text")
#  -reject string
#        comma-separated list of kinds or GVKs to reject
#  -schema-location value
#        override schemas location search path (can be specified multiple times)
#  -skip string
#        comma-separated list of kinds or GVKs to ignore
#  -strict
#        disallow additional properties not in schema or duplicated keys
#  -summary
#        print a summary at the end (ignored for junit output)
#  -v    show version information
#  -verbose
#        print results for all resources (ignored for tap and junit outp
#

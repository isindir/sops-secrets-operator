.PHONY: all dep list test

CHART_NAME?=$(shell cat Chart.yaml | awk 'BEGIN { FS=": " } $$0~/^name:/ { gsub(/['\'',]/, ""); print $$2; }')
VERSION_TAG?=$(shell cat Chart.yaml | awk 'BEGIN { FS=": " } $$0~/^version/ { gsub(/['\'',]/, ""); print $$2; }')

SHELL=/bin/bash

## all: run all test targets
all: echo lint test validate

## versions: show currently installed tool versions
versions:
	helm version ; echo
	helm plugin list | grep unittest ; echo
	kubeval --version
	@echo '--------------------'

## echo: print chart information
echo:
	@echo '-=-=-=-=-=-=-=-=-=-=- "${CHART_NAME}" version: "${VERSION_TAG}" -=-=-=-=-=-=-=-=-=-=-'

## test: run unittests
test:
	helm unittest --color .
	@echo '--------------------'

## lint: lint helm chart
lint:
	helm lint .
	@echo '--------------------'

## validate: validate rendered chart templates using 'kubeval'
validate:
	helm template . | kubeval --strict --schema-location https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/ --ignore-missing-schemas --force-color -
	@echo '--------------------'

## docs: generate chart documentation to stdout, if firgate is installed
docs:
	frigate gen .

.PHONY: help
## help: prints this help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

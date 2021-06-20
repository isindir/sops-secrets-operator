# Development Notes

## Files to check and update when building new version

* version of docker image and kubebuilder dependencies

```
Makefile
```

* golang and base image

```
Dockerfile
```

* golang, tools to build and test

```
.circleci/config.yml
.tool-versions
```

* Dependencies - libraries

```
go.mod
```
> needs `make clean; make tidy`

* cluster version (same as kubectl in other files)

```
chart/helm3/sops-secrets-operator/Makefile
```

* chart version, docker image version

```
chart/helm3/sops-secrets-operator/Chart.yaml
chart/helm3/sops-secrets-operator/values.yaml
chart/helm3/sops-secrets-operator/tests/operator_test.yaml
```

* for any new `SopsSecret` api version change needs to be updated

```
PROJECT
```

## Before final merge of the release

Before final merge of the release - package helm chart:

```bash
package-helm
```

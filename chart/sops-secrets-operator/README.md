# sops-secrets-operator

Installs [sops-secrets-operator](https://github.com/isindir/sops-secrets-operator.git) to provide encrypted secrets in Weaveworks GitOps Flux environment.

## TL;DR;

```console
$ kubectl create namespace sops

$ kubectl apply -f deploy/crds/isindir_v1alpha1_sopssecret_crd.yaml

$ helm upgrade --install sops chart/sops-secrets-operator/ \
  --namespace sops -f custom.values.yaml
```

> where `custom.values.yaml` must customise deployment and configure access to Cloud KMS

* AWS is supported via `kiam` namespace and pod annotations
* GCP is supported via service account secret which allows decryption using GCP KMS
* **TODO:** GPG support
* **TODO:** Azure support

## Introduction

This chart bootstraps a [sops-secrets-operator](https://github.com/isindir/sops-secrets-operator.git) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites
  - Kubernetes 1.12+

## Installing the Chart

### AWS

* Deploy [kiam](https://github.com/uswitch/kiam) using [kiam chart](https://github.com/helm/charts/tree/master/stable/kiam)
* Create IAM assume role which allows to use KMS key for decryption
* Create Kubernetes namespace for operator deployment, with kiam annotation
* Apply `sops-secrets-operator` CRD
* Deploy helm chart

### GCP

* Create GCP Service Account which allows to use KMS to decrypt
* Create custom values file in a following format:

```yaml
gcp:
  enabled: true
  svcAccSecret: |-
    {
      "type": "service_account",
      ...
    }
```

* Create Kubernetes namespace for operator deployment
* Apply `sops-secrets-operator` CRD
* Deploy helm chart specifying extra values file

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm delete --purge sops
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following table lists the configurable parameters of the kiam chart and their default values.

Parameter | Description | Default
--- | --- | ---
`replicaCount` | Deployment replica count  - should not be modified | `1`
`image.repository` | Operator image | `isindir/sops-secrets-operator`
`image.tag` | Operator image tag | `0.0.6`
`image.pullPolicy` | Operator image pull policy | `AlwaysPull`
`imagePullSecrets` | Secrets to pull image from private docker repository | `[]`
`nameOverride` | Overrides auto-generated short resource name | `""`
`fullnameOverride` | Overrides auto-generated long resource name | `""`
`podAnnotations` | Annotations to be added to agent pods | `{}`
`watchNamespace` | Namespace to watch CRs, if not specified all namespaces will be watched | `""`
`gcp.enabled` | If `true` GCP secret will be created from provided value and mounted as environment variable | `false`
`gcp.svcAccSecretCustomName` | Name of the secret to create - will override default secret name if specified | `''`
`gcp.svcAccSecret` | If `gcp.enabled` is `true`, this value must be specified as GCP service account secret json payload | `''`
`nodeSelector` | Node labels for operator pod assignment | `{}`
`resources` | Operator container resources | `{}`
`tolerations` | Tolerations to be applied to operator pod | `[]`
`affinity` | Node affinity for pod assignment | `{}`
`rbac.enabled` | If `true`, create & use RBAC resources | `true`

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

> **Tip**: You can use the default [values.yaml](values.yaml)

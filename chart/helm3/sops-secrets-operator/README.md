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

* AWS is supported via `kiam` namespace and pod annotations or via [IAM roles for service accounts](https://docs.aws.amazon.com/eks/latest/userguide/specify-service-account-role.html)
* GCP is supported via service account secret which allows decryption using GCP KMS
* GPG is supported via secrets with GPG configuration
* Azure is supported via a Service principal plus a secret

## Introduction

This chart bootstraps a [sops-secrets-operator](https://github.com/isindir/sops-secrets-operator.git) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites
  - Kubernetes 1.12+
  - helm 3.+

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

### Azure

* Create a KeyVault if you don't have one already
* Create a Key in that KeyVault
* Create Service principal with permissions to use the key for Encryption/Decryption
  * follow the [SOPS documentation](https://github.com/mozilla/sops#encrypting-using-azure-key-vault)
* Either put Tenant ID, Client ID and Client Secret for the Service Principal in your custom values.yaml file or create a Kubernetes Secret with the same information and put the name of that secret in your values.yaml. Enable Azure in the Helm Chart by setting `azure.enabled: true` in values.yaml.

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm uninstall sops
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following table lists the configurable parameters of the Sops-secrets-operator chart and their default values.

| Parameter                | Description             | Default        |
| ------------------------ | ----------------------- | -------------- |
| `replicaCount` | Deployment replica count  - should not be modified | `1` |
| `image.repository` | Operator image | `"isindir/sops-secrets-operator"` |
| `image.tag` | Operator image tag | `"0.1.7"` |
| `image.pullPolicy` | Operator image pull policy | `"Always"` |
| `imagePullSecrets` | Secrets to pull image from private docker repository | `[]` |
| `nameOverride` | Overrides auto-generated short resource name | `""` |
| `fullnameOverride` | Overrides auto-generated long resource name | `""` |
| `podAnnotations` | Annotations to be added to operator pod | `{}` |
| `serviceAccount.annotations` | Annotations to be added to the service account | `{}` |
| `gpg.enabled` | If `true` gcp secret will be created from provided value and mounted as environment variable | `false` |
| `gpg.secret1` | Name of the secret to create - will override default secret name if specified | `"gpg1"` |
| `gpg.secret2` | Name of the secret to create - will override default secret name if specified | `"gpg2"` |
| `gcp.enabled` | Node labels for operator pod assignment | `false` |
| `gcp.svcAccSecretCustomName` | Name of the secret to create - will override default secret name if specified | `""` |
| `gcp.svcAccSecret` | If `gcp.enabled` is `true`, this value must be specified as gcp service account secret json payload | `""` |
| `azure.enabled` | If true azure keyvault will be used | `false` |
| `azure.tenantId` | Tenantid of azure service principal to use | `""` |
| `azure.clientId` | Clientid (application id) of azure service principal to use | `""` |
| `azure.clientSecret` | Client secret of azure service principal | `""` |
| `azure.existingSecretName` | Name of a pre-existing secret containing azure service principal credentials (clientid, clientsecret, tenantid) | `""` |
| `secretsAsEnvVars` | Configure custom secrets to be used as environment variables at runtime, see values.yaml | `[]` |
| `secretsAsFiles` | Configure custom secrets to be mounted at runtime, see values.yaml | `[]` |
| `resources` | Operator container resources | `{}` |
| `nodeSelector` | Node selector to use for pod configuration | `{}` |
| `securityContext.enabled` | Enable securitycontext | `false` |
| `securityContext.runAsUser` | Uid to run as | `1000` |
| `securityContext.runAsGroup` | Gid to run as | `3000` |
| `securityContext.fsGroup` | Fs group | `2000` |
| `tolerations` | Tolerations to be applied to operator pod | `[]` |
| `affinity` | Node affinity for pod assignment | `{}` |
| `rbac.enabled` | Create and use rbac resources | `true` |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

> **Tip**: You can use the default [values.yaml](values.yaml)

---
_Documentation generated by [Frigate](https://frigate.readthedocs.io)._

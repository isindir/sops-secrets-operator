# sops-secrets-operator

Helm chart deploys sops-secrets-operator

## Source Code

* <https://github.com/isindir/sops-secrets-operator.git>

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
* Vault is supported via vault agent injections
* Age is supported via mounting secret and defining environment variable

## Introduction

This chart bootstraps a [sops-secrets-operator](https://github.com/isindir/sops-secrets-operator.git) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Installing the Chart

### AWS

* Deploy [kiam](https://github.com/uswitch/kiam) using [kiam chart](https://github.com/helm/charts/tree/master/stable/kiam)
  * Alternatively [kube2iam](https://github.com/jtblin/kube2iam)
  * Or provide credentials to perform AWS KMS operations
* Create IAM assume role which allows to use KMS key for decryption
* Create Kubernetes namespace for operator deployment, with kiam annotation
* Apply `sops-secrets-operator` CRD
* Deploy helm chart

### GCP

* Create GCP Service Account which allows to use KMS to decrypt
* Either put the GCP Service Account JSON file in your custom values.yaml file or create a Kubernetes Secret with the same information and put the name of that secret in your values.yaml. Enable GCP in the Helm Chart by setting `gcp.enabled: true` in values.yaml.

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

or

```yaml
gcp:
  enabled: true
  existingSecretName: gcp-sa-existing-secret-name
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

### Age

* Create age `keys.txt` file
* Create Kubernetes secret using `keys.txt`
* When deploying helm chart use `extraEnv` value to speicify environment variable `SOPS_AGE_RECIPIENTS` and `secretsAsFiles` value to mount `keys.txt`

For reference see:

* [Age encryption tool](https://github.com/FiloSottile/age)
* [sops section on how to encrypt](https://github.com/mozilla/sops#22encrypting-using-age)
* Also see: [Local testing using age](docs/age/README.md)

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm uninstall sops
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following table lists the configurable parameters of the Sops-secrets-operator chart and their default values.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Node affinity for pod assignment |
| azure | object | `{"clientId":"","clientSecret":"","enabled":false,"existingSecretName":"","tenantId":""}` | Azure KeyVault configuration section |
| azure.clientId | string | `""` | ClientID (Application ID) of Azure Service Principal to use |
| azure.clientSecret | string | `""` | Client Secret of Azure Service Principal |
| azure.enabled | bool | `false` | if true Azure KeyVault will be used |
| azure.existingSecretName | string | `""` | Name of a pre-existing secret containing Azure Service Principal Credentials (ClientID, ClientSecret, TenantID) |
| azure.tenantId | string | `""` | TenantID of Azure Service principal to use |
| extraEnv | list | `[]` | A list of additional environment variables |
| fullnameOverride | string | `""` | Overrides auto-generated long resource name |
| gcp | object | `{"enabled":false,"existingSecretName":"","svcAccSecret":"","svcAccSecretCustomName":""}` | GCP KMS configuration section |
| gcp.enabled | bool | `false` | Node labels for operator pod assignment |
| gcp.existingSecretName | string | `""` | Name of a pre-existing secret containing GCP service account secret json payload |
| gcp.svcAccSecret | string | `""` | If `gcp.enabled` is `true`, this value must be specified as GCP service account secret json payload |
| gcp.svcAccSecretCustomName | string | `""` | Name of the secret to create - will override default secret name if specified |
| gpg | object | `{"enabled":false,"secret1":"gpg1","secret2":"gpg2"}` | GPG configuration section |
| gpg.enabled | bool | `false` | If `true` GCP secret will be created from provided value and mounted as environment variable |
| gpg.secret1 | string | `"gpg1"` | Name of the secret to create - will override default secret name if specified |
| gpg.secret2 | string | `"gpg2"` | Name of the secret to create - will override default secret name if specified |
| healthProbes.liveness | object | `{"initialDelaySeconds":15,"periodSeconds":20}` | Liveness probe configuration |
| healthProbes.port | int | `8081` | The address the probe endpoint binds to. (default ":8081") |
| healthProbes.readiness | object | `{"initialDelaySeconds":5,"periodSeconds":10}` | Readiness probe configuration |
| image.pullPolicy | string | `"Always"` | Operator image pull policy |
| image.repository | string | `"isindir/sops-secrets-operator"` | Operator image name |
| image.tag | string | `"0.4.4"` | Operator image tag |
| imagePullSecrets | list | `[]` | Secrets to pull image from private docker repository |
| initImage.pullPolicy | string | `"Always"` | Init container image pull policy |
| initImage.repository | string | `"ubuntu"` | Init container image name |
| initImage.tag | string | `"focal-20220113"` | Init container image tag |
| kubeconfig | object | `{"enabled":false,"path":null}` | Paths to a kubeconfig. Only required if out-of-cluster. |
| logging | object | `{"encoder":"json","level":"info","stacktraceLevel":"error"}` | Logging configuration section suggested values Development Mode (encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode (encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error) (default) |
| logging.encoder | string | `"json"` | Zap log encoding (one of 'json' or 'console') |
| logging.level | string | `"info"` | Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity |
| logging.stacktraceLevel | string | `"error"` | Zap Level at and above which stacktraces are captured (one of 'info', 'error'). |
| nameOverride | string | `""` | Overrides auto-generated short resource name |
| nodeSelector | object | `{}` | Node selector to use for pod configuration |
| podAnnotations | object | `{}` | Annotations to be added to operator pod (can be used with kiam or kube2iam) |
| rbac.enabled | bool | `true` | Create and use RBAC resources |
| replicaCount | int | `1` | Deployment replica count - should not be modified |
| requeueAfter | int | `5` | Requeue failed reconciliation in minutes (min 1). (default 5) |
| resources | object | `{}` | Operator container resources |
| secretsAsEnvVars | list | `[]` | configure custom secrets to be used as environment variables at runtime, see values.yaml |
| secretsAsFiles | list | `[]` | configure custom secrets to be mounted at runtime, see values.yaml |
| securityContext.enabled | bool | `false` | Enable securityContext |
| securityContext.fsGroup | int | `13001` | fs group |
| securityContext.runAsGroup | int | `13001` | GID to run as |
| securityContext.runAsUser | int | `13001` | UID to run as |
| serviceAccount.annotations | object | `{}` | Annotations to be added to the service account |
| tolerations | list | `[]` | Tolerations to be applied to operator pod |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

> **Tip**: You can use the default [values.yaml](values.yaml)

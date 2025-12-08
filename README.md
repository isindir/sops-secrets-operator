[![Go Report Card](https://goreportcard.com/badge/github.com/isindir/sops-secrets-operator?)](https://goreportcard.com/report/github.com/isindir/sops-secrets-operator)
[![Github release action](https://github.com/isindir/sops-secrets-operator/actions/workflows/release.yaml/badge.svg?branch=master)](https://github.com/isindir/sops-secrets-operator/actions/workflows/release.yaml)
[![GitHub release](https://img.shields.io/github/tag/isindir/sops-secrets-operator.svg)](https://github.com/isindir/sops-secrets-operator/releases)
[![Docker pulls](https://img.shields.io/docker/pulls/isindir/sops-secrets-operator.svg)](https://hub.docker.com/r/isindir/sops-secrets-operator)
[![Artifact HUB](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/sops-secrets-operator)](https://artifacthub.io/packages/search?repo=sops-secrets-operator)
[![MPL v2.0](http://img.shields.io/github/license/isindir/sops-secrets-operator.svg)](LICENSE)

# SOPS: Secrets OPerationS - Kubernetes Operator

Operator which manages Kubernetes Secret Resources created from user defined SopsSecrets
CRs, inspired by [Bitnami SealedSecrets](https://github.com/bitnami-labs/sealed-secrets) and
[sops](https://github.com/mozilla/sops). SopsSecret CR defines multiple
kubernetes Secret resources. It supports managing kubernetes Secrets with
annotations and labels, that allows using these kubernetes secrets as [Jenkins Credentials](https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/).
The SopsSecret resources can be deployed by [Flux GitOps CD](https://fluxcd.io/) and
encrypted using [sops](https://github.com/mozilla/sops) for AWS, GCP, Azure or
on-prem hosted kubernetes clusters. Using `sops` greatly simplifies changing
encrypted files stored in `git` repository.

# Versioning

[//]: # "UPDATE_HERE"

| Kubernetes | Sops    | Chart  | Operator |
| ---------- | ------- | ------ | -------- |
| v1.34.x    | v3.11.0 | 0.24.1 | 0.17.3   |
| v1.33.x    | v3.10.2 | 0.22.0 | 0.16.0   |
| v1.32.x    | v3.9.4  | 0.21.0 | 0.15.0   |
| v1.31.x    | v3.9.4  | 0.20.5 | 0.14.3   |
| v1.30.x    | v3.9.0  | 0.19.4 | 0.13.3   |
| v1.29.x    | v3.8.1  | 0.18.6 | 0.12.6   |
| v1.28.x    | v3.8.1  | 0.17.4 | 0.11.4   |
| v1.27.x    | v3.7.3  | 0.15.5 | 0.9.5    |
| v1.26.x    | v3.7.3  | 0.14.2 | 0.8.2    |
| v1.25.x    | v3.7.3  | 0.12.5 | 0.6.4    |
| v1.24.x    | v3.7.3  | 0.11.3 | 0.5.3    |
| v1.23.x    | v3.7.2  | 0.10.8 | 0.4.8    |
| v1.22.x    | v3.7.1  | 0.9.7  | 0.3.7    |
| v1.21.x    | v3.7.1  | 0.9.6  | 0.3.6    |

# Requirements for building operator from source code

Requirements for building operator from source code can be found in [.tool-versions](.tool-versions), this file can be used with [asdf](https://asdf-vm.com/#/)

# Operator Installation

## Helm repository

Add `helm` repository for chart installation:

```bash
helm repo add sops https://isindir.github.io/sops-secrets-operator/
```

## AWS

- Create KMS key
- Create AWS Role which can be used by operator to decrypt CR data structure,
  follow [sops documentation](https://github.com/mozilla/sops#26assuming-roles-and-using-kms-in-various-aws-accounts)
- Deploy CRD:

```bash
kubectl apply -f config/crd/bases/isindir.github.com_sopssecrets.yaml
```

> **NOTE:** to grant access to aws for `sops-secret-operator` -
> [kiam](https://github.com/uswitch/kiam), [kube2iam](https://github.com/jtblin/kube2iam) or [IAM roles for service accounts](https://docs.aws.amazon.com/eks/latest/userguide/specify-service-account-role.html) can be used.

- Deploy helm chart:

```bash
kubectl create namespace sops

helm repo add sops https://isindir.github.io/sops-secrets-operator/
helm upgrade --install sops sops/sops-secrets-operator --namespace sops
```

## Age

- Create age reference `keys.txt` file, create kubernetes secret from it.
- Deploy helm chart:
  - Use `secretsAsFiles` to specify the secret which contains the `keys.txt`.
  - Use `extraEnv` and specify mounted `keys.txt` path `SOPS_AGE_KEY_FILE` environment variable.

See example:

```yaml
---
secretsAsFiles:
  - mountPath: /etc/sops-age-key-file
    name: sops-age-key-file
    secretName: sops-age-key-file
extraEnv:
  - name: SOPS_AGE_KEY_FILE
    value: /etc/sops-age-key-file/key
```

- Also see: [Local testing using age](docs/age/README.md)

References:

- [Age git repository](https://github.com/FiloSottile/age)
- [SOPS Age documentation](https://github.com/mozilla/sops#22encrypting-using-age)

## PGP

For instructions on how-to configure PGP keys for operator, see [Preparing GPG keys](docs/gpg/README.md)

Then install operator:

```bash
kubectl create namespace sops

kubectl apply -f docs/gpg/1.yaml --namespace sops
kubectl apply -f docs/gpg/2.yaml --namespace sops

kubectl apply -f config/crd/bases/isindir.github.com_sopssecrets.yaml

helm repo add sops https://isindir.github.io/sops-secrets-operator/
helm upgrade --install sops sops/sops-secrets-operator \
  --namespace sops --set gpg.enabled=true
```

## Azure

### Outline

- Create a KeyVault if you don't have one already
- Create a Key in that KeyVault
- Create Service principal with permissions to use the key for Encryption/Decryption
  - follow the [SOPS documentation](https://github.com/mozilla/sops#encrypting-using-azure-key-vault)
- Either put Tenant ID, Client ID and Client Secret for the Service Principal in your custom values.yaml file or create a Kubernetes Secret with the same information and put the name of that secret in your values.yaml. Enable Azure in the Helm Chart by setting `azure.enabled: true` in values.yaml.

### Login info in values.yaml

```bash
cat <<EOF > azure_values.yaml
azure:
  enabled: true
  tenantId: 6ec4c881-32ee-4340-a456-d6ca65a42193
  clientId: 9c325550-b264-4aee-ab6f-719771adda28
  clientSecret: 'YOUR_CLIENT_SECRET'
EOF

kubectl create namespace sops

helm repo add sops https://isindir.github.io/sops-secrets-operator/
helm upgrade --install sops sops/sops-secrets-operator \
  --namespace sops -f azure_values.yaml
```

### Use pre-existing secret for Azure login

```bash
cat <<EOF > azure_secret.yaml
kind: Secret
apiVersion: v1
metadata:
  name: azure-sp-credentials
type: Opaque
stringData:
  clientId: 9c325550-b264-4aee-ab6f-719771adda28
  tenantId: 6ec4c881-32ee-4340-a456-d6ca65a42193
  clientSecret: 'YOUR_CLIENT_SECRET'
EOF

cat <<EOF > azure_values.yaml
azure:
  enabled: true
  existingSecret: azure-sp-credentials
EOF

kubectl create namespace sops
kubectl apply -n sops -f azure_secret.yaml

helm repo add sops https://isindir.github.io/sops-secrets-operator/
helm upgrade --install sops sops/sops-secrets-operator \
  --namespace sops -f azure_values.yaml
```

## SopsSecret Custom Resource File creation

- create SopsSecret file, for example:

```yaml
cat >jenkins-secrets.yaml <<EOF
apiVersion: isindir.github.com/v1alpha3
kind: SopsSecret
metadata:
  name: example-sopssecret
spec:
  # suspend reconciliation of the sops secret object
  suspend: false
  secretTemplates:
    - name: my-secret-name-1
      labels:
        label1: value1
      annotations:
        key1: value1
      stringData:
        data-name0: data-value0
      data:
        data-name1: ZGF0YS12YWx1ZTE=
    - name: jenkins-secret
      labels:
        "jenkins.io/credentials-type": "usernamePassword"
      annotations:
        "jenkins.io/credentials-description": "credentials from Kubernetes"
      stringData:
        username: myUsername
        password: 'Pa$$word'
    - name: some-token
      stringData:
        token: Wb4ziZdELkdUf6m6KtNd7iRjjQRvSeJno5meH4NAGHFmpqJyEsekZ2WjX232s4Gj
    - name: docker-login
      type: 'kubernetes.io/dockerconfigjson'
      stringData:
        .dockerconfigjson: '{"auths":{"index.docker.io":{"username":"imyuser","password":"mypass","email":"myuser@abc.com","auth":"aW15dXNlcjpteXBhc3M="}}}'
EOF
```

## Cross-Namespace Secret Deployment

Starting with v1alpha3, `SopsSecret` supports deploying secrets to multiple namespaces using the `targetNamespaces` field. This is useful when the same secret needs to be accessible across different namespaces without duplication.

### Example: Deploy Secret to Multiple Namespaces

```yaml
apiVersion: isindir.github.com/v1alpha3
kind: SopsSecret
metadata:
  name: database-credentials
  namespace: production
spec:
  secretTemplates:
    # This secret will be created in multiple namespaces
    - name: db-secret
      targetNamespaces:
        - production  # Same namespace as SopsSecret (uses controller ownership)
        - backend     # Cross-namespace (uses annotation-based management)
        - worker      # Cross-namespace (uses annotation-based management)
      stringData:
        DB_HOST: mydb.example.com
        DB_PASSWORD: encrypted_password_here
        DB_USER: dbuser
```

### How Cross-Namespace Secrets Work

- **Same-namespace secrets**: When a secret is deployed to the same namespace as the `SopsSecret`, it uses standard Kubernetes controller ownership for automatic garbage collection.
- **Cross-namespace secrets**: When a secret is deployed to a different namespace, it uses annotation-based management (`sopssecret/owner` annotation) since Kubernetes doesn't allow cross-namespace owner references.
- **Backward compatibility**: If `targetNamespaces` is not specified or is empty, the secret is deployed to the `SopsSecret`'s namespace (existing behavior).

### RBAC Requirements for Cross-Namespace Secrets

To deploy secrets across namespaces, the operator requires appropriate RBAC permissions. Ensure the operator's `ClusterRole` includes:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: sops-secrets-operator
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

- Encrypt file using `sops` and AWS kms key:

```bash
sops --encrypt \
  --kms 'arn:aws:kms:<region>:<account>:alias/<key-alias-name>' \
  --encrypted-suffix='Templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

or

```bash
sops --encrypt \
  --kms 'arn:aws:kms:<region>:<account>:alias/<key-alias-name>' \
  --encrypted-regex='^(data)$' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

> **NOTE:** after using regex `sops --encrypted-regex` resulting file may be inapplicable to the kubernetes cluster, use
> this feature with care

- Encrypt file using `sops` and GCP KMS key:

```bash
sops --encrypt \
  --gcp-kms 'projects/<project-name>/locations/<location>/keyRings/<keyring-name>/cryptoKeys/<key-name>' \
  --encrypted-suffix='Templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

- Encrypt file using `sops` and Azure Keyvault key:

```bash
sops --encrypt \
  --azure-kv 'https://<vault-url>/keys/<key-name>/<key-version>' \
  --encrypted-suffix='Templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

- Encrypt file using `sops` and PGP key:

```bash
sops --encrypt \
  --pgp '<pgp-finger-print>' \
  --encrypted-suffix='Templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

> **Note:** Multiple keys can be used to encrypt secrets. At the time of decryption
> access to one of these is needed. For more information see `sops`
> documentation.

## Changing ownership of existing secrets

If there is a need to re-own existing `Secrets` by `SopsSecret`, following annotation should
be added to the target kubernetes native secret:

```yaml
---
metadata:
  annotations:
    "sopssecret/managed": "true"
```

> previously not managed secret will be replaced by `SopsSecret` owned at the next rescheduled
> reconciliation event.

## Example procedure to upgrade from one `SopsSecret` API version to another

Please see document here: [SopsSecret API and Operator Upgrade](docs/api_upgrade_example/README.md)

# License

Mozilla Public License Version 2.0

# Known Issues

- `sops-secrets-operator` is not using standard `sops` library decryption
  interface function, modified upstream function is used to decrypt data which
  ignores `enc` signature field in `sops` metadata. This means if some encrypted
  fields are removed or changed to plain text - it still will be able to decrypt
  the resource.This is due to the fact that when Kubernetes resource is applied
  it is always mutated by Kubernetes, for example resource version is generated
  and added to the resource. But any mutation invalidates `sops` metadata `enc`
  field and standard decryption function fails.
- `sops-secrets-operator` by design is not wrapping encrypted object to some
  field in spec. This was deliberate decision for the simplicity of the
  operations - ability to directly encrypt the whole `SopsSecret` resource using
  `sops` cli. This causes side effects like: if the user of the k8s cluster
  (which runs `sops-secrets-operator`) has RBAC access to read secrets in some
  namespace - it allows directly applying encrypted `SopsSecret` resource to
  that namespaces and getting access to the secret material. This operator was
  only designed to protect access to the secret material from git repository.
- `sops-secrets-operator` is not strictly following
  [Kubernetes OpenAPI naming conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#naming-conventions).
  This is due to the fact that `sops` generates substructures in encrypted file
  with incompatible to OpenAPI names (containing underscore symbols, where it
  should be `lowerCamelCase` for OpenAPI compatibility).

# Links

Projects and tools inspired development of `sops-secrets-operator`:

- [sops](https://github.com/mozilla/sops)
  - [Configuring AWS KMS for use with sops](https://github.com/mozilla/sops#26assuming-roles-and-using-kms-in-various-aws-accounts)
  - [helm secrets plugin](https://github.com/jkroepke/helm-secrets)
- [kube2iam](https://github.com/jtblin/kube2iam)
  - [kiam](https://github.com/uswitch/kiam) - in ABANDONED mode now
- [Flux GitOps CD](https://fluxcd.io/) - flux supports `sops` out of the box
  - [Flux github repositories](https://github.com/fluxcd)
  - [Flux sops native integration documentation](https://fluxcd.io/flux/guides/mozilla-sops/)
- [Jenkins Configuration as Code](https://jenkins.io/projects/jcasc/)
  - [Jenkins - Kubernetes Credentials Provider](https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/)
  - [Jenkins Kubernetes Plugin](https://github.com/jenkinsci/kubernetes-plugin)
- [Bitnami SealedSecrets](https://github.com/bitnami-labs/sealed-secrets)
  - [Using sealed secrets with Flux](https://fluxcd.io/flux/guides/sealed-secrets/)
- [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
  - [operator-sdk](https://github.com/operator-framework/operator-sdk)

## Alternative tools

- [Sops Operator](https://github.com/peak-scale/sops-operator) (Peak Scale)
- [Kubernetes external secrets](https://github.com/external-secrets/external-secrets)
- [Vault Secrets Operator](https://github.com/ricoberger/vault-secrets-operator)
- [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets)
- [Secrets Store CSI driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver)
- [Kamus](https://kamus.soluto.io/)
- [Tesoro](https://github.com/kapicorp/tesoro)
- [Sops Operator](https://github.com/craftypath/sops-operator) (Craftypath)

## Stargazers over time

[![Stargazers over time](https://starchart.cc/isindir/sops-secrets-operator.svg?variant=light)](https://starchart.cc/isindir/sops-secrets-operator)

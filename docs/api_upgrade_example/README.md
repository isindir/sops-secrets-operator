# SopsSecret API and Operator Upgrade

This readme describes how to upgrade without downtime from one `SopsSecret`
API version to another. This example is very specific, but same principles
should work for other versions. `sops-secrets-operator` does not provide
[Conversion Webhook Service](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#webhook-conversion),
but it is still possible to convert `SopsSecrets` from one version to
another and install the latest operator without disruption.

Let's take as an example current deployment which `sops-secrets-operator`
chart version `0.1.7` and application version `0.0.9`, which needs
to be upgraded to the latest at the time of writing version of chart `0.9.0`
and application version `0.3.0`. Assuming we have [asdf](https://asdf-vm.com/#/)
installed and `sops`, `kind`, `kubectl` and `helm` plugins added (which
will be needed to manage multiple versions of mozilla `sops` tool)
and that current helm release deployed via `helm` version `3.x.x`
and we are using latest `helm` version to upgrade the release.

The test environment will be AWS KMS key and local
[kind](https://kind.sigs.k8s.io/docs/user/quick-start/) cluster.

## Create and prepare `kind` test cluster

* Create cluster:

```bash
kind create cluster
```
> Single node cluster

* Create `namespace` for our test:

```bash
kubectl create ns sops
```

* Switch context namespace to `sops`:

```bash
kubens sops
```

## Check compatible `sops` versions

* Checkout `sops-secrets-operator` repository to specific version
  of application, in our case `0.0.9` and then `0.3.0` and check `sops`
  library version:

```bash
cd sops-secrets-operator
git checkout 0.0.9
grep sops go.mod
```

> output is something like:

```
module github.com/isindir/sops-secrets-operator
        go.mozilla.org/sops/v3 v3.5.0
```

> or for older versions could be:

```
module github.com/isindir/sops-secrets-operator
        go.mozilla.org/sops v0.0.0-20190611200209-e9e1e87723c8
```

For version `0.0.9` of operator it is recommended to use version `3.5.0` of `sops`:

```bash
asdf install sops 3.5.0
asdf shell sops 3.5.0
```

For older versions `git` commit sha we are interested in is in version,
so we can obtain compatible version of `sops`:

```bash
cd sops
git checkout e9e1e87723c8
git describe --tags --abbrev=0
```

> the output will be:

```
3.3.1
```

> install and use `sops` version `3.3.1` via `asdf`

# Installing old CRD/Operator/SopsSecret

* Install CRD:

```bash
# crd version v1alpha1
kubectl apply -f crd.0.0.9.yaml
```

* Install operator:

```bash
# app: 0.0.9; chart: 0.1.7
helm upgrade \
  --install sops sops-secrets-operator.0.0.9 \
  -f values.0.0.9.custom.yaml \
  -n sops
```

> where `values.0.0.9.custom.yaml` is something like follows:

```yaml
extraEnv:
- name: AWS_SDK_LOAD_CONFIG
  value: "1"
- name: AWS_ACCESS_KEY_ID
  value: "<place-yours-here>"
- name: AWS_SECRET_ACCESS_KEY
  value: "<place-yours-here>"
- name: AWS_DEFAULT_REGION
  value: eu-west-2
```

> **NOTE:** original chart does not support `.Values.extraEnv`,
  so it was added for testing purposes

* Create test `SopsSecrets` object and observe its reconciliation:

```bash
cat <<EOF >>plain.v1alpha1.yaml
apiVersion: isindir.github.com/v1alpha1
kind: SopsSecret
metadata:
  name: example-sopssecret
spec:
  secret_templates:
    - name: jenkins-secret
      labels:
        "jenkins.io/credentials-type": "usernamePassword"
      annotations:
        "jenkins.io/credentials-description" : "credentials from Kubernetes"
      data:
        username: myUsername
        password: 'Pa$$word'
    - name: some-token
      data:
        token: Wb4ziZdELkdUf6m6KtNd7iRjjQRvSeJno5meH4NAGHFmpqJyEsekZ2WjX232s4Gj
EOF

# sops: 3.5.0
sops -e \
  -k 'arn:aws:kms:<region>:<account>:alias/<kms-key-alias-name>' \
  --encrypted-suffix='_templates' \
  plain.v1alpha1.yaml > enc.v1alpha1.yaml

kubectl apply -f enc.v1alpha1.yaml
```

## Patch CRD to support multiple versions

* See [Documentation](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#specify-multiple-versions)
  on how to specify multiple version in a CRD, by default `SopsSecrets` CRD
  is bound to one version only. As our source is of `apiextensions.k8s.io/v1beta1`
  CRD definition, we add following block:

```diff
  version: v1alpha1               version: v1alpha1
  versions:                       versions:
  - name: v1alpha1                - name: v1alpha1
    served: true                    served: true
    storage: true                   storage: true
                              >     deprecated: true
                              >   - name: v1alpha3
                              >     storage: false
                              >     served: true
                              >     deprecated: false
```

* Apply modified version of the CRD:

```bash
kubectl apply -f crd.0.0.9.patched.yaml
```

## Scale down replicas of operator

So that no secret overwrites happens, scale down `sops-secrets-operator` deployment:

```bash
kubectl scale --replicas=0 deployment/sops-sops-secrets-operator
```

## Prepare new version of secret

From `v1alpha1` to `v1alpha3` following changes happend:

* api version changed
* data `_template` prefix changed to `Template`
* `data` of secret element becomes `stringData`

So our new secret will look like follows:

```bash
cat <<EOF >>plain.v1alpha3.yaml
apiVersion: isindir.github.com/v1alpha3
kind: SopsSecret
metadata:
  name: example-sopssecret
spec:
  secretTemplates:
    - name: jenkins-secret
      labels:
        "jenkins.io/credentials-type": "usernamePassword"
      annotations:
        "jenkins.io/credentials-description" : "credentials from Kubernetes"
      stringData:
        username: myUsername
        password: 'Pa$$word'
    - name: some-token
      stringData:
        token: Wb4ziZdELkdUf6m6KtNd7iRjjQRvSeJno5meH4NAGHFmpqJyEsekZ2WjX232s4Gj
EOF

```

* encryption command also changes to:

```bash
sops -e \
  -k 'arn:aws:kms:<region>:<account>:alias/<kms-key-alias-name>' \
  --encrypted-suffix='Templates' \
  plain.v1alpha3.yaml > enc.v1alpha3.yaml
```

## Patch latest CRD to support multiple versions

As our source is of `apiextensions.k8s.io/v1` CRD definition, we add
following block to support multiple versions:

```diff
  versions:                    versions:
                           >   - name: v1alpha1
                           >     served: true
                           >     storage: false
                           >     deprecated: true
                           >     subresources:
                           >       status: {}
                           >     schema:
                           >       openAPIV3Schema:
                           >         properties:
                          ...
                           >               status:
                           >                 type: object
```

> where `openAPIV3Schema` is taken from `v1alpha1` version of CRD.

## Patch CRD and all `SopsSecrets`

Patch CRD and all `SopsSecrets` to the latest api version:

```bash
kubectl apply -f crd.0.3.0.patched.yaml
kubectl apply -f enc.v1alpha3.yaml
```

## Upgrade helm chart to the latest version

* Verify how the deployment will change (for this step [helm diff](https://github.com/databus23/helm-diff)
  plugin is needed):

```bash
helm diff upgrade --install sops sops-secrets-operator.0.3.0 \
  -f values.0.3.0.custom.yaml \
  -n sops
```

* Perform upgrade:

```bash
helm upgrade --install sops sops-secrets-operator.0.3.0 \
  -f values.0.3.0.custom.yaml \
  -n sops
```

> where `values.0.3.0.custom.yaml` is like follows:

```yaml
extraEnv:
  - name: AWS_SDK_LOAD_CONFIG
    value: "1"
  - name: AWS_ACCESS_KEY_ID
    value: "<place-yours-here>"
  - name: AWS_SECRET_ACCESS_KEY
    value: "<place-yours-here>"
  - name: AWS_DEFAULT_REGION
    value: eu-west-2
```

* Observe secrets will be refreshed by operator and k8s native secrets
  are in sync.

Some aforementioned files can be found in [examples](examples).

## Cleanup

Finally delete the `kind` cluster

```bash
kind delete cluster
```

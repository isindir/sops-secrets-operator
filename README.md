[![Go Report Card](https://goreportcard.com/badge/github.com/isindir/sops-secrets-operator?)](https://goreportcard.com/report/github.com/isindir/sops-secrets-operator)
[![CircleCI](https://circleci.com/gh/isindir/sops-secrets-operator.svg?style=svg)](https://circleci.com/gh/isindir/sops-secrets-operator)
[![GitHub release](https://img.shields.io/github/tag/isindir/sops-secrets-operator.svg)](https://github.com/isindir/sops-secrets-operator/releases)
[![Docker pulls](https://img.shields.io/docker/pulls/isindir/sops-secrets-operator.svg)](https://hub.docker.com/r/isindir/sops-secrets-operator)
[![MPL v2.0](http://img.shields.io/github/license/isindir/sops-secrets-operator.svg)](LICENSE)

# SOPS: Secrets OPerationS - Kubernetes Operator

Operator which manages Kubernetes Secret Resources created from user defined SopsSecrets
CRs, inspired by [Bitnami SealedSecrets](https://github.com/bitnami-labs/sealed-secrets) and
[sops](https://github.com/mozilla/sops). SopsSecret CR defines multiple
kubernetes Secret resources. It supports managing kubernetes Secrets with
annotations and labels, that allows using these kubernetes secrets as [Jenkins Credentials](https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/).
The SopsSecret resources can be deployed by [Weaveworks Flux GitOps CD](https://www.weave.works/blog/managing-helm-releases-the-gitops-way) and
encrypted using [sops](https://github.com/mozilla/sops) for AWS, GCP, Azure or
on-prem hosted kubernetes clusters. Using `sops` greatly simplifies changing
encrypted files stored in `git` repository.

# Requirements for building operator from source code

* sops - 3.5.0
* operator-sdk 0.18.2
* golang - 1.14.4
* helm - 3.+

# Operator Installation

## AWS

* Create KMS key
* Create AWS Role which can be used by operator to decrypt CR data structure,
  follow [sops documentation](https://github.com/mozilla/sops#26assuming-roles-and-using-kms-in-various-aws-accounts)
* Deploy CRD:

```bash
kubectl apply -f deploy/crds/isindir_v1alpha1_sopssecret_crd.yaml
```
> **NOTE:** to grant access to aws for `sops-secret-operator` -
> [kiam](https://github.com/uswitch/kiam) can be used.

* Deploy helm chart:

```bash
kubectl create namespace sops

helm upgrade --install sops chart/sops-secrets-operator/ \
  --namespace sops
```

## PGP

For instructions on howto configure PGP keys for operator, see [Preparing GPG keys](docs/gpg/README.md)

Then install operator:

```bash
kubectl create namespace sops

kubectl apply -f docs/gpg/1.yaml --namespace sops
kubectl apply -f docs/gpg/2.yaml --namespace sops

kubectl apply -f chart/crds/isindir_v1alpha1_sopssecret_crd.yaml

helm upgrade --install sops chart/sops-secrets-operator/ \
  --namespace sops --set gpg.enabled=true
```

## SopsSecret Custom Resource File creation

* create SopsSecret file, for example:

```yaml
cat >jenkins-secrets.yaml <<EOF
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
    - name: docker-login
      type: 'kubernetes.io/dockerconfigjson'
      data:
        .dockerconfigjson: '{"auths":{"index.docker.io":{"username":"imyuser","password":"mypass","email":"myuser@abc.com","auth":"aW15dXNlcjpteXBhc3M="}}}'
EOF
```

* Encrypt file using `sops` and AWS kms key:

```bash
sops --encrypt \
  --kms 'arn:aws:kms:<region>:<account>:alias/<key-alias-name>' \
  --encrypted-suffix='_templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

* Encrypt file using `sops` and GCP KMS key:

```bash
sops --encrypt \
  --gcp-kms 'projects/<project-name>/locations/<location>/keyRings/<keyring-name>/cryptoKeys/<key-name>' \
  --encrypted-suffix='_templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

* Encrypt file using `sops` and Azure Keyvault key:

```bash
sops --encrypt \
  --azure-kv 'https://<vault-url>/keys/<key-name>/<key-version>' \
  --encrypted-suffix='_templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

* Encrypt file using `sops` and PGP key:

```bash
sops --encrypt \
  --pgp '<pgp-finger-print>' \
  --encrypted-suffix='_templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

> **Note:** Multiple keys can be used to encrypt secrets. At the time of decryption
> access to one of these is needed. For more information see `sops`
> documentation.

# License

Mozilla Public License Version 2.0

# Known Issues

* `sops-secrets-operator` is not following
  [Kubernetes OpenAPI naming conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#naming-conventions),
  because of that it is not possible to generate OpenAPI definition using
  `operator-sdk generate openapi` command. This is due to the fact that `sops`
  generates substructures in encrypted file with incompatible to OpenAPI names
  (containing underscore symbols, where it should be `camelCase` for OpenAPI
  compatibility).
* `sops-secrets-operator` is not using standard `sops` library decryption
  interface function, modified upstream function is used to decrypt data which
  ignores `enc` signature field in `sops` metadata. This is due to the fact that
  when Kubernetes resource is applied it is always mutated by Kubernetes, for
  example resource version is generated and added to the resource. But any
  mutation invalidates `sops` metadata `enc` field and standard decryption
  function fails.

# Links

Projects and tools inspired development of `sops-secrets-operator`:

* [sops](https://github.com/mozilla/sops)
  * [Configuring AWS KMS for use with sops](https://github.com/mozilla/sops#26assuming-roles-and-using-kms-in-various-aws-accounts)
  * [helm secrets plugin](https://github.com/futuresimple/helm-secrets)
* [kiam](https://github.com/uswitch/kiam)
* [Weaveworks Flux - GitOps](https://www.weave.works/blog/managing-helm-releases-the-gitops-way)
  * [Flux github repository](https://github.com/weaveworks/flux)
* [Jenkins Configuration as Code](https://jenkins.io/projects/jcasc/)
  * [Jenkins - Kubernetes Credentials Provider](https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/)
  * [Jenkins Kubernetes Plugin](https://github.com/jenkinsci/kubernetes-plugin)
* [Bitnami SealedSecrets](https://github.com/bitnami-labs/sealed-secrets)
* [operator-sdk](https://github.com/operator-framework/operator-sdk)

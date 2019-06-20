[![CircleCI](https://circleci.com/gh/isindir/sops-secrets-operator.svg?style=svg)](https://circleci.com/gh/isindir/sops-secrets-operator)
[![Docker pulls](https://img.shields.io/docker/pulls/isindir/sops-secrets-operator.svg)](https://hub.docker.com/r/isindir/sops-secrets-operator)

# Operator Installation

## Requirements

### AWS

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
EOF
```

* Encrypt file using `sops` and AWS kms key:

```bash
sops --encrypt \
  --kms 'arn:aws:kms:<region>:<account>:alias/<key-alias-name>' \
  --encrypted-suffix='_templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

* Encrypt file using `sops` and Azure Keyvault key:

```bash
sops --encrypt \
  --azure-kv "https://<vault-url>/keys/<key-name>/<key-version>" \
  --encrypted-suffix='_templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

* Encrypt file using `sops` and PGP key:

```bash
sops --encrypt \
  --pgp "<pgp-finger-print>" \
  --encrypted-suffix='_templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

> **Note:** multiple keys can be used to encrypt secrets, at the time of decryption
> access to one of these is needed, for more information see `sops`
> documentation.

# License

Mozilla Public License Version 2.0

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

# Operator Installation

## Requirements

### AWS

* Create KMS key for use
* Create AWS Role which can be used by operator to decrypt CR data structure,
  follow [sops documentation](https://github.com/mozilla/sops#26assuming-roles-and-using-kms-in-various-aws-accounts)
* Deploy CRD:

```bash
kubectl create namespace sops

kubectl apply -f deploy/crds/isindir_v1alpha1_sopssecret_crd.yaml
```
> **NOTE:** to grant access to aws for `sops-secret-operator` -
> [kiam](https://github.com/uswitch/kiam) can be used.

* Deploy helm chart:

```bash
helm upgrade --install sops chart/sops-secrets-operator/ \
  --namespace sops
```
> **NOTE:** pod annotations can be used to specify AWS role if `kiam` is used.

### SopsSecret Resource File encryption

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
sops -e -k 'arn:aws:kms:<region>:<account>:alias/<key-alias-name>' \
  --encrypted-suffix='_templates' jenkins-secrets.yaml \
  > jenkins-secrets.enc.yaml
```

# Roadmap

With little changes operator should work with GCP KMS and Azure KMS.

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

# sops-secrets-operator

Kubernetes operator that turns user-defined, sops-encrypted SopsSecrets into Kubernetes Secrets.

## Language

### Resources

**SopsSecret**:
The custom resource holding sops-encrypted Secret templates, from which the operator produces Child Secrets.
_Avoid_: sops secret, encrypted secret

**Secret template**:
One entry in a SopsSecret's `spec.secretTemplates` describing exactly one Child Secret.
_Avoid_: template secret

**Child Secret**:
The Kubernetes Secret created from a Secret template and owned by its SopsSecret.
_Avoid_: managed secret, managed k8s secret, owned secret, kube secret

### Keys and decryption

**SSH recipient**:
An SSH public key (`ssh-ed25519` or `ssh-rsa`) used as an age recipient when encrypting a SopsSecret.
_Avoid_: direct SSH key, SSH key (ambiguous: says neither which half nor which role)

**SSH identity**:
The unencrypted SSH private key matching an SSH recipient, made available to the operator so it can decrypt.
_Avoid_: SSH key, SSH private key file, direct SSH key

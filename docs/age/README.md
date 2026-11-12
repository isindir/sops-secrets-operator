# Local testing using age

```bash
rm -f qqq.key.txt
export SOPS_AGE_RECIPIENTS=$( age-keygen -o qqq.key.txt 2>&1 | awk '{ print $3 }' )
export SOPS_AGE_KEY_FILE=$PWD/qqq.key.txt

cat >qqq.jenkins-secrets.yaml <<EOF
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
    - name: docker-login
      type: 'kubernetes.io/dockerconfigjson'
      stringData:
        .dockerconfigjson: '{"auths":{"index.docker.io":{"username":"imyuser","password":"mypass","email":"myuser@abc.com","auth":"aW15dXNlcjpteXBhc3M="}}}'
EOF

sops encrypt --age ${SOPS_AGE_RECIPIENTS} --encrypted-suffix Templates qqq.jenkins-secrets.yaml > qqq.jenkins-secrets.enc.yaml

# check
cat qqq.jenkins-secrets.enc.yaml
sops decrypt qqq.jenkins-secrets.enc.yaml
```

```bash
# build and test
make all

# installs crds
make install

# run controller locally
make run
kubectl apply -f qqq.jenkins-secrets.enc.yaml
```

## Local testing using an SSH recipient

Same flow, reusing `qqq.jenkins-secrets.yaml` from above, but encrypted for an SSH recipient and decrypted with
the matching SSH identity (requires sops `3.10.0` or newer). The file is encrypted only for the SSH recipient, so
decryption relies on the SSH identity; age key variables are unset to make that obvious.

```bash
unset SOPS_AGE_KEY SOPS_AGE_KEY_FILE SOPS_AGE_KEY_CMD SOPS_AGE_RECIPIENTS

rm -f qqq.ssh_ed25519 qqq.ssh_ed25519.pub
# SSH identity must not be passphrase protected
ssh-keygen -q -t ed25519 -N '' -C 'sops-secrets-operator-local-test' -f qqq.ssh_ed25519
export SOPS_AGE_SSH_PRIVATE_KEY_FILE=$PWD/qqq.ssh_ed25519

sops encrypt --age "$(cat qqq.ssh_ed25519.pub)" --encrypted-suffix Templates \
  qqq.jenkins-secrets.yaml > qqq.jenkins-secrets.ssh.enc.yaml

# check: recipient is the SSH public key
grep 'recipient: ssh-ed25519' qqq.jenkins-secrets.ssh.enc.yaml
sops decrypt qqq.jenkins-secrets.ssh.enc.yaml
```

```bash
# installs crds
make install

# run controller locally, SOPS_AGE_SSH_PRIVATE_KEY_FILE is inherited from the shell
make run
kubectl apply -f qqq.jenkins-secrets.ssh.enc.yaml
```

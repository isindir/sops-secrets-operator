# Local testing using age

```bash
rm -f qqq.key.txt
export SOPS_AGE_KEY=$( age-keygen -o qqq.key.txt 2>&1 | awk '{ print $3 }' )
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

sops -e --age ${SOPS_AGE_KEY} --encrypted-suffix Templates qqq.jenkins-secrets.yaml > qqq.jenkins-secrets.enc.yaml

# check
cat qqq.jenkins-secrets.enc.yaml
sops -d qqq.jenkins-secrets.enc.yaml
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

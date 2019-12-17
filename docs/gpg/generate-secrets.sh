#!/usr/bin/env bash

export GNUPGHOME="$( mktemp -d )"

cat >${GNUPGHOME}/foo <<EOF
     %echo Generating a default key
     %no-protection
     Key-Type: default
     Subkey-Type: default
     Name-Real: sops Secrets Operator
     Name-Comment: for use with sops to encrypt/decrypt data files
     Name-Email: sops@secrets.operator
     Expire-Date: 0
     # Do a commit here, so that we can later print "done" :-)
     %commit
     %echo done
EOF

gpg2 --batch --full-gen-key ${GNUPGHOME}/foo

echo

rm -fr ${GNUPGHOME}/foo ${GNUPGHOME}/openpgp-revocs.d/ ${GNUPGHOME}/pubring.kbx~
pkill gpg-agent
tree ${GNUPGHOME}
gpg2 --list-secret-keys

kubectl create secret generic gpg1 --from-file=${GNUPGHOME} -o yaml --dry-run > 1.yaml
kubectl create secret generic gpg2 --from-file=${GNUPGHOME}/private-keys-v1.d -o yaml --dry-run > 2.yaml

mv ${GNUPGHOME} keys
tar -czvf keys.tar.gz keys
rm -fr keys

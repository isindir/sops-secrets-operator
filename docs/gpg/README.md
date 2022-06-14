# Preparing GPG keys

This procedure describes basic setup to use PGP keys with sops-secrets-operator.

## Create PGP keys

Run docker container in the directory of this README file:

```bash
docker run --rm -v $( pwd ):/tmp/scripts -ti ubuntu:20.04 bash
```

Then generate PGP keys inside container. PGP key files will remain in the folder
after closing container session:

```bash
cd /tmp/scripts/docs/gpg
./install.sh
make
```

Following files will be generated:

* `keys.tar.gz` - GPG configuration, which can be used to encrypt/decrypt
  secrets, however the better approach is to use user keys to encrypt secrets,
  allowing these keys to decrypt secrets within cluster.
* `1.yaml` and `2.yaml` - these files should be applied to the namespace where
  `sops-secrets-operator` will be deployed via helm chart.

Sourcing `keys-env` sets up working environment for data encryption:

```bash
source ./keys-env
```

After sourcing sops can be used to encrypt data, for example:

```bash
sops -e -p $FP --encrypted-suffix='Templates' ../../config/samples/isindir_v1alpha3_sopssecret.yaml > example-secrets.enc.yaml
```

Then `example-secrets.enc.yaml` can be applied to the cluster to create secrets using
sops CR. Resulting `keys.tar.gz`, `1.yaml` and `2.yaml` files should be kept secret
itself.

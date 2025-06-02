#!/usr/bin/env bash

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt install -y tree curl vim gnupg2 make
cd /usr/local/bin
curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl
chmod +x kubectl

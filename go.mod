module github.com/isindir/sops-secrets-operator

go 1.14

require (
	github.com/go-logr/logr v0.1.0
	github.com/hashicorp/vault/api v1.0.4
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onsi/ginkgo v1.15.2
	github.com/onsi/gomega v1.11.0
	github.com/sirupsen/logrus v1.8.1
	go.mozilla.org/sops/v3 v3.7.1
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.0
)

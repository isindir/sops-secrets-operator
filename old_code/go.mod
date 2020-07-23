module github.com/isindir/sops-secrets-operator

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.3
	github.com/operator-framework/operator-sdk v0.13.0
	github.com/spf13/pflag v1.0.5
	go.mozilla.org/sops/v3 v3.5.0
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.18.2
	k8s.io/gengo v0.0.0-20200114144118-36b2048a9120
	k8s.io/kube-openapi v0.0.0-20200121204235-bf4fb3bd569c
	sigs.k8s.io/controller-runtime v0.6.0
)

// Pinned to kubernetes-1.18.2
replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.18.2
	k8s.io/client-go => k8s.io/client-go v0.18.2 // Required by prometheus-operator
)

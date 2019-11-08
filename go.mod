module github.com/isindir/sops-secrets-operator

require (
	github.com/Azure/azure-sdk-for-go v30.0.0+incompatible // indirect
	github.com/aws/aws-sdk-go v1.19.41 // indirect
	github.com/dimchansky/utfbom v1.1.0 // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.0
	github.com/goware/prefixer v0.0.0-20160118172347-395022866408 // indirect
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c // indirect
	github.com/lib/pq v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/mozilla-services/yaml v0.0.0-20180922153656-28ffe5d0cafb // indirect
	github.com/operator-framework/operator-sdk v0.10.1-0.20190809191117-c1e2eae6580e
	github.com/spf13/pflag v1.0.3
	go.mozilla.org/gopgagent v0.0.0-20170926210634-4d7ea76ff71a // indirect
	go.mozilla.org/sops v0.0.0-20190611200209-e9e1e87723c8
	k8s.io/api v0.0.0-20190612125737-db0771252981
	k8s.io/apimachinery v0.0.0-20190612125636-6a5db36e93ad
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/code-generator v0.0.0-20181203235156-f8cba74510f3
	k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a
	k8s.io/kube-openapi v0.0.0-20190603182131-db7b694dc208
	sigs.k8s.io/controller-runtime v0.1.12
	sigs.k8s.io/controller-tools v0.1.10
)

// Pinned to kubernetes-1.13.4
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
)

replace (
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.29.0
	// Pinned to v2.9.2 (kubernetes-1.13.1) so https://proxy.golang.org can
	// resolve it correctly.
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.0.0-20190424153033-d3245f150225
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20180711000925-0cf8f7e6ed1d
	k8s.io/kube-state-metrics => k8s.io/kube-state-metrics v1.6.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.1.12
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.1.11-0.20190411181648-9d55346c2bde
)

replace github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.10.0

go 1.13

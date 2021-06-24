module github.com/isindir/sops-secrets-operator

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v31.2.0+incompatible
	github.com/aws/aws-sdk-go v1.37.18
	github.com/blang/semver v3.5.1+incompatible
	github.com/fatih/color v1.7.0
	github.com/go-logr/logr v0.4.0
	github.com/golang/protobuf v1.5.2
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/goware/prefixer v0.0.0-20160118172347-395022866408
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/lib/pq v1.2.0
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mozilla-services/yaml v0.0.0-20201007153854-c369669a6625
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.6.1
	go.mozilla.org/gopgagent v0.0.0-20170926210634-4d7ea76ff71a
	go.mozilla.org/sops v0.0.0-20190912205235-14a22d7a7060
	go.mozilla.org/sops/v3 v3.7.1
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	golang.org/x/net v0.0.0-20210428140749-89ef3d95e781
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.20.0
	google.golang.org/grpc v1.27.1
	gopkg.in/ini.v1 v1.51.0
	gopkg.in/urfave/cli.v1 v1.20.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	sigs.k8s.io/controller-runtime v0.8.3
)

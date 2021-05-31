module github.com/giantswarm/azure-operator/v5

go 1.15

require (
	github.com/Azure/azure-sdk-for-go v48.2.2+incompatible
	github.com/Azure/azure-storage-blob-go v0.11.0
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.7
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/coreos/go-semver v0.3.0
	github.com/giantswarm/apiextensions/v2 v2.6.2
	github.com/giantswarm/apiextensions/v3 v3.23.0
	github.com/giantswarm/badnodedetector v1.0.1
	github.com/giantswarm/certs/v3 v3.1.1
	github.com/giantswarm/conditions v0.3.0
	github.com/giantswarm/conditions-handler v0.2.1
	github.com/giantswarm/errors v0.3.0
	github.com/giantswarm/exporterkit v0.2.1
	github.com/giantswarm/ipam v0.3.0
	github.com/giantswarm/k8sclient/v5 v5.11.0
	github.com/giantswarm/k8scloudconfig/v10 v10.5.0
	github.com/giantswarm/kubelock/v2 v2.0.0
	github.com/giantswarm/microendpoint v0.2.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/microkit v0.2.2
	github.com/giantswarm/micrologger v0.5.0
	github.com/giantswarm/operatorkit/v4 v4.3.1
	github.com/giantswarm/tenantcluster/v3 v3.0.0
	github.com/giantswarm/to v0.3.0
	github.com/giantswarm/versionbundle v0.2.0
	github.com/golang/mock v1.4.4
	github.com/google/go-cmp v0.5.5
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/markbates/pkger v0.17.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pelletier/go-toml v1.7.0 // indirect
	github.com/prometheus/client_golang v1.10.0
	github.com/spf13/afero v1.5.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/viper v1.7.1
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	k8s.io/api v0.18.19
	k8s.io/apiextensions-apiserver v0.18.18
	k8s.io/apimachinery v0.18.19
	k8s.io/client-go v0.18.18
	sigs.k8s.io/cluster-api v0.3.16
	sigs.k8s.io/cluster-api-provider-azure v0.4.14
	sigs.k8s.io/controller-runtime v0.6.4
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Microsoft/hcsshim v0.8.7 => github.com/Microsoft/hcsshim v0.8.10
	github.com/coreos/etcd v3.3.13+incompatible => github.com/coreos/etcd v3.3.24+incompatible
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.13-gs
	sigs.k8s.io/cluster-api-provider-azure => github.com/giantswarm/cluster-api-provider-azure v0.4.12-gsalpha3
)

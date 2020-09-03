module github.com/giantswarm/azure-operator/v4

go 1.14

require (
	github.com/Azure/azure-sdk-for-go v45.0.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-autorest/autorest v0.11.2
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/coreos/go-semver v0.3.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/giantswarm/apiextensions v0.4.20
	github.com/giantswarm/appcatalog v0.1.11
	github.com/giantswarm/apprclient v0.2.0
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/certs/v2 v2.0.0
	github.com/giantswarm/e2e-harness v0.3.0
	github.com/giantswarm/e2eclients v0.2.0
	github.com/giantswarm/e2esetup v0.1.0
	github.com/giantswarm/e2etemplates v0.2.0
	github.com/giantswarm/e2etests v0.1.0
	github.com/giantswarm/errors v0.2.3
	github.com/giantswarm/helmclient v1.0.4
	github.com/giantswarm/ipam v0.2.0
	github.com/giantswarm/k8sclient v0.2.0
	github.com/giantswarm/k8sclient/v2 v2.0.0
	github.com/giantswarm/k8sclient/v3 v3.1.1
	github.com/giantswarm/k8scloudconfig/v7 v7.1.0
	github.com/giantswarm/kubelock v0.2.0
	github.com/giantswarm/microendpoint v0.2.0
	github.com/giantswarm/microerror v0.2.0
	github.com/giantswarm/microkit v0.2.0
	github.com/giantswarm/micrologger v0.3.1
	github.com/giantswarm/operatorkit v1.2.0
	github.com/giantswarm/randomkeys v0.2.0
	github.com/giantswarm/statusresource v0.4.0
	github.com/giantswarm/tenantcluster/v2 v2.0.0
	github.com/giantswarm/to v0.2.0
	github.com/giantswarm/versionbundle v0.2.0
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/google/go-cmp v0.5.1
	github.com/markbates/pkger v0.17.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pelletier/go-toml v1.7.0 // indirect
	github.com/spf13/afero v1.3.1
	github.com/spf13/viper v1.6.3
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.17.8
	k8s.io/apiextensions-apiserver v0.17.8
	k8s.io/apimachinery v0.17.8
	k8s.io/client-go v0.17.8
	sigs.k8s.io/cluster-api v0.3.8
	sigs.k8s.io/cluster-api-provider-azure v0.4.7
	sigs.k8s.io/controller-runtime v0.5.9
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.8
	k8s.io/kubernetes => k8s.io/kubernetes v1.17.8
)

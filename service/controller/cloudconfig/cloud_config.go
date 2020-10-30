package cloudconfig

import (
	"github.com/Azure/go-autorest/autorest/azure/auth"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v7/pkg/template"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/randomkeys"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

const (
	CertFilePermission          = 0400
	CloudProviderFilePermission = 0640
	FileOwnerUserName           = "root"
	FileOwnerGroupName          = "root"
	FileOwnerGroupIDNobody      = 65534
	FilePermission              = 0700
)

type Config struct {
	CertsSearcher      certs.Interface
	Logger             micrologger.Logger
	RandomkeysSearcher randomkeys.Interface

	Azure                  setting.Azure
	AzureClientCredentials auth.ClientCredentialsConfig
	Ignition               setting.Ignition
	DockerhubToken         string
	OIDC                   setting.OIDC
	RegistryMirrors        []string
	SSOPublicKey           string
	SubscriptionID         string
}

type CloudConfig struct {
	logger             micrologger.Logger
	randomkeysSearcher randomkeys.Interface

	azure                  setting.Azure
	azureClientCredentials auth.ClientCredentialsConfig
	dockerhubToken         string
	ignition               setting.Ignition
	OIDC                   setting.OIDC
	registryMirrors        []string
	ssoPublicKey           string
	subscriptionID         string
}

func New(config Config) (*CloudConfig, error) {
	if config.Ignition.Path == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.IgnitionPath must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.RandomkeysSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.RandomkeysSearcher must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	if config.AzureClientCredentials.ClientID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.azureClientCredentials must not be empty", config)
	}

	if config.DockerhubToken == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.DockerhubToken must not be empty", config)
	}

	if config.SubscriptionID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.SubscriptionID must not be empty", config)
	}

	c := &CloudConfig{
		logger:             config.Logger,
		randomkeysSearcher: config.RandomkeysSearcher,

		azure:                  config.Azure,
		azureClientCredentials: config.AzureClientCredentials,
		dockerhubToken:         config.DockerhubToken,
		ignition:               config.Ignition,
		OIDC:                   config.OIDC,
		registryMirrors:        config.RegistryMirrors,
		ssoPublicKey:           config.SSOPublicKey,
		subscriptionID:         config.SubscriptionID,
	}

	return c, nil
}

func (c CloudConfig) getEncryptionkey(customObject providerv1alpha1.AzureConfig) (string, error) {
	cluster, err := c.randomkeysSearcher.SearchCluster(key.ClusterID(&customObject))
	if err != nil {
		return "", microerror.Mask(err)
	}
	return string(cluster.APIServerEncryptionKey), nil
}

func newCloudConfig(template string, params k8scloudconfig.Params) (string, error) {
	c := k8scloudconfig.DefaultCloudConfigConfig()
	c.Params = params
	c.Template = template

	cloudConfig, err := k8scloudconfig.NewCloudConfig(c)
	if err != nil {
		return "", microerror.Mask(err)
	}
	err = cloudConfig.ExecuteTemplate()
	if err != nil {
		return "", microerror.Mask(err)
	}

	return cloudConfig.String(), nil
}

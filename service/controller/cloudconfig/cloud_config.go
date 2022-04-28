package cloudconfig

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	apiextensionslabels "github.com/giantswarm/apiextensions/v6/pkg/label"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v13/pkg/template"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/setting"
)

const (
	CertFilePermission          = 0400
	CloudProviderFilePermission = 0640
	FileOwnerUserName           = "root"
	FileOwnerGroupName          = "root"
	FileOwnerGroupIDNobody      = 65534
	FilePermission              = 0700
	// randomKeyLabel is the label used in the secret to identify a secret
	// containing the random key.
	randomKeyLabel      = "giantswarm.io/randomkey"
	randomKeyLabelValue = "encryption"
	secretKey           = "encryption"
)

type Key string

func (k Key) String() string {
	return string(k)
}

type Config struct {
	Azure                  setting.Azure
	AzureClientCredentials auth.ClientCredentialsConfig
	CtrlClient             ctrl.Client
	DockerhubToken         string
	Ignition               setting.Ignition
	Logger                 micrologger.Logger
	OIDC                   setting.OIDC
	RegistryMirrors        []string
	SSOPublicKey           string
	SubscriptionID         string
}

type CloudConfig struct {
	azure                  setting.Azure
	azureClientCredentials auth.ClientCredentialsConfig
	ctrlClient             ctrl.Client
	dockerhubToken         string
	ignition               setting.Ignition
	logger                 micrologger.Logger
	OIDC                   setting.OIDC
	registryMirrors        []string
	ssoPublicKey           string
	subscriptionID         string
}

func New(config Config) (*CloudConfig, error) {
	if config.DockerhubToken == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.DockerhubToken must not be empty", config)
	}
	if config.Ignition.Path == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.IgnitionPath must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if config.AzureClientCredentials.ClientID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.azureClientCredentials must not be empty", config)
	}
	if config.SubscriptionID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.SubscriptionID must not be empty", config)
	}

	c := &CloudConfig{
		azure:                  config.Azure,
		azureClientCredentials: config.AzureClientCredentials,
		ctrlClient:             config.CtrlClient,
		dockerhubToken:         config.DockerhubToken,
		ignition:               config.Ignition,
		logger:                 config.Logger,
		OIDC:                   config.OIDC,
		registryMirrors:        config.RegistryMirrors,
		ssoPublicKey:           config.SSOPublicKey,
		subscriptionID:         config.SubscriptionID,
	}

	return c, nil
}

func (c CloudConfig) getEncryptionkey(ctx context.Context, customObject providerv1alpha1.AzureConfig) (string, error) {
	secretList := &corev1.SecretList{}
	{
		c.logger.Debugf(ctx, "try to find encryption secret")
		err := c.ctrlClient.List(
			ctx,
			secretList,
			ctrl.MatchingLabels{
				randomKeyLabel:              randomKeyLabelValue,
				apiextensionslabels.Cluster: key.ClusterID(&customObject),
			},
		)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	if len(secretList.Items) > 0 {
		c.logger.Debugf(ctx, "found encryption secret in namespace '%s/%s'", secretList.Items[0].Namespace, secretList.Items[0].Name)
		randomkey, ok := secretList.Items[0].Data[secretKey]
		if !ok {
			return "", microerror.Maskf(invalidSecretError, "%q key missing", secretKey)
		}

		return string(randomkey), nil
	}

	return "", microerror.Mask(secretNotFoundError)

}

func newCloudConfig(template string, params k8scloudconfig.Params) (string, error) {
	c := k8scloudconfig.CloudConfigConfig{
		Params:   params,
		Template: template,
	}
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

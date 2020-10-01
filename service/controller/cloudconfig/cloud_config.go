package cloudconfig

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	apiextensionslabels "github.com/giantswarm/apiextensions/v2/pkg/label"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v8/pkg/template"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

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
	// randomKeyLabel is the label used in the secret to identify a secret
	// containing the random key.
	randomKeyLabel = "giantswarm.io/randomkey"
	// clusterLabel is the label used in the secret to identify a secret
	// containing the random key.
	clusterLabel = "giantswarm.io/cluster"
)

type Key string

func (k Key) String() string {
	return string(k)
}

const (
	EncryptionKey Key = "encryption"
)

type RandomKey []byte

type Cluster struct {
	APIServerEncryptionKey RandomKey
}

type Config struct {
	Azure                  setting.Azure
	AzureClientCredentials auth.ClientCredentialsConfig
	CtrlClient             ctrl.Client
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
	ignition               setting.Ignition
	logger                 micrologger.Logger
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
	cluster, err := SearchCluster(ctx, c.ctrlClient, key.ClusterID(&customObject), key.OrganizationNamespace(&customObject))
	if err != nil {
		return "", microerror.Mask(err)
	}
	return string(cluster.APIServerEncryptionKey), nil
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

func SearchCluster(ctx context.Context, ctrlClient ctrl.Client, clusterID, namespace string) (Cluster, error) {
	var cluster Cluster

	keys := []struct {
		RandomKey *RandomKey
		Type      Key
	}{
		{RandomKey: &cluster.APIServerEncryptionKey, Type: EncryptionKey},
	}

	for _, k := range keys {
		err := search(ctx, ctrlClient, k.RandomKey, clusterID, namespace, k.Type)
		if err != nil {
			return Cluster{}, microerror.Mask(err)
		}
	}

	return cluster, nil
}

func search(ctx context.Context, ctrlClient ctrl.Client, randomKey *RandomKey, clusterID, namespace string, key Key) error {
	secretList := &corev1.SecretList{}
	{
		err := ctrlClient.List(
			ctx,
			secretList,
			ctrl.InNamespace(namespace),
			ctrl.MatchingLabels{
				randomKeyLabel:              key.String(),
				apiextensionslabels.Cluster: clusterID,
			},
		)
		if err != nil {
			return microerror.Mask(err)
		}

		if secretList.Size() < 1 {
			err := ctrlClient.List(
				ctx,
				secretList,
				ctrl.InNamespace(corev1.NamespaceDefault),
				ctrl.MatchingLabels{
					randomKeyLabel:              key.String(),
					apiextensionslabels.Cluster: clusterID,
				},
			)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	if secretList.Size() > 0 {
		err := fillRandomKeyFromSecret(randomKey, (*secretList).Items[0], clusterID, key)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return microerror.Mask(timeoutError)
}

func fillRandomKeyFromSecret(randomkey *RandomKey, secret corev1.Secret, clusterID string, key Key) error {
	gotClusterID := secret.Labels[clusterLabel]
	if clusterID != gotClusterID {
		return microerror.Maskf(invalidSecretError, "expected clusterID = %q, got %q", clusterID, gotClusterID)
	}
	gotKeys := secret.Labels[randomKeyLabel]
	if string(key) != gotKeys {
		return microerror.Maskf(invalidSecretError, "expected random key = %q, got %q", key, gotKeys)
	}
	var ok bool
	if *randomkey, ok = secret.Data[string(EncryptionKey)]; !ok {
		return microerror.Maskf(invalidSecretError, "%q key missing", EncryptionKey)
	}

	return nil
}

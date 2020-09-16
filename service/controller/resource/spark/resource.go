package spark

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/randomkeys/v2"

	"github.com/giantswarm/azure-operator/v4/pkg/credential"
	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/azureconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"

	"github.com/giantswarm/certs/v3/pkg/certs"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Name = "spark"

	credentialDefaultName = "credential-default"
	credentialNamespace   = "giantswarm"
)

type Config struct {
	APIServerSecurePort int
	Azure               setting.Azure
	Calico              azureconfig.CalicoConfig
	CertsSearcher       certs.Interface
	ClusterIPRange      string
	CredentialProvider  credential.Provider
	CtrlClient          client.Client
	EtcdPrefix          string
	Ignition            setting.Ignition
	Logger              micrologger.Logger
	OIDC                setting.OIDC
	RandomKeysSearcher  randomkeys.Interface
	RegistryDomain      string
	SSHUserList         string
	SSOPublicKey        string
}

// Resource simulates a CAPI Bootstrap provider by rendering cloudconfig files while watching Spark CRs.
type Resource struct {
	apiServerSecurePort int
	azure               setting.Azure
	calico              azureconfig.CalicoConfig
	certsSearcher       certs.Interface
	clusterIPRange      string
	credentialProvider  credential.Provider
	ctrlClient          client.Client
	etcdPrefix          string
	ignition            setting.Ignition
	logger              micrologger.Logger
	oidc                setting.OIDC
	randomKeysSearcher  randomkeys.Interface
	registryDomain      string
	sshUserList         string
	ssoPublicKey        string
}

func New(config Config) (*Resource, error) {
	if config.APIServerSecurePort == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.APIServerSecurePort must not be empty", config)
	}

	err := config.Azure.Validate()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if config.Calico.MTU == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.Calico.MTU must not be empty", config)
	}

	if config.CertsSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CertsSearcher must not be empty", config)
	}

	if config.ClusterIPRange == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClusterIPRange must not be empty", config)
	}

	if config.CredentialProvider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CredentialProvider must not be empty", config)
	}

	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}

	if config.EtcdPrefix == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.EtcdPrefix must not be empty", config)
	}

	if config.Ignition.Path == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Ignition.Path must not be empty", config)
	}

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.RandomKeysSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.RandomKeysSearcher must not be empty", config)
	}

	if config.RegistryDomain == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.RegistryDomain must not be empty", config)
	}

	if config.SSHUserList == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.SSHUserList must not be empty", config)
	}

	if config.SSOPublicKey == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.SSOPublicKey must not be empty", config)
	}

	r := &Resource{
		apiServerSecurePort: config.APIServerSecurePort,
		azure:               config.Azure,
		calico:              config.Calico,
		certsSearcher:       config.CertsSearcher,
		clusterIPRange:      config.ClusterIPRange,
		credentialProvider:  config.CredentialProvider,
		ctrlClient:          config.CtrlClient,
		etcdPrefix:          config.EtcdPrefix,
		ignition:            config.Ignition,
		logger:              config.Logger,
		oidc:                config.OIDC,
		randomKeysSearcher:  config.RandomKeysSearcher,
		registryDomain:      config.RegistryDomain,
		sshUserList:         config.SSHUserList,
		ssoPublicKey:        config.SSOPublicKey,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) toEncrypterObject(ctx context.Context, secretName string) (encrypter.Interface, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "retrieving encryptionkey")

	secret := &corev1.Secret{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: key.CertificateEncryptionNamespace, Name: secretName}, secret)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var enc *encrypter.Encrypter
	{
		if _, ok := secret.Data[key.CertificateEncryptionKeyName]; !ok {
			return nil, microerror.Maskf(invalidConfigError, "encryption key not found in secret %q", secret.Name)
		}
		if _, ok := secret.Data[key.CertificateEncryptionIVName]; !ok {
			return nil, microerror.Maskf(invalidConfigError, "encryption iv not found in secret %q", secret.Name)
		}
		c := encrypter.Config{
			Key: secret.Data[key.CertificateEncryptionKeyName],
			IV:  secret.Data[key.CertificateEncryptionIVName],
		}

		enc, err = encrypter.New(c)
		if err != nil {
			return nil, microerror.Mask(err)

		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "encryptionkey found")

	return enc, nil
}

func (r *Resource) getCredentialSecret(ctx context.Context, azureMachinePool key.LabelsGetter) (*v1alpha1.CredentialSecret, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding credential secret")

	organization, exists := azureMachinePool.GetLabels()[label.Organization]
	if !exists {
		return nil, microerror.Mask(missingOrganizationLabel)
	}

	secretList := &corev1.SecretList{}
	{
		err := r.ctrlClient.List(
			ctx,
			secretList,
			client.InNamespace(credentialNamespace),
			client.MatchingLabels{
				label.App:          "credentiald",
				label.Organization: organization,
			},
		)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// We currently only support one credential secret per organization.
	// If there are more than one, return an error.
	if len(secretList.Items) > 1 {
		return nil, microerror.Mask(tooManyCredentialsError)
	}

	// If one credential secret is found, we use that.
	if len(secretList.Items) == 1 {
		secret := secretList.Items[0]

		credentialSecret := &v1alpha1.CredentialSecret{
			Namespace: secret.Namespace,
			Name:      secret.Name,
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found credential secret %s/%s", credentialSecret.Namespace, credentialSecret.Name))

		return credentialSecret, nil
	}

	// If no credential secrets are found, we use the default.
	credentialSecret := &v1alpha1.CredentialSecret{
		Namespace: credentialNamespace,
		Name:      credentialDefaultName,
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "did not find credential secret, using default secret")

	return credentialSecret, nil
}

func secretName(base string) string {
	return base + "-machine-pool-ignition"
}

package spark

import (
	"context"

	"github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	apiextensionslabels "github.com/giantswarm/apiextensions/v6/pkg/label"
	"github.com/giantswarm/certs/v4/pkg/certs"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v5/pkg/credential"
	"github.com/giantswarm/azure-operator/v5/pkg/employees"
	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/setting"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Name = "spark"

	credentialDefaultName      = "credential-default" // nolint:gosec
	credentialDefaultNamespace = "giantswarm"
)

type Config struct {
	APIServerSecurePort int
	Azure               setting.Azure
	CalicoCIDRSize      int
	CalicoMTU           int
	CalicoSubnet        string
	CertsSearcher       certs.Interface
	ClusterIPRange      string
	CredentialProvider  credential.Provider
	CtrlClient          client.Client
	DockerhubToken      string
	EtcdPrefix          string
	Ignition            setting.Ignition
	Logger              micrologger.Logger
	OIDC                setting.OIDC
	RegistryDomain      string
	RegistryMirrors     []string
	SSHUserList         employees.SSHUserList
	SSOPublicKey        string
}

// Resource simulates a CAPI Bootstrap provider by rendering cloudconfig files while watching Spark CRs.
type Resource struct {
	apiServerSecurePort int
	azure               setting.Azure
	calicoCIDRSize      int
	calicoMTU           int
	calicoSubnet        string
	certsSearcher       certs.Interface
	clusterIPRange      string
	credentialProvider  credential.Provider
	ctrlClient          client.Client
	dockerhubToken      string
	etcdPrefix          string
	ignition            setting.Ignition
	logger              micrologger.Logger
	oidc                setting.OIDC
	registryDomain      string
	registryMirrors     []string
	sshUserList         employees.SSHUserList
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

	if config.CalicoMTU == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.CalicoMTU must not be empty", config)
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

	if config.DockerhubToken == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.DockerhubToken must not be empty", config)
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

	if config.RegistryDomain == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.RegistryDomain must not be empty", config)
	}

	if config.SSOPublicKey == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.SSOPublicKey must not be empty", config)
	}

	r := &Resource{
		apiServerSecurePort: config.APIServerSecurePort,
		azure:               config.Azure,
		calicoCIDRSize:      config.CalicoCIDRSize,
		calicoMTU:           config.CalicoMTU,
		calicoSubnet:        config.CalicoSubnet,
		certsSearcher:       config.CertsSearcher,
		clusterIPRange:      config.ClusterIPRange,
		credentialProvider:  config.CredentialProvider,
		ctrlClient:          config.CtrlClient,
		dockerhubToken:      config.DockerhubToken,
		etcdPrefix:          config.EtcdPrefix,
		ignition:            config.Ignition,
		logger:              config.Logger,
		oidc:                config.OIDC,
		registryDomain:      config.RegistryDomain,
		registryMirrors:     config.RegistryMirrors,
		sshUserList:         config.SSHUserList,
		ssoPublicKey:        config.SSOPublicKey,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) toEncrypterObject(ctx context.Context, secretName string) (encrypter.Interface, error) {
	r.logger.Debugf(ctx, "retrieving encryptionkey")

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

	r.logger.Debugf(ctx, "encryptionkey found")

	return enc, nil
}

func secretName(base string) string {
	return base + "-machine-pool-ignition"
}

func (r *Resource) getCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error) {
	r.logger.Debugf(ctx, "finding credential secret")

	var err error
	var credentialSecret *v1alpha1.CredentialSecret

	credentialSecret, err = r.getOrganizationCredentialSecret(ctx, objectMeta)
	if IsCredentialsNotFoundError(err) {
		credentialSecret, err = r.getLegacyCredentialSecret(ctx, objectMeta)
		if IsCredentialsNotFoundError(err) {
			r.logger.Debugf(ctx, "did not find credential secret, using default '%s/%s'", credentialDefaultNamespace, credentialDefaultName)
			return &v1alpha1.CredentialSecret{
				Namespace: credentialDefaultNamespace,
				Name:      credentialDefaultName,
			}, nil
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return credentialSecret, nil
}

// getOrganizationCredentialSecret tries to find a Secret in the organization namespace.
func (r *Resource) getOrganizationCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error) {
	r.logger.Debugf(ctx, "try in namespace %#q filtering by organization %#q", objectMeta.Namespace, key.OrganizationID(&objectMeta))
	secretList := &corev1.SecretList{}
	{
		err := r.ctrlClient.List(
			ctx,
			secretList,
			client.InNamespace(objectMeta.Namespace),
			client.MatchingLabels{
				label.App:                        "credentiald",
				apiextensionslabels.Organization: key.OrganizationID(&objectMeta),
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

	if len(secretList.Items) < 1 {
		return nil, microerror.Mask(credentialsNotFoundError)
	}

	// If one credential secret is found, we use that.
	secret := secretList.Items[0]

	credentialSecret := &v1alpha1.CredentialSecret{
		Namespace: secret.Namespace,
		Name:      secret.Name,
	}

	r.logger.Debugf(ctx, "found credential secret %s/%s", credentialSecret.Namespace, credentialSecret.Name)

	return credentialSecret, nil
}

// getLegacyCredentialSecret tries to find a Secret in the default credentials namespace but labeled with the organization name.
// This is needed while we migrate everything to the org namespace and org credentials are created in the org namespace instead of the default namespace.
func (r *Resource) getLegacyCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error) {
	r.logger.Debugf(ctx, "try in namespace %#q filtering by organization %#q", objectMeta.Namespace, key.OrganizationID(&objectMeta))
	secretList := &corev1.SecretList{}
	{
		err := r.ctrlClient.List(
			ctx,
			secretList,
			client.InNamespace(credentialDefaultNamespace),
			client.MatchingLabels{
				label.App:                        "credentiald",
				apiextensionslabels.Organization: key.OrganizationID(&objectMeta),
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

	if len(secretList.Items) < 1 {
		return nil, microerror.Mask(credentialsNotFoundError)
	}

	// If one credential secret is found, we use that.
	secret := secretList.Items[0]

	credentialSecret := &v1alpha1.CredentialSecret{
		Namespace: secret.Namespace,
		Name:      secret.Name,
	}

	r.logger.Debugf(ctx, "found credential secret %s/%s", credentialSecret.Namespace, credentialSecret.Name)

	return credentialSecret, nil
}

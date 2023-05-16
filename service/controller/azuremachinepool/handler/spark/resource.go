package spark

import (
	"context"

	"github.com/giantswarm/certs/v4/pkg/certs"

	v5client "github.com/giantswarm/azure-operator/v8/client"
	"github.com/giantswarm/azure-operator/v8/pkg/credential"
	"github.com/giantswarm/azure-operator/v8/pkg/employees"
	"github.com/giantswarm/azure-operator/v8/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v8/service/controller/key"
	"github.com/giantswarm/azure-operator/v8/service/controller/setting"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Name = "spark"
)

type Config struct {
	APIServerSecurePort int
	Azure               setting.Azure
	CalicoCIDRSize      int
	CalicoMTU           int
	CalicoSubnet        string
	CertsSearcher       certs.Interface
	ClientFactory       v5client.OrganizationFactory
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
	clientFactory       v5client.OrganizationFactory
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
		clientFactory:       config.ClientFactory,
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

package blobobject

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	"github.com/giantswarm/certs/v3/pkg/certs"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialsawarefactory"
	"github.com/giantswarm/azure-operator/v5/pkg/employees"
	"github.com/giantswarm/azure-operator/v5/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "blobobject"
)

type Config struct {
	CertsSearcher         certs.Interface
	CtrlClient            client.Client
	G8sClient             versioned.Interface
	K8sClient             kubernetes.Interface
	Logger                micrologger.Logger
	RegistryDomain        string
	SSHUserList           employees.SSHUserList
	StorageAccountsClient *storage.AccountsClient
	WCAzureClientFactory  credentialsawarefactory.Interface
}

type Resource struct {
	certsSearcher        certs.Interface
	ctrlClient           client.Client
	g8sClient            versioned.Interface
	k8sClient            kubernetes.Interface
	logger               micrologger.Logger
	registryDomain       string
	sshUserList          employees.SSHUserList
	wcAzureClientFactory credentialsawarefactory.Interface
}

func New(config Config) (*Resource, error) {
	if config.CertsSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CertsSearcher must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.RegistryDomain == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.RegistryDomain must not be empty", config)
	}
	if config.WCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCAzureClientFactory must not be empty", config)
	}

	r := &Resource{
		certsSearcher:        config.CertsSearcher,
		ctrlClient:           config.CtrlClient,
		g8sClient:            config.G8sClient,
		k8sClient:            config.K8sClient,
		logger:               config.Logger,
		registryDomain:       config.RegistryDomain,
		sshUserList:          config.SSHUserList,
		wcAzureClientFactory: config.WCAzureClientFactory,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) toEncrypterObject(ctx context.Context, secretName string) (encrypter.Interface, error) {
	r.logger.Debugf(ctx, "retrieving encryptionkey")

	secret, err := r.k8sClient.CoreV1().Secrets(key.CertificateEncryptionNamespace).Get(ctx, secretName, metav1.GetOptions{})
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

	return enc, nil
}

func toContainerObjectState(v interface{}) ([]ContainerObjectState, error) {
	if v == nil {
		return nil, nil
	}

	containerObjectState, ok := v.([]ContainerObjectState)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", containerObjectState, v)
	}

	return containerObjectState, nil
}

func objectInSliceByKey(obj ContainerObjectState, list []ContainerObjectState) bool {
	for _, item := range list {
		if obj.Key == item.Key {
			return true
		}
	}
	return false
}

func objectInSliceByKeyAndBody(obj ContainerObjectState, list []ContainerObjectState) bool {
	for _, item := range list {
		if obj.Key == item.Key && obj.Body == item.Body {
			return true
		}
	}
	return false
}

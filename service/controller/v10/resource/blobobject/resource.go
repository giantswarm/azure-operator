package blobobject

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/service/controller/v10/encrypter"
	"github.com/giantswarm/azure-operator/service/controller/v10/key"
)

const (
	// Name is the identifier of the resource.
	Name = "blobobjectv10"
)

type Config struct {
	CertsSearcher         certs.Interface
	K8sClient             kubernetes.Interface
	Logger                micrologger.Logger
	StorageAccountsClient *storage.AccountsClient
}

type Resource struct {
	certsSearcher certs.Interface
	k8sClient     kubernetes.Interface
	logger        micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CertsSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CertsSearcher must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		certsSearcher: config.CertsSearcher,
		k8sClient:     config.K8sClient,
		logger:        config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) toEncrypterObject(ctx context.Context, secretName string) (encrypter.Interface, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "retrieving encryptionkey")

	secret, err := r.k8sClient.CoreV1().Secrets(key.CertificateEncryptionNamespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var enc *encrypter.Encrypter
	{
		if _, ok := secret.Data[key.CertificateEncryptionKeyName]; !ok {
			return nil, microerror.Maskf(invalidConfigError, "encryption key not found in secret", secret.Name)
		}
		if _, ok := secret.Data[key.CertificateEncryptionIVName]; !ok {
			return nil, microerror.Maskf(invalidConfigError, "encryption iv not found in secret", secret.Name)
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

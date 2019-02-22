package instance

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v5/debugger"
	enc "github.com/giantswarm/azure-operator/service/controller/v5/encrypter"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
)

const (
	Name = "instancev5"
)

type Config struct {
	Debugger  *debugger.Debugger
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	Azure           setting.Azure
	TemplateVersion string
}

type Resource struct {
	debugger  *debugger.Debugger
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	azure           setting.Azure
	templateVersion string
}

func New(config Config) (*Resource, error) {
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
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

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.TemplateVersion must not be empty", config)
	}

	r := &Resource{
		debugger:  config.Debugger,
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		azure:           config.Azure,
		templateVersion: config.TemplateVersion,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getDeploymentsClient(ctx context.Context) (*azureresource.DeploymentsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.DeploymentsClient, nil
}

func (r *Resource) getScaleSetsClient(ctx context.Context) (*compute.VirtualMachineScaleSetsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualMachineScaleSetsClient, nil
}

func (r *Resource) getStorageAccountsClient(ctx context.Context) (*storage.AccountsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.StorageAccountsClient, nil
}

func (r *Resource) getVMsClient(ctx context.Context) (*compute.VirtualMachineScaleSetVMsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualMachineScaleSetVMsClient, nil
}

func (r *Resource) getEncrypterObject(ctx context.Context, secretName string) (enc.Encrypter, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "retrieving encryptionkey")

	secret, err := r.k8sClient.CoreV1().Secrets(key.CertificateEncryptionNamespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return enc.Encrypter{}, microerror.Mask(err)
	}

	var encrypter enc.Encrypter
	{
		if _, ok := secret.Data[key.CertificateEncryptionKeyName]; !ok {
			return enc.Encrypter{}, microerror.Maskf(invalidConfigError, "encryption key not found in secret", secret.Name)
		}
		if _, ok := secret.Data[key.CertificateEncryptionIVName]; !ok {
			return enc.Encrypter{}, microerror.Maskf(invalidConfigError, "encryption iv not found in secret", secret.Name)
		}
		c := enc.Config{
			Key: secret.Data[key.CertificateEncryptionKeyName],
			IV:  secret.Data[key.CertificateEncryptionIVName],
		}

		encrypter, err = enc.New(c)
		if err != nil {
			return enc.Encrypter{}, microerror.Mask(err)

		}
	}

	return encrypter, nil
}

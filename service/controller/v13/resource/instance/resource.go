package instance

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/tenantcluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v13/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v13/debugger"
	"github.com/giantswarm/azure-operator/service/controller/v13/encrypter"
	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
)

const (
	Name = "instancev13"
)

type Config struct {
	Debugger  *debugger.Debugger
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
	Tenant    tenantcluster.Interface

	Azure           setting.Azure
	TemplateVersion string
}

type Resource struct {
	debugger        *debugger.Debugger
	g8sClient       versioned.Interface
	k8sClient       kubernetes.Interface
	logger          micrologger.Logger
	stateMachine    state.Machine
	tenant          tenantcluster.Interface
	tenantK8sClient kubernetes.Interface

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
	if config.Tenant == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Tenant must not be empty", config)
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
		tenant:    config.Tenant,

		azure:           config.Azure,
		templateVersion: config.TemplateVersion,
	}

	r.configureStateMachine()

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

func (r *Resource) getEncrypterObject(ctx context.Context, secretName string) (encrypter.Interface, error) {
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

func (r *Resource) getK8sClient(ctx context.Context, obj interface{}) (kubernetes.Interface, error) {
	if r.tenantK8sClient != nil {
		return r.tenantK8sClient, nil
	}

	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var restConfig *rest.Config
	{
		restConfig, err = r.tenant.NewRestConfig(ctx, key.ClusterID(customObject), key.ClusterAPIEndpoint(customObject))
		if tenantcluster.IsTimeout(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "timeout fetching tenant cluster certificates")
			return nil, microerror.Mask(err)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var k8sClient k8sclient.Interface
	{
		c := k8sclient.ClientsConfig{
			Logger:     r.logger,
			RestConfig: rest.CopyConfig(restConfig),
		}

		k8sClient, err = k8sclient.NewClients(c)
		if tenant.IsAPINotAvailable(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available yet")
			return nil, microerror.Mask(err)

		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r.tenantK8sClient = k8sClient.K8sClient()

	return r.tenantK8sClient, nil
}

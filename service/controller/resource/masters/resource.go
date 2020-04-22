package masters

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/service/controller/debugger"
	"github.com/giantswarm/azure-operator/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/controller/resource/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/resource/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

const (
	Name = "masters"
)

type Config struct {
	Debugger  *debugger.Debugger
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	Azure            setting.Azure
	TemplateVersion  string
	VMSSCheckWorkers int
}

type Resource struct {
	debugger     *debugger.Debugger
	g8sClient    versioned.Interface
	k8sClient    kubernetes.Interface
	logger       micrologger.Logger
	stateMachine state.Machine

	azure            setting.Azure
	instanceWatchdog vmsscheck.InstanceWatchdog
	templateVersion  string
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

	var iwd vmsscheck.InstanceWatchdog
	{
		c := vmsscheck.Config{
			Logger:     config.Logger,
			NumWorkers: config.VMSSCheckWorkers,
		}

		var err error
		iwd, err = vmsscheck.NewInstanceWatchdog(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r := &Resource{
		debugger:  config.Debugger,
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		azure:            config.Azure,
		instanceWatchdog: iwd,
		templateVersion:  config.TemplateVersion,
	}

	r.configureStateMachine()

	return r, nil
}

func (r *Resource) Name() string {
	return Name
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

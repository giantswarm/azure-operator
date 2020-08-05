package nodes

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"

	"github.com/giantswarm/azure-operator/v4/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

type Config struct {
	Debugger  *debugger.Debugger
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	Azure            setting.Azure
	ClientFactory    *client.Factory
	InstanceWatchdog vmsscheck.InstanceWatchdog
	Name             string
}

type Resource struct {
	Debugger     *debugger.Debugger
	G8sClient    versioned.Interface
	k8sClient    kubernetes.Interface
	Logger       micrologger.Logger
	StateMachine state.Machine

	Azure            setting.Azure
	ClientFactory    *client.Factory
	InstanceWatchdog vmsscheck.InstanceWatchdog
	name             string
}

func New(config Config) (*Resource, error) {
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.InstanceWatchdog == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.InstanceWatchdog must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.ClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClientFactory must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	if len(config.Name) == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.Name must not be empty", config)
	}

	r := &Resource{
		Debugger:  config.Debugger,
		G8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		Logger:    config.Logger,

		Azure:            config.Azure,
		ClientFactory:    config.ClientFactory,
		InstanceWatchdog: config.InstanceWatchdog,
		name:             config.Name,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return r.name
}

func (r *Resource) SetStateMachine(stateMachine state.Machine) {
	r.StateMachine = stateMachine
}

func (r *Resource) GetEncrypterObject(ctx context.Context, secretName string) (encrypter.Interface, error) {
	r.Logger.LogCtx(ctx, "level", "debug", "message", "retrieving encryptionkey")

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

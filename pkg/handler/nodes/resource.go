package nodes

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/client"
	"github.com/giantswarm/azure-operator/v8/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v8/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v8/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v8/service/controller/key"
	"github.com/giantswarm/azure-operator/v8/service/controller/setting"
)

type Config struct {
	CtrlClient ctrlclient.Client
	Debugger   *debugger.Debugger
	Logger     micrologger.Logger

	Azure         setting.Azure
	ClientFactory client.OrganizationFactory
	Name          string
}

type Resource struct {
	CtrlClient   ctrlclient.Client
	Debugger     *debugger.Debugger
	Logger       micrologger.Logger
	StateMachine state.Machine

	Azure         setting.Azure
	ClientFactory client.OrganizationFactory
	name          string
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	if len(config.Name) == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.Name must not be empty", config)
	}

	r := &Resource{
		CtrlClient: config.CtrlClient,
		Debugger:   config.Debugger,
		Logger:     config.Logger,

		Azure:         config.Azure,
		ClientFactory: config.ClientFactory,
		name:          config.Name,
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
	r.Logger.Debugf(ctx, "retrieving encryptionkey")

	secret := &v1.Secret{}
	err := r.CtrlClient.Get(ctx, ctrlclient.ObjectKey{Namespace: key.CertificateEncryptionNamespace, Name: secretName}, secret)
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

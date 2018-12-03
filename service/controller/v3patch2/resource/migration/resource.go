// Package migration provides an operatorkit resource that migrates azureconfig CRs
// to reference the default credential secret if they do not already.
// It can be safely removed once all azureconfig CRs reference a credential secret.
package migration

import (
	"context"
	"reflect"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/v3patch2/key"
)

const (
	name = "migrationv3patch2"

	azureConfigNamespace             = "default"
	credentialSecretDefaultNamespace = "giantswarm"
	credentialSecretDefaultName      = "credential-default"
)

type Config struct {
	G8sClient versioned.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	g8sClient versioned.Interface
	logger    micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		g8sClient: config.G8sClient,
		logger:    config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return name
}

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	customObject = *customObject.DeepCopy()

	oldSpec := *customObject.Spec.DeepCopy()

	if customObject.Spec.Azure.CredentialSecret.Name != "" {
		r.logger.LogCtx(ctx, "level", "debug", "message", "CR is missing credential, setting the default")

		customObject.Spec.Azure.CredentialSecret.Namespace = credentialSecretDefaultNamespace
		customObject.Spec.Azure.CredentialSecret.Name = credentialSecretDefaultName
	}

	if reflect.DeepEqual(customObject.Spec, oldSpec) {
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "updating CR")

	_, err = r.g8sClient.ProviderV1alpha1().AzureConfigs(azureConfigNamespace).Update(&customObject)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "updated CR")
	reconciliationcanceledcontext.SetCanceled(ctx)
	r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")

	return nil
}

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

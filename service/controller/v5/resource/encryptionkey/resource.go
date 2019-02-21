package encryptionkey

import (
	"context"

	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v5/encrypter"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"
)

const (
	keySize = 32
	Name    = "encryptionkeyv5"
)

type Config struct {
	K8sClient   kubernetes.Interface
	Logger      micrologger.Logger
	ProjectName string
}

type Resource struct {
	k8sClient   kubernetes.Interface
	logger      micrologger.Logger
	projectName string
}

func New(config Config) (*Resource, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ProjectName must not be empty", config)
	}

	newResource := &Resource{
		k8sClient:   config.K8sClient,
		logger:      config.Logger,
		projectName: config.ProjectName,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) addEncrypterToContext(ctx context.Context, encKey, encIV []byte) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "setting encrypter to context")

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	c := encrypter.Config{
		Key: encKey,
		IV:  encIV,
	}

	enc, err := encrypter.New(c)
	if err != nil {
		return microerror.Mask(err)
	}

	cc.Encrypter = &enc

	r.logger.LogCtx(ctx, "level", "debug", "message", "set encrypter to context")

	return nil
}

package env

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

// Env should be created using Builder.
type Env struct {
	Azure   Azure
	Common  Common
	Cluster Cluster
}

type BuilderConfig struct {
	Logger micrologger.Logger
}

type Builder struct {
	logger micrologger.Logger
}

func NewBuilder(config BuilderConfig) (*Builder, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	b := &Bilder{
		logger: config.Logger,
	}

	return b, nil
}

func (b *Builder) Build(ctx context.Context) (Env, error) {
	var common Common
	{
		// ...
	}

	var cluster Cluster
	{
		// ...
	}

	var azure Azure
	{
		c := azureBuilderConfig{
			Logger: b.logger,

			CircleBuildNumber: common.Circle.BuildNumber,
		}

		builder, err := newAzureBuilder(c)
		if err != nil {
			return Env{}, microerror.Mask(err)
		}

		azure, err = builder.Build(ctx)
		if err != nil {
			return Env{}, microerror.Mask(err)
		}
	}

	e := Env{
		Azure:   azure,
		Common:  common,
		Cluster: cluster,
	}

	return e, nil
}

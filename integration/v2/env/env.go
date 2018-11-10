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

	b := &Builder{
		logger: config.Logger,
	}

	return b, nil
}

func (b *Builder) Build(ctx context.Context) (Env, error) {
	var common Common
	{
		c := commonBuilderConfig{
			Logger: b.logger,
		}

		builder, err := newCommonBuilder(c)
		if err != nil {
			return Env{}, microerror.Mask(err)
		}

		common, err = builder.Build(ctx)
		if err != nil {
			return Env{}, microerror.Mask(err)
		}
	}

	var version Version
	{
		c := versionBuilderConfig{
			Logger: b.logger,
		}

		builder, err := newVersionBuilder(c)
		if err != nil {
			return Env{}, microerror.Mask(err)
		}

		version, err = builder.Build(ctx)
	}

	var cluster Cluster
	{
		c := clusterBuilderConfig{
			Logger: b.logger,

			CircleSHA:     common.CircleSHA,
			TestDir:       common.TestDir,
			TestedVersion: version.TestedVersion,
		}

		builder, err := newClusterBuilder(c)
		if err != nil {
			return Env{}, microerror.Mask(err)
		}

		cluster, err = builder.Build(ctx)
	}

	var azure Azure
	{
		c := azureBuilderConfig{
			Logger: b.logger,

			CircleBuildNumber: common.CircleBuildNumber,
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

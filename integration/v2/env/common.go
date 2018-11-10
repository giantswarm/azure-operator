package env

import (
	"context"
	"strconv"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	component = "azure-operator"
	provider  = "azure"
)

const (
	EnvVarCircleBuildNum     = "CIRCLE_BUILD_NUM"
	EnvVarCircleCI           = "CIRCLECI"
	EnvVarCircleSHA1         = "CIRCLE_SHA1"
	EnvVarRegistryPullSecret = "REGISTRY_PULL_SECRET"
	EnvVarTestDir            = "TEST_DIR"
)

type Common struct {
	CircleBuildNumber  uint
	CircleCI           bool
	CircleSHA          string
	RegistryPullSecret string
	TestDir            string
}

type commonBuilderConfig struct {
	Logger micrologger.Logger
}

type commonBuilder struct {
	logger micrologger.Logger
}

func newCommonBuilder(config commonBuilderConfig) (*commonBuilder, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	c := &commonBuilder{
		logger: config.Logger,
	}

	return c, nil
}

func (c *commonBuilder) Build(ctx context.Context) (Common, error) {
	circleBuildNum, err := getEnvVarRequired(EnvVarCircleBuildNum)
	if err != nil {
		return Common{}, microerror.Mask(err)
	}
	circleCI, err := getEnvVarOptional(EnvVarCircleCI)
	if err != nil {
		return Common{}, microerror.Mask(err)
	}
	circleSHA1, err := getEnvVarRequired(EnvVarCircleSHA1)
	if err != nil {
		return Common{}, microerror.Mask(err)
	}
	registryPullSecret, err := getEnvVarRequired(EnvVarRegistryPullSecret)
	if err != nil {
		return Common{}, microerror.Mask(err)
	}
	testDir, err := getEnvVarRequired(EnvVarTestDir)
	if err != nil {
		return Common{}, microerror.Mask(err)
	}

	var cCircleBuildNumber uint
	{
		i, err := strconv.Atoi(circleBuildNum)
		if err != nil {
			return Common{}, microerror.Maskf(executionFailedError, "converting circle build number %#q to int", circleBuildNum)
		}
		if i < 1 {
			return Common{}, microerror.Maskf(executionFailedError, "circle build number must be a positive number but got %d", cCircleBuildNumber)
		}

		cCircleBuildNumber = uint(i)
	}

	var cCircleCI bool
	{
		cCircleCI = circleCI == "true"
	}

	common := Common{
		CircleBuildNumber:  cCircleBuildNumber,
		CircleCI:           cCircleCI,
		CircleSHA:          circleSHA1,
		RegistryPullSecret: registryPullSecret,
		TestDir:            testDir,
	}

	return common, nil
}

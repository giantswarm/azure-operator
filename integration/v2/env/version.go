package env

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	EnvVarGithubBotToken = "GITHUB_BOT_TOKEN"
	EnvVarTestedVersion  = "TESTED_VERSION"
)

type Version struct {
	TestedVersion TestedVersion

	AzureOperator string
}

type versionBuilderConfig struct {
	Logger micrologger.Logger
}

type versionBuilder struct {
	logger micrologger.Logger
}

func newVersionBuilder(config versionBuilderConfig) (*versionBuilder, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	v := &versionBuilder{
		logger: config.Logger,
	}

	return v, nil
}

func (v *versionBuilder) Build(ctx context.Context) (Version, error) {
	githubBotToken, err := getEnvVarRequired(EnvVarGithubBotToken)
	if err != nil {
		return Version{}, microerror.Mask(err)
	}
	testedVersion, err := getEnvVarRequired(EnvVarTestedVersion)
	if err != nil {
		return Version{}, microerror.Mask(err)
	}

	// This builder will require further rethinking when it's moved to
	// e2esetup. Most likely it will expose methods specific for an
	// operator. Nothing to worry now.

	// This code block is going to be replaced with releaseindex package of
	// e2esetup in a follow up PR.
	var vTestedVersion TestedVersion
	{
		switch testedVersion {
		case TestedVersionLast.String(), "wip":
			vTestedVersion = TestedVersionLast
		case TestedVersionPrev.String(), "current":
			vTestedVersion = TestedVersionPrev
		default:
			return Version{}, microerror.Maskf(executionFailedError, "expected tested version to be %#q or %#q but got %#q", TestedVersionLast, TestedVersionPrev, testedVersion)
		}
	}

	// This code block is going to be replaced with releaseindex package of
	// e2esetup in a follow up PR.
	var vAzureOperator string
	{
		params := &framework.VBVParams{
			Component: "azure-operator",
			Provider:  "azure",
			Token:     githubBotToken,
			VType:     testedVersion,
		}
		vAzureOperator, err = framework.GetVersionBundleVersion(params)
		if err != nil {
			panic(err.Error())
		}
		if vAzureOperator == "" {
			if strings.ToLower(testedVersion) == "wip" {
				log.Println("WIP version bundle version not present, exiting.")
				os.Exit(0)
			}
			panic("version bundle version  must not be empty")
		}
	}

	version := Version{
		TestedVersion: vTestedVersion,

		AzureOperator: vAzureOperator,
	}

	return version, nil
}

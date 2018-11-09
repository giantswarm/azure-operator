package env

import (
	"fmt"
	"os"

	"github.com/giantswarm/e2esetup/pkg/releaseindex"
	"github.com/giantswarm/micrologger"
)

const (
	EnvVarGithubBotToken = "GITHUB_BOT_TOKEN"
)

var (
	azureOperatorLatestVersion   string
	azureOperatorPreviousVersion string
)

func init() {
	var err error

	var logger micrologger.Logger
	{
		logger, err = micrologger.New(micrologger.Config{})
		if err != nil {
			panic(fmt.Sprintf("%#v", err))
		}
	}

	githubToken := os.Getenv(EnvVarGithubBotToken)
	if githubToken == "" {
		panic(fmt.Sprintf("env var %q must not be empty", EnvVarGithubBotToken))
	}

	var releaseIndex *releaseindex.ReleaseIndex
	{
		c := releaseindex.Config{
			GithubToken: githubToken,
			Logger:      logger,
		}

		releaseIndex, err = releaseindex.New(c)
		if err != nil {
			panic(fmt.Sprintf("%#v", err))
		}
	}

	releaseIndex.GetVersion()
}

func AzureOperatorLatestVersion() string {
	return azureOperatorLatestVersion
}

func AzureOperatorPreviousVersion() string {
	return azureOperatorPreviousVersion
}

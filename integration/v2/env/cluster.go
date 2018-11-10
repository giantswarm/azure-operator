package env

import (
	"context"
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	EnvVarClusterID     = "E2ESETUP_CLUSTER_ID"
	EnvVarCommonDomain  = "COMMON_DOMAIN"
	EnvVarKeepResources = "KEEP_RESOURCES"
	EnvVarVaultToken    = "VAULT_TOKEN"
)

type Cluster struct {
	BaseDomain    string
	ID            string
	KeepResources bool
	VaultToken    string
}

type clusterBuilderConfig struct {
	Logger micrologger.Logger

	CircleSHA     string
	TestDir       string
	TestedVersion TestedVersion
}

type clusterBuilder struct {
	logger micrologger.Logger

	circleSHA     string
	testDir       string
	testedVersion string
}

func newClusterBuilder(config clusterBuilderConfig) (*clusterBuilder, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.CircleSHA == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.CircleSHA must not be empty", config)
	}
	if config.TestDir == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.TestDir must not be empty", config)
	}

	c := &clusterBuilder{
		logger: config.Logger,

		circleSHA:     config.CircleSHA,
		testDir:       config.TestDir,
		testedVersion: config.TestedVersion,
	}

	return c, nil
}

func (c *clusterBuilder) Build(ctx context.Context) (Cluster, error) {
	clusterID, err := getEnvVarOptional(EnvVarClusterID)
	if err != nil {
		return Cluster{}, microerror.Mask(err)
	}
	commonDomain, err := getEnvVarRequired(EnvVarCommonDomain)
	if err != nil {
		return Cluster{}, microerror.Mask(err)
	}
	keepResources, err := getEnvVarOptional(EnvVarKeepResources)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	vaultToken, err := getEnvVarRequired(EnvVarVaultToken)
	if err != nil {
		return Cluster{}, microerror.Mask(err)
	}

	var cID string
	{
		if clusterID != "" {
			cID = clusterID
		} else {
			parts = append(parts, "ci")
			parts = append(parts, c.testedVersion[0:3])
			parts = append(parts, c.circleSHA()[0:5])
			if c.testDir != "" {
				h := sha1.New()
				h.Write([]byte(c.testDir))
				p := fmt.Sprintf("%x", h.Sum(nil))[0:5]
				parts = append(parts, p)
			}

			cID = strings.Join(parts, "-")
		}
	}

	var cKeepResources bool
	{
		cKeepResources = keepResources == "true"
	}

	cluster := Cluster{
		BaseDomain:    commonDomain,
		ID:            cID,
		KeepResources: cKeepResources,
		VaultToken:    vaultToken,
	}

	return cluster, nil
}

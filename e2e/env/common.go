package env

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/azure-operator/v4/e2e/entityid"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
)

const (
	DefaultClusterIDPrefix    = "ci"
	DefaultRandomizeClusterID = false
	DefaultTestedVersion      = "wip"

	EnvVarCircleSHA               = "CIRCLE_SHA1" // #nosec
	EnvVarClusterIDPrefix         = "CLUSTER_ID_PREFIX"
	EnvVarKeepResources           = "KEEP_RESOURCES"
	EnvVarOperatorHelmTarballPath = "OPERATOR_HELM_TARBALL_PATH"
	EnvVarRandomizeClusterID      = "RANDOMIZE_CLUSTER_ID"
	EnvVarTestedVersion           = "TESTED_VERSION"
	EnvVarTestDir                 = "TEST_DIR"
	EnvVarVersionBundleVersion    = "VERSION_BUNDLE_VERSION"
	EnvVarLogAnalyticsWorkspaceID = "LOG_ANALYTICS_WORKSPACE_ID"
	EnvVarLogAnalyticsSharedKey   = "LOG_ANALYTICS_SHARED_KEY"
)

var (
	circleSHA               string
	clusterIDPrefix         string
	logAnalyticsWorkspaceID string
	logAnalyticsSharedKey   string
	nodepoolID              string
	operatorTarballPath     string
	randomizeClusterID      bool
	testDir                 string
	testedVersion           string
	keepResources           string
	versionBundleVersion    string
)

func init() {
	keepResources = os.Getenv(EnvVarKeepResources)
	operatorTarballPath = os.Getenv(EnvVarOperatorHelmTarballPath)

	randomizeClusterIDString := os.Getenv(EnvVarRandomizeClusterID)
	if randomizeClusterIDString == "" {
		randomizeClusterID = DefaultRandomizeClusterID
		fmt.Printf("No value found in '%s': using default value %t\n", EnvVarRandomizeClusterID, DefaultRandomizeClusterID)
	} else {
		randomizeClusterIDEnvVar, err := strconv.ParseBool(randomizeClusterIDString)
		if err != nil {
			panic(fmt.Sprintf("Error while converting provided env var %s value %q to boolean\n", EnvVarRandomizeClusterID, randomizeClusterIDString))
		}

		randomizeClusterID = randomizeClusterIDEnvVar
	}

	clusterIDPrefix = os.Getenv(EnvVarClusterIDPrefix)
	if clusterIDPrefix == "" {
		// Default cluster ID prefix is always the same for CI
		clusterIDPrefix = DefaultClusterIDPrefix
		fmt.Printf("No value found in '%s': using default value %s\n", EnvVarClusterIDPrefix, DefaultClusterIDPrefix)
	}

	if randomizeClusterID {
		randomPrefixPart := entityid.Generate()
		clusterIDPrefix = fmt.Sprintf("%s-%s", clusterIDPrefix, randomPrefixPart)
	}

	circleSHA = os.Getenv(EnvVarCircleSHA)
	if circleSHA == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarCircleSHA))
	}

	logAnalyticsWorkspaceID = os.Getenv(EnvVarLogAnalyticsWorkspaceID)
	logAnalyticsSharedKey = os.Getenv(EnvVarLogAnalyticsSharedKey)

	testedVersion = os.Getenv(EnvVarTestedVersion)
	if testedVersion == "" {
		testedVersion = DefaultTestedVersion
		fmt.Printf("No value found in '%s': using default value %s\n", EnvVarTestedVersion, DefaultTestedVersion)
	}

	testDir = os.Getenv(EnvVarTestDir)

	clusterID := os.Getenv("CLUSTER_NAME")
	if clusterID == "" {
		err := os.Setenv("CLUSTER_NAME", ClusterID())
		if err != nil {
			panic(err)
		}
	}

	{
		switch testedVersion {
		case "latest", "wip":
			vbs := []versionbundle.Bundle{project.NewVersionBundle()}
			versionBundleVersion = vbs[len(vbs)-1].Version
		case "previous", "current":
			vbs := []versionbundle.Bundle{project.NewVersionBundle()}
			versionBundleVersion = vbs[len(vbs)-2].Version
		}
	}
	err := os.Setenv(EnvVarVersionBundleVersion, VersionBundleVersion())
	if err != nil {
		panic(err)
	}
}

func CircleSHA() string {
	return circleSHA
}

func OperatorHelmTarballPath() string {
	return operatorTarballPath
}

// ClusterID returns a cluster ID unique to a run e2e test. It might
// look like ci-wip-3cc75-5e958.
//
//     ci is a static identifier stating a CI run of the azure-operator.
//     wip is a version reference which can also be cur for the current version.
//     3cc75 is the Git SHA.
//     5e958 is a hash of the e2e test dir, if any.
//
func ClusterID() string {
	var parts []string

	parts = append(parts, clusterIDPrefix)
	parts = append(parts, TestedVersion()[0:3])
	parts = append(parts, CircleSHA()[0:5])
	if TestHash() != "" {
		parts = append(parts, TestHash())
	}

	return strings.Join(parts, "-")
}

func KeepResources() string {
	return keepResources
}

func LogAnalyticsWorkspaceID() string {
	return logAnalyticsWorkspaceID
}

func LogAnalyticsSharedKey() string {
	return logAnalyticsSharedKey
}

func NodePoolID() string {
	if nodepoolID == "" {
		nodepoolID = entityid.Generate()
	}

	return nodepoolID
}

func TestedVersion() string {
	return testedVersion
}

func TestDir() string {
	return testDir
}

func TestHash() string {
	if TestDir() == "" {
		return ""
	}

	h := sha256.New()
	_, err := h.Write([]byte(TestDir()))
	if err != nil {
		panic(fmt.Sprintf("couldn't write hash of test dir '%s'", TestDir()))
	}
	s := fmt.Sprintf("%x", h.Sum(nil))[0:5]

	return s
}

func VersionBundleVersion() string {
	return versionBundleVersion
}

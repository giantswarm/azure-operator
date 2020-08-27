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

	// MaxClusterIDPrefixLength is set to 15 due to limitations for some Azure resource names (like storage account)
	MaxClusterIDPrefixAndTestedVersionLength = 15

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
	clusterID               string
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

	clusterIDPrefix = os.Getenv(EnvVarClusterIDPrefix)
	if clusterIDPrefix == "" {
		// Default cluster ID prefix is always the same for CI
		clusterIDPrefix = DefaultClusterIDPrefix
		fmt.Printf("No value found in '%s': using default value %s\n", EnvVarClusterIDPrefix, DefaultClusterIDPrefix)
	}

	testDir = os.Getenv(EnvVarTestDir)

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

	clusterID = os.Getenv("CLUSTER_NAME")
	if clusterID == "" {
		var parts []string

		// default "ci", can override with CLUSTER_ID_PREFIX env var
		parts = append(parts, clusterIDPrefix)

		// default "false", enable randomization by setting RANDOMIZE_CLUSTER_ID env var to true
		if randomizeClusterID {
			parts = append(parts, entityid.Generate())
		}

		// verify max length of the cluster ID
		if randomizeClusterID && (len(clusterIDPrefix)+len(testedVersion) > MaxClusterIDPrefixAndTestedVersionLength) {
			panic(fmt.Sprintf(
				"Max length for cluster ID prefix (%s) + tested version (%s) version, when combined, "+
					"is %d due to Azure resource name limitations (e.g. storage account name). Try passing a shorter prefix in '%s'.",
				clusterIDPrefix, testedVersion, MaxClusterIDPrefixAndTestedVersionLength, EnvVarClusterIDPrefix))
		}

		// default "wip", can override with TESTED_VERSION env var
		parts = append(parts, testedVersion[0:3])

		// for deterministic cluster ID, append commit hash and test name hash
		if !randomizeClusterID {
			parts = append(parts, circleSHA[0:5])
			if TestHash() != "" {
				parts = append(parts, TestHash())
			}
		}

		clusterID = strings.Join(parts, "-")
		err = os.Setenv("CLUSTER_NAME", clusterID)
		if err != nil {
			panic(err)
		}
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
// You can set CLUSTER_ID_PREFIX environment variable to replace "ci" prefix
// with a custom value, for example jane-wip-3cc75-5e958.
//
// You can also set RANDOMIZE_CLUSTER_ID environment variable to true, and you
// will get a random cluster ID every time, for example jane-d12q5-wip, where
// d12q5 are randomly generated 5 characters.
//
// You can set CLUSTER_NAME environment variable if you want a fully custom
// cluster ID.
//
func ClusterID() string {
	return clusterID
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

func RandomizeClusterID() bool {
	return randomizeClusterID
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

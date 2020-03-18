package env

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/azure-operator/pkg/project"
)

const (
	EnvVarCircleCI             = "CIRCLECI"    // #nosec
	EnvVarCircleSHA            = "CIRCLE_SHA1" // #nosec
	EnvVarKeepResources        = "KEEP_RESOURCES"
	EnvVarRegistryPullSecret   = "REGISTRY_PULL_SECRET" // #nosec
	EnvVarTestedVersion        = "TESTED_VERSION"
	EnvVarTestDir              = "TEST_DIR"
	EnvVarVersionBundleVersion = "VERSION_BUNDLE_VERSION"
)

var (
	circleCI             string
	circleSHA            string
	registryPullSecret   string
	githubToken          string
	testDir              string
	testedVersion        string
	keepResources        string
	versionBundleVersion string
)

func init() {
	circleCI = os.Getenv(EnvVarCircleCI)
	keepResources = os.Getenv(EnvVarKeepResources)

	circleSHA = os.Getenv(EnvVarCircleSHA)
	if circleSHA == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarCircleSHA))
	}

	testedVersion = os.Getenv(EnvVarTestedVersion)
	if testedVersion == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarTestedVersion))
	}

	registryPullSecret = os.Getenv(EnvVarRegistryPullSecret)
	if registryPullSecret == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarRegistryPullSecret))
	}

	testDir = os.Getenv(EnvVarTestDir)

	// TODO(xh3b4sd) this can be changed after the revamp of the e2e templates. I
	// have this on my list.
	clusterID := os.Getenv("CLUSTER_NAME")
	if clusterID == "" {
		os.Setenv("CLUSTER_NAME", ClusterID())
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
	os.Setenv(EnvVarVersionBundleVersion, VersionBundleVersion())
}

func CircleCI() string {
	return circleCI
}

func CircleSHA() string {
	return circleSHA
}

// ClusterID returns a cluster ID unique to a run integration test. It might
// look like ci-wip-3cc75-5e958.
//
//     ci is a static identifier stating a CI run of the azure-operator.
//     wip is a version reference which can also be cur for the current version.
//     3cc75 is the Git SHA.
//     5e958 is a hash of the integration test dir, if any.
//
func ClusterID() string {
	var parts []string

	parts = append(parts, "ci")
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

func RegistryPullSecret() string {
	return registryPullSecret
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

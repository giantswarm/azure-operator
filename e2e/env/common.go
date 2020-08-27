package env

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/azure-operator/v4/pkg/project"
)

const (
	DefaultTestedVersion = "wip"

	EnvVarCircleSHA               = "CIRCLE_SHA1" // #nosec
	EnvVarKeepResources           = "KEEP_RESOURCES"
	EnvVarOperatorHelmTarballPath = "OPERATOR_HELM_TARBALL_PATH"
	EnvVarTestedVersion           = "TESTED_VERSION"
	EnvVarTestDir                 = "TEST_DIR"
	EnvVarVersionBundleVersion    = "VERSION_BUNDLE_VERSION"
	EnvVarLogAnalyticsWorkspaceID = "LOG_ANALYTICS_WORKSPACE_ID"
	EnvVarLogAnalyticsSharedKey   = "LOG_ANALYTICS_SHARED_KEY"

	// idChars represents the character set used to generate node pool IDs.
	// (does not contain 1 and l, to avoid confusion)
	idChars = "023456789abcdefghijkmnopqrstuvwxyz"

	// idLength represents the number of characters used to create a node pool ID.
	idLength = 5
)

var (
	circleSHA               string
	logAnalyticsWorkspaceID string
	logAnalyticsSharedKey   string
	nodepoolID              string
	operatorTarballPath     string
	testDir                 string
	testedVersion           string
	keepResources           string
	versionBundleVersion    string
)

var (
	// Use local instance of RNG. Can be overwritten with fixed seed in tests
	// if needed.
	localRng = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func init() {
	keepResources = os.Getenv(EnvVarKeepResources)
	operatorTarballPath = os.Getenv(EnvVarOperatorHelmTarballPath)

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

func LogAnalyticsWorkspaceID() string {
	return logAnalyticsWorkspaceID
}

func LogAnalyticsSharedKey() string {
	return logAnalyticsSharedKey
}

func NewRandomEntityID() string {
	pattern := regexp.MustCompile("^[a-z]+$")
	for {
		letterRunes := []rune(idChars)
		b := make([]rune, idLength)
		for i := range b {
			b[i] = letterRunes[localRng.Intn(len(letterRunes))]
		}

		id := string(b)

		if _, err := strconv.Atoi(id); err == nil {
			// string is numbers only, which we want to avoid
			continue
		}

		if pattern.MatchString(id) {
			// strings is letters only, which we also avoid
			continue
		}

		return id
	}
}

func NodePoolID() string {
	if nodepoolID == "" {
		nodepoolID = NewRandomEntityID()
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

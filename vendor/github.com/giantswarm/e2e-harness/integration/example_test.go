// +build k8srequired

package integration

import (
	"os"
	"strings"
	"testing"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAggregationClient(t *testing.T) {
	acs, err := getK8sAggregationClient()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	apis, err := acs.ApiregistrationV1beta1().APIServices().List(metav1.ListOptions{})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if len(apis.Items) == 0 {
		t.Errorf("Unexpected empty list of apiservices")
	}
}

func TestZeroInitialPods(t *testing.T) {
	cs, err := getK8sClient()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	pods, err := cs.CoreV1().Pods("default").List(metav1.ListOptions{})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if len(pods.Items) != 0 {
		t.Errorf("Unexpected number of pods, expected 0, got %d", len(pods.Items))
	}
}

func TestEnvVars(t *testing.T) {
	expected := "expected_value"
	actual := os.Getenv("EXPECTED_KEY")

	if expected != actual {
		t.Errorf("unexpected value for EXPECTED_KEY, expected %q, got %q", expected, actual)
	}
}

func TestVersionBundles(t *testing.T) {
	token := os.Getenv("GITHUB_BOT_TOKEN")

	params := &framework.VBVParams{
		VType:     "wip",
		Token:     token,
		Provider:  "aws",
		Component: "aws-operator",
	}

	wipVersion, err := framework.GetVersionBundleVersion(params)
	if err != nil {
		t.Errorf("failed getting index file content for wip: %v", err)
	}
	wipItems := strings.Split(wipVersion, ".")
	if len(wipItems) != 3 {
		t.Errorf("WIP version bundle version doesn't look like semver: %v", wipVersion)
	}

	params.VType = "current"
	currentVersion, err := framework.GetVersionBundleVersion(params)
	if err != nil {
		t.Errorf("failed getting index file content for current: %v", err)
	}
	currentItems := strings.Split(currentVersion, ".")
	if len(currentItems) != 3 {
		t.Errorf("Current version bundle version doesn't look like semver: %v", currentVersion)
	}
}

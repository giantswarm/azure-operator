package loadtest

import (
	"context"
)

const (
	// ApdexPassThreshold is the minimum value allowed for the test to pass.
	// Apdex (Application Performance Index) is an index that summarizes the
	// performance of a system under test. For more information see
	// https://en.wikipedia.org/wiki/Apdex#Apdex_method.
	ApdexPassThreshold      = 0.95
	AppChartName            = "loadtest-app-chart"
	CNRAddress              = "https://quay.io"
	CNROrganization         = "giantswarm"
	ChartChannel            = "stable"
	ChartNamespace          = "e2e-app"
	CustomResourceName      = "kubernetes-nginx-ingress-controller-chart"
	CustomResourceNamespace = "giantswarm"
	JobChartName            = "stormforger-cli-chart"
	TestName                = "aws-operator-e2e"
	UserConfigMapName       = "nginx-ingress-controller-user-values"
)

type Interface interface {
	// Test executes the loadtest test that checks that tenant cluster
	// components behave correctly under load. This primarily involves testing
	// the HPA configuration for Nginx Ingress Controller is correct and
	// interacts correctly with the cluster-autoscaler when it is enabled.
	//
	// The load test is performed by Stormforger. Their testapp is installed as
	// the test workload and a job is created to trigger the loadtest via their
	// CLI.
	//
	// https://github.com/stormforger/cli
	// https://github.com/stormforger/testapp
	//
	//     - Generate loadtest-app endpoint for the tenant cluster.
	//     - Wait for tenant cluster kubernetes API to be up.
	//     - Install loadtest-app chart in the tenant cluster.
	//     - Wait for loadtest-app deployment to be ready.
	//     - Enable HPA for Nginx Ingress Controller.
	//     - Install stormforger-cli chart.
	//     - Wait for stormforger-cli job to be completed.
	//     - Get logs for stormforger-cli pod with the results.
	//     - Parse the results and determine whether the test passed.
	//
	Test(ctx context.Context) error
}

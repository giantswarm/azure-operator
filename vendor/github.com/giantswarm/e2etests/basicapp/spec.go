package basicapp

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/client-go/kubernetes"
)

type Clients interface {
	// K8sClient returns a properly configured control plane client for the
	// Kubernetes API.
	K8sClient() kubernetes.Interface
}

// Chart is the chart to test.
type Chart struct {
	Name            string
	URL             string
	ChartValues     string
	Namespace       string
	RunReleaseTests bool
}

func (cc Chart) Validate() error {
	if cc.URL == "" {
		return microerror.Maskf(invalidConfigError, "%T.URL must not be empty", cc)
	}
	if cc.Name == "" {
		return microerror.Maskf(invalidConfigError, "%T.Name must not be empty", cc)
	}
	if cc.Namespace == "" {
		return microerror.Maskf(invalidConfigError, "%T.Namespace must not be empty", cc)
	}

	return nil
}

// ChartResources are the key resources deployed by the chart.
type ChartResources struct {
	DaemonSets  []DaemonSet
	Deployments []Deployment
	Services    []Service
}

func (cr ChartResources) Validate() error {
	if len(cr.DaemonSets) == 0 && len(cr.Deployments) == 0 && len(cr.Services) == 0 {
		return microerror.Maskf(invalidConfigError, "at least one daemonset, deployment or service must be specified")
	}

	return nil
}

// DaemonSet is a daemonset to be tested.
type DaemonSet struct {
	Name        string
	Namespace   string
	Labels      map[string]string
	MatchLabels map[string]string
}

// Deployment is a deployment to be tested.
type Deployment struct {
	Name             string
	Namespace        string
	DeploymentLabels map[string]string
	MatchLabels      map[string]string
	PodLabels        map[string]string
}

// Service is a service to be tested.
type Service struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

type Interface interface {
	// Test executes the test of a managed services chart with basic
	// functionality that applies to all managed services charts.
	//
	// - Install chart.
	// - Check chart is deployed.
	// - Check key resources are correct.
	// - Run helm release tests if configured.
	//
	Test(ctx context.Context) error
}

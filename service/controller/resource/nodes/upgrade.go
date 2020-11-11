package nodes

import (
	"context"

	releasev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// AnyOutOfDate iterates over all nodes in tenant cluster and finds
// corresponding azure-operator version from node labels. If node doesn't have
// this label or was created with older version than currently reconciling one,
// then this function returns true. Otherwise (including on error) false.
func AnyOutOfDate(ctx context.Context, destinationRelease string, releases []releasev1alpha1.Release) (bool, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		return false, clientNotFoundError
	}

	nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, microerror.Mask(err)
	}

	for _, n := range nodeList.Items {
		rollingRequired, err := nodeNeedsToBeRolled(n, destinationRelease, releases)
		if err != nil {
			return false, microerror.Mask(err)
		}

		if rollingRequired {
			return true, nil
		}
	}

	return false, nil
}

func nodeNeedsToBeRolled(node corev1.Node, destinationRelease string, releases []releasev1alpha1.Release) (bool, error) {
	nodeVer := key.ReleaseVersion(&node)
	if nodeVer == "" {
		return true, nil
	}

	nodeRelease, found := findRelease(nodeVer, releases)

	// Current release doesn't exist anymore. Must be upgraded.
	if !found {
		return true, nil
	}

	destRelease, found := findRelease(destinationRelease, releases)
	if !found {
		return false, microerror.Maskf(executionFailedError, "destination release %q not found", destinationRelease)
	}

	componentsThatRequireRolling := []string{
		"azure-operator",
		"calico",
		"containerlinux",
		"etcd",
		"kubernetes",
	}

	for _, c := range componentsThatRequireRolling {
		nodeComponentVersion, err := key.ComponentVersion(nodeRelease, c)
		if err != nil {
			return false, microerror.Mask(err)
		}

		destComponentVersion, err := key.ComponentVersion(destRelease, c)
		if err != nil {
			return false, microerror.Mask(err)
		}

		if nodeComponentVersion != destComponentVersion {
			return true, nil
		}
	}

	return false, nil
}

func findRelease(v string, releases []releasev1alpha1.Release) (releasev1alpha1.Release, bool) {
	var found bool
	var release releasev1alpha1.Release
	for _, r := range releases {
		if r.Name == key.ReleaseName(v) {
			found = true
			release = r
			break
		}
	}

	return release, found
}

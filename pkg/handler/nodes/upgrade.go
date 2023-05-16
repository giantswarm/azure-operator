package nodes

import (
	"context"

	"github.com/giantswarm/microerror"
	releasev1alpha1 "github.com/giantswarm/release-operator/v4/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

// AnyOutOfDate iterates over all nodes in tenant cluster and finds
// corresponding azure-operator version from node labels. If node doesn't have
// this label or was created with older version than currently reconciling one,
// then this function returns true. Otherwise (including on error) false.
func AnyOutOfDate(ctx context.Context, tenantClusterK8sClient ctrlclient.Client, destinationRelease string, releases []releasev1alpha1.Release, nodeLabels map[string]string) (bool, error) {
	var nodeList *corev1.NodeList
	{
		nodeList = &corev1.NodeList{}
		err := tenantClusterK8sClient.List(ctx, nodeList, ctrlclient.MatchingLabels(nodeLabels))
		if err != nil {
			return false, microerror.Mask(err)
		}
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

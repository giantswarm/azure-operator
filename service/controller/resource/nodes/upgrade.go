package nodes

import (
	"context"

	"github.com/coreos/go-semver/semver"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
)

// AnyOutOfDate iterates over all nodes in tenant cluster and finds
// corresponding azure-operator version from node labels. If node doesn't have
// this label or was created with older version than currently reconciling one,
// then this function returns true. Otherwise (including on error) false.
func AnyOutOfDate(ctx context.Context) (bool, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		return false, clientNotFoundError
	}

	nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return false, microerror.Mask(err)
	}

	myVersion := semver.New(project.Version())
	for _, n := range nodeList.Items {
		v, exists := n.GetLabels()[label.OperatorVersion]
		if !exists {
			return true, nil
		}

		nodeVersion := semver.New(v)

		if nodeVersion.LessThan(*myVersion) {
			return true, nil
		}
	}

	return false, nil
}

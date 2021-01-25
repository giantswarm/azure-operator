package nodes

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetTenantClusterNodes(ctx context.Context, tenantClusterK8sClient client.Client) ([]corev1.Node, error) {
	nodeList := &corev1.NodeList{}
	err := tenantClusterK8sClient.List(ctx, nodeList)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return nodeList.Items, nil
}

package crmapper

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) getWorkers(ctx context.Context, cluster capiv1alpha3.Cluster, azureCluster capzv1alpha3.AzureCluster) ([]providerv1alpha1.AzureConfigSpecAzureNode, error) {
	azureMachineList := &capzv1alpha3.AzureMachineList{}
	{
		err := r.ctrlClient.List(
			ctx,
			azureMachineList,
			client.InNamespace(cluster.Namespace),
			client.MatchingLabels{label.Cluster: key.ClusterID(&cluster)},
		)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var workers []providerv1alpha1.AzureConfigSpecAzureNode
	for _, m := range azureMachineList.Items {
		if !key.IsControlPlaneMachine(&m) {
			n := providerv1alpha1.AzureConfigSpecAzureNode{
				VMSize:              m.Spec.VMSize,
				DockerVolumeSizeGB:  dockerVolumeSizeGB,
				KubeletVolumeSizeGB: kubeletVolumeSizeGB,
			}
			workers = append(workers, n)
		}
	}

	return workers, nil
}

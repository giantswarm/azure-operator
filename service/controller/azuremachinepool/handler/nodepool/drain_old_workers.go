package nodepool

import (
	"context"
	"strings"
	"time"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v6/pkg/drainer"
	"github.com/giantswarm/azure-operator/v6/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v6/service/controller/key"

	"github.com/giantswarm/azure-operator/v6/pkg/handler/nodes/state"
)

func (r *Resource) drainOldWorkerInstances(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		r.Logger.Debugf(ctx, "Cluster is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	if azureMachinePool.Spec.Template.SpotVMOptions != nil {
		r.Logger.Debugf(ctx, "Skipping state %s because node pool is using Spot Instances.", currentState)
		return TerminateOldWorkerInstances, nil
	}

	oldInstances, _, err := r.splitInstancesByUpdatedStatus(ctx, azureMachinePool)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	if len(oldInstances) > 0 {
		r.Logger.Debugf(ctx, "There are still %d workers from the previous release running", len(oldInstances))

		tenantClusterK8sClients, err := r.tenantClientFactory.GetAllClients(ctx, cluster)
		if tenantcluster.IsAPINotAvailableError(err) {
			r.Logger.Debugf(ctx, "tenant API not available yet")
			r.Logger.Debugf(ctx, "canceling resource")

			return currentState, nil
		} else if err != nil {
			return currentState, microerror.Mask(err)
		}

		nodeDrainer, err := drainer.New(drainer.Config{
			Logger:    r.Logger,
			WCClients: tenantClusterK8sClients,
		})
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		completed := true
		for _, instance := range oldInstances {
			nodeName := strings.ToLower(*instance.OsProfile.ComputerName)
			r.Logger.Debugf(ctx, "Draining node %q (instance name %q)", nodeName, *instance.Name)
			err = nodeDrainer.DrainNode(ctx, nodeName, 15*time.Minute)
			if drainer.IsEvictionInProgress(err) {
				// Node still draining.
				r.Logger.Debugf(ctx, "Node %q is still draining: %s", nodeName, err)
				completed = false
			} else if drainer.IsDrainTimeout(err) {
				// Node drain timed out.
				r.Logger.Debugf(ctx, "Timeout while draining node %q", nodeName)
			} else if err != nil {
				r.Logger.Debugf(ctx, "Error draining node %q: %s", nodeName, err)
				return currentState, microerror.Mask(err)
			} else {
				r.Logger.Debugf(ctx, "Node %q drained successfully", nodeName)
			}
		}

		if completed {
			return TerminateOldWorkerInstances, nil
		}

		// Nodes still to be drained.
		return currentState, nil
	}

	// All old nodes are terminated.
	r.Logger.Debugf(ctx, "no old workers were found")

	return TerminateOldWorkerInstances, nil
}

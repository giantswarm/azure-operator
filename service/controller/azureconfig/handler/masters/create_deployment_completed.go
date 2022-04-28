package masters

import (
	"context"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/tenantcluster/v6/pkg/tenantcluster"

	"github.com/giantswarm/microerror"
	releasev1alpha1 "github.com/giantswarm/release-operator/v3/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/checksum"
	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes"
	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v5/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) deploymentCompletedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return Empty, microerror.Mask(err)
	}
	deploymentsClient, err := r.ClientFactory.GetDeploymentsClient(ctx, cr.ObjectMeta)
	if err != nil {
		return Empty, microerror.Mask(err)
	}
	groupsClient, err := r.ClientFactory.GetGroupsClient(ctx, cr.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	d, err := deploymentsClient.Get(ctx, key.ClusterID(&cr), key.MastersVmssDeploymentName)
	if IsDeploymentNotFound(err) {
		r.Logger.Debugf(ctx, "deployment should be completed but is not found")
		r.Logger.Debugf(ctx, "going back to DeploymentUninitialized")
		return Empty, nil
	} else if err != nil {
		return Empty, microerror.Mask(err)
	}

	s := *d.Properties.ProvisioningState
	r.Logger.Debugf(ctx, "deployment is in state '%s'", s)

	group, err := groupsClient.Get(ctx, key.ClusterID(&cr))
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	if key.IsSucceededProvisioningState(s) {
		// Check if any of the nodes is out of date.
		var releases []releasev1alpha1.Release
		{
			var rels releasev1alpha1.ReleaseList
			err := r.ctrlClient.List(ctx, &rels)
			if err != nil {
				return "", microerror.Mask(err)
			}
			releases = rels.Items
		}

		var tenantClusterK8sClient client.Client
		{
			tenantClusterK8sClient, err = r.getTenantClusterClient(ctx, &cr)
			if tenant.IsAPINotAvailable(err) || tenantcluster.IsTimeout(err) {
				// The kubernetes API is not reachable. This usually happens when a new cluster is being created.
				// This makes the whole controller to fail and stops next handlers from being executed even if they are
				// safe to run. We don't want that to happen so we just return and we'll try again during next loop.
				r.Logger.Debugf(ctx, "tenant API not available yet")
				r.Logger.Debugf(ctx, "canceling resource")

				return currentState, nil
			} else if err != nil {
				return "", microerror.Mask(err)
			}
		}

		anyOldNodes, err := nodes.AnyOutOfDate(ctx, tenantClusterK8sClient, key.ReleaseVersion(&cr), releases, map[string]string{"role": "master"})
		if nodes.IsClientNotFound(err) {
			r.Logger.Debugf(ctx, "tenant cluster client not found")
			return currentState, nil
		} else if err != nil {
			return "", microerror.Mask(err)
		}

		if anyOldNodes {
			r.Logger.Debugf(ctx, "tenant cluster has master node[s] out of date")
			return Empty, nil
		}

		computedDeployment, err := r.newDeployment(ctx, cr, nil, *group.Location)
		if blobclient.IsBlobNotFound(err) {
			r.Logger.Debugf(ctx, "ignition blob not found")
			return currentState, nil
		} else if err != nil {
			return "", microerror.Mask(err)
		} else {
			desiredDeploymentTemplateChk, err := checksum.GetDeploymentTemplateChecksum(computedDeployment)
			if err != nil {
				return "", microerror.Mask(err)
			}

			desiredDeploymentParametersChk, err := checksum.GetDeploymentParametersChecksum(computedDeployment)
			if err != nil {
				return "", microerror.Mask(err)
			}

			currentDeploymentTemplateChk, err := r.GetResourceStatus(ctx, cr, DeploymentTemplateChecksum)
			if err != nil {
				return "", microerror.Mask(err)
			}

			currentDeploymentParametersChk, err := r.GetResourceStatus(ctx, cr, DeploymentParametersChecksum)
			if err != nil {
				return "", microerror.Mask(err)
			}

			if currentDeploymentTemplateChk != desiredDeploymentTemplateChk || currentDeploymentParametersChk != desiredDeploymentParametersChk {
				r.Logger.Debugf(ctx, "template or parameters changed")
				// As current and desired state differs, start process from the beginning.
				return Empty, nil
			}

			r.Logger.Debugf(ctx, "template and parameters unchanged")

			return currentState, nil
		}
	} else if key.IsFinalProvisioningState(s) {
		// Deployment has failed. Restart from beginning.
		return Empty, nil
	}

	r.Logger.LogCtx(ctx, "level", "warning", "message", "instances reconciliation process reached unexpected state")

	// Normally the process should never get here. In case this happens, start
	// from the beginning.
	return Empty, nil
}

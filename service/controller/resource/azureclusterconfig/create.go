package azureclusterconfig

import (
	"context"
	"fmt"
	"strings"

	corev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/core/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/reconciliationcanceledcontext"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureCluster, err := key.ToAzureCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding required cluster api types")

	var cluster capiv1alpha3.Cluster
	{
		nsName := types.NamespacedName{
			Name:      key.ClusterName(&azureCluster),
			Namespace: azureCluster.Namespace,
		}
		err = r.ctrlClient.Get(ctx, nsName, &cluster)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("referenced Cluster CR (%q) not found", nsName.String()))
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var masterMachines []capzv1alpha3.AzureMachine
	var workerMachines []capzv1alpha3.AzureMachine
	{
		azureMachineList := &capzv1alpha3.AzureMachineList{}
		{
			err := r.ctrlClient.List(
				ctx,
				azureMachineList,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabels{capiv1alpha3.ClusterLabelName: key.ClusterName(&cluster)},
			)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		for _, m := range azureMachineList.Items {
			if key.IsControlPlaneMachine(&m) {
				masterMachines = append(masterMachines, m)
			} else {
				workerMachines = append(workerMachines, m)
			}
		}

		if len(masterMachines) < 1 {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("no control plane AzureMachines found for cluster %q", key.ClusterName(&cluster)))
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found required cluster api types")
	r.logger.LogCtx(ctx, "level", "debug", "message", "building azureclusterconfig from cluster api crs")

	var mappedAzureClusterConfig corev1alpha1.AzureClusterConfig
	{
		mappedAzureClusterConfig, err = r.buildAzureClusterConfig(ctx, cluster, azureCluster, masterMachines, workerMachines)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "built azureclusterconfig from cluster api crs")
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding existing azureclusterconfig")

	var presentAzureClusterConfig corev1alpha1.AzureClusterConfig
	{
		nsName := types.NamespacedName{
			Name:      clusterConfigName(key.ClusterName(&cluster)),
			Namespace: azureCluster.Namespace,
		}
		err = r.ctrlClient.Get(ctx, nsName, &presentAzureClusterConfig)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not found existing azureclusterconfig")
			r.logger.LogCtx(ctx, "level", "debug", "message", "creating azureclusterconfig")
			err = r.ctrlClient.Create(ctx, &mappedAzureClusterConfig)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "created azureclusterconfig")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding if existing azureclusterconfig needs update")
	{
		// Were there any changes that requires CR update?
		changed := false
		if !azureClusterConfigsEqual(mappedAzureClusterConfig, presentAzureClusterConfig) {
			// Copy Spec section as-is. This should always match desired state.
			presentAzureClusterConfig.Spec = mappedAzureClusterConfig.Spec
			changed = true
		}

		// Copy mapped labels if missing or changed, but don't touch labels
		// that we don't manage.
		for k, v := range mappedAzureClusterConfig.Labels {
			old, exists := presentAzureClusterConfig.Labels[k]
			if old != v || !exists {
				presentAzureClusterConfig.Labels[k] = v
				changed = true
			}
		}

		// Were there any changes that requires CR update?
		if !changed {
			r.logger.LogCtx(ctx, "level", "debug", "message", "no update for existing azureclusterconfig needed")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "existing azureclusterconfig needs update")

		err = r.ctrlClient.Update(ctx, &presentAzureClusterConfig)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "existing azureclusterconfig updated")
	}

	return nil
}

func (r *Resource) buildAzureClusterConfig(ctx context.Context, cluster capiv1alpha3.Cluster, azureCluster capzv1alpha3.AzureCluster, masters, workers []capzv1alpha3.AzureMachine) (corev1alpha1.AzureClusterConfig, error) {
	var err error
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return corev1alpha1.AzureClusterConfig{}, microerror.Mask(err)
	}

	credentialSecret, err := r.getCredentialSecret(ctx, cluster.ObjectMeta)
	if err != nil {
		return corev1alpha1.AzureClusterConfig{}, microerror.Mask(err)
	}

	clusterOperatorVersion, err := key.ComponentVersion(cc.Release.Release, "cluster-operator")
	if err != nil {
		return corev1alpha1.AzureClusterConfig{}, microerror.Mask(err)
	}

	failureDomains, err := getAvailabilityZones(masters)
	if err != nil {
		return corev1alpha1.AzureClusterConfig{}, microerror.Mask(err)
	}

	clusterConfig := corev1alpha1.AzureClusterConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureClusterConfig",
			APIVersion: "core.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterConfigName(key.ClusterName(&cluster)),
			Namespace: azureCluster.Namespace,
			Labels: map[string]string{
				label.ClusterOperatorVersion: clusterOperatorVersion,
			},
		},
		Spec: corev1alpha1.AzureClusterConfigSpec{
			Guest: corev1alpha1.AzureClusterConfigSpecGuest{
				ClusterGuestConfig: corev1alpha1.ClusterGuestConfig{
					AvailabilityZones: len(failureDomains),
					DNSZone:           dnsZoneFromAPIEndpoint(azureCluster.Spec.ControlPlaneEndpoint.Host),
					ID:                key.ClusterName(&cluster),
					Name:              key.ClusterName(&cluster),
					Owner:             key.OrganizationID(&cluster),
					ReleaseVersion:    key.ReleaseVersion(&cluster),
					VersionBundles:    componentsToClusterGuestConfigVersionBundles(cc.Release.Release.Spec.Components),
				},
				CredentialSecret: corev1alpha1.AzureClusterConfigSpecGuestCredentialSecret{
					Name:      credentialSecret.Name,
					Namespace: credentialSecret.Namespace,
				},
				Masters: nil,
				Workers: nil,
			},
			VersionBundle: corev1alpha1.AzureClusterConfigSpecVersionBundle{
				Version: clusterOperatorVersion,
			},
		},
	}

	for _, master := range masters {
		m := corev1alpha1.AzureClusterConfigSpecGuestMaster{
			AzureClusterConfigSpecGuestNode: corev1alpha1.AzureClusterConfigSpecGuestNode{
				ID:     master.Name,
				VMSize: master.Spec.VMSize,
			},
		}

		clusterConfig.Spec.Guest.Masters = append(clusterConfig.Spec.Guest.Masters, m)
	}

	for _, worker := range workers {
		w := corev1alpha1.AzureClusterConfigSpecGuestWorker{
			AzureClusterConfigSpecGuestNode: corev1alpha1.AzureClusterConfigSpecGuestNode{
				ID:     worker.Name,
				VMSize: worker.Spec.VMSize,
			},
			Labels: worker.Labels,
		}

		clusterConfig.Spec.Guest.Workers = append(clusterConfig.Spec.Guest.Workers, w)
	}

	return clusterConfig, nil
}

func clusterConfigName(clusterID string) string {
	return fmt.Sprintf("%s-%s", clusterID, "azure-cluster-config")
}

func dnsZoneFromAPIEndpoint(apiEndpointHost string) string {
	domainLabels := strings.Split(apiEndpointHost, ".")

	switch {
	case len(domainLabels) < 2:
		return domainLabels[0]
	case len(domainLabels[0]) == 0 || len(domainLabels[1]) == 0:
		return apiEndpointHost
	default:
		// Drop first label of domain (e.g. api.foobar.com -> foobar.com).
		return strings.Join(domainLabels[1:], ".")
	}
}

func componentsToClusterGuestConfigVersionBundles(components []releasev1alpha1.ReleaseSpecComponent) []corev1alpha1.ClusterGuestConfigVersionBundle {
	versions := make([]corev1alpha1.ClusterGuestConfigVersionBundle, 0)
	for _, c := range components {
		v := corev1alpha1.ClusterGuestConfigVersionBundle{
			Name:    c.Name,
			Version: c.Version,
		}
		versions = append(versions, v)
	}

	return versions
}

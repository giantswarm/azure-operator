package azureclusterupgrade

import (
	"context"

	"github.com/giantswarm/apiextensions/v6/pkg/annotation"
	"github.com/giantswarm/apiextensions/v6/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/pkg/helpers"
	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "azureclusterupgrade"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

type Resource struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return r, nil
}

// EnsureCreated ensures that all AzureMachines has the same release label as
// reconciled AzureCluster CR.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToAzureCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensuring release labels are set on AzureMachines")

	azureMachineList := capz.AzureMachineList{}
	err = r.ctrlClient.List(ctx, &azureMachineList, client.MatchingLabels{capi.ClusterLabelName: key.ClusterName(&cr)})
	if err != nil {
		return microerror.Mask(err)
	}

	// All found AzureMachines belonging to same cluster are processed at once.
	// This will change when HA masters story is implemented.
	for i := range azureMachineList.Items {
		m := azureMachineList.Items[i]
		changed := false
		if m.Labels == nil {
			m.Labels = make(map[string]string)
		}

		// First initialize release.giantswarm.io/last-deployed-version annotation
		// and set it to the current release version.
		// We need this for existing clusters, and we do it both in Cluster
		// controller (clusterupgrade) and in AzureMachine controller, because
		// we do not know which controller/handler will be executed first, and
		// we need annotation set correctly in both places.
		_, annotationWasSet := m.Annotations[annotation.LastDeployedReleaseVersion]
		err = helpers.InitAzureMachineAnnotations(ctx, r.ctrlClient, r.logger, &m)
		if err != nil {
			return microerror.Mask(err)
		}
		_, annotationIsSet := m.Annotations[annotation.LastDeployedReleaseVersion]
		changed = !annotationWasSet && annotationIsSet

		if m.Labels[label.AzureOperatorVersion] != cr.Labels[label.AzureOperatorVersion] ||
			m.Labels[label.ReleaseVersion] != cr.Labels[label.ReleaseVersion] {

			m.Labels[label.AzureOperatorVersion] = cr.Labels[label.AzureOperatorVersion]
			m.Labels[label.ReleaseVersion] = cr.Labels[label.ReleaseVersion]
			changed = true
		}

		if changed {
			err = r.ctrlClient.Update(ctx, &m)
			if apierrors.IsConflict(err) {
				r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
				r.logger.Debugf(ctx, "cancelling resource")
				return nil
			} else if err != nil {
				return microerror.Mask(err)
			}
		}
	}
	r.logger.Debugf(ctx, "ensured release labels are set on respective AzureMachines")

	return nil
}

// EnsureDeleted is no-op.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

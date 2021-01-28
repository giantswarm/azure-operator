package azureclusterupgrade

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
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

	azureMachineList := capzv1alpha3.AzureMachineList{}
	err = r.ctrlClient.List(ctx, &azureMachineList, client.MatchingLabels{capiv1alpha3.ClusterLabelName: key.ClusterName(&cr)})
	if err != nil {
		return microerror.Mask(err)
	}

	// All found AzureMachines belonging to same cluster are processed at once.
	// This will change when HA masters story is implemented.
	for _, m := range azureMachineList.Items {
		changed := false
		if m.Labels == nil {
			m.Labels = make(map[string]string)
		}

		// First initialize release.giantswarm.io/last-deployed-version annotation
		// and set it to the current release version.
		if m.Annotations == nil {
			m.Annotations = map[string]string{}
		}

		_, ok := m.Annotations[annotation.LastDeployedReleaseVersion]
		if !ok {
			// Initialize annotation for the CRs that do not have it set. This
			// will ensure that Creating and Upgrading conditions are set
			// properly for the first time.
			m.Annotations[annotation.LastDeployedReleaseVersion] = m.Labels[label.ReleaseVersion]
			changed = true
		}

		if m.Labels[label.AzureOperatorVersion] != cr.Labels[label.AzureOperatorVersion] ||
			m.Labels[label.ReleaseVersion] != cr.Labels[label.ReleaseVersion] {

			m.Labels[label.AzureOperatorVersion] = cr.Labels[label.AzureOperatorVersion]
			m.Labels[label.ReleaseVersion] = cr.Labels[label.ReleaseVersion]
			changed = true
		}

		if changed {
			err = r.ctrlClient.Update(ctx, &m)
			if apierrors.IsConflict(err) {
				r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently", "stack", microerror.JSON(microerror.Mask(err)))
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

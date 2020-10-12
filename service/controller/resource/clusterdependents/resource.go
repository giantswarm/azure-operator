package clusterdependents

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/finalizerskeptcontext"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "clusterdependents"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

// Resource ensures that there are no orphaned dependent AzureCluster or MachinePool CRs.
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

// EnsureCreated is a no-op.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	return nil
}

// EnsureDeleted ensures that all dependent CRs are deleted before finalizer
// from Cluster is removed.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring Cluster CR is deleted")

	deleted, err := r.ensureMachinePoolCRsDeleted(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	if !deleted {
		finalizerskeptcontext.SetKept(ctx)
		return nil
	}

	deleted, err = r.ensureInfrastructureCRDeleted(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	if !deleted {
		finalizerskeptcontext.SetKept(ctx)
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured Cluster CR is deleted")
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) ensureInfrastructureCRDeleted(ctx context.Context, cr capiv1alpha3.Cluster) (bool, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring AzureCluster CR is deleted")
	objKey := client.ObjectKey{
		Namespace: cr.Namespace,
		Name:      cr.Spec.InfrastructureRef.Name,
	}
	azureCluster := new(v1alpha3.AzureCluster)
	err := r.ctrlClient.Get(ctx, objKey, azureCluster)
	if errors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureCluster CR is deleted")
		return true, nil
	} else if err != nil {
		r.logger.LogCtx(ctx, "level", "warning", "message", "an error occurred while ensuring AzureCluster CR is deleted")
		return false, microerror.Mask(err)
	}

	err = r.ctrlClient.Delete(ctx, azureCluster)
	if errors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureCluster CR is deleted")
		return true, nil
	} else if err != nil {
		r.logger.LogCtx(ctx, "level", "warning", "message", "an error occurred while ensuring AzureCluster CR is deleted")
		return false, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "AzureCluster CR is still not deleted")
	return false, nil
}

func (r *Resource) ensureMachinePoolCRsDeleted(ctx context.Context, cr capiv1alpha3.Cluster) (bool, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring MachinePool CRs are deleted")
	o := client.MatchingLabels{
		capiv1alpha3.ClusterLabelName: key.ClusterName(&cr),
	}
	mpList := new(expcapiv1alpha3.MachinePoolList)
	err := r.ctrlClient.List(ctx, mpList, o)
	if err != nil {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("error while getting all MachinePool CRs for cluster %q", cr.Name))
		return false, microerror.Mask(err)
	}

	for _, mp := range mpList.Items {
		if !mp.GetDeletionTimestamp().IsZero() {
			// Don't handle deleted child
			continue
		}

		err = r.ctrlClient.Delete(ctx, &mp)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured MachinePool CR %q is deleted", mp.Name))
			continue
		} else if err != nil {
			r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("error while ensuring MachinePool CR %q is deleted", mp.Name))
			return false, microerror.Mask(err)
		}
	}

	allMachinePoolCRsDeleted := len(mpList.Items) == 0
	if allMachinePoolCRsDeleted {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured MachinePool CRs are deleted")
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "MachinePool CRs are still not deleted")
	}

	return allMachinePoolCRsDeleted, nil
}

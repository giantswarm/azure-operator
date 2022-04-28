package machinepoolownerreference

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "machinepoolownerreference"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
	Scheme     *runtime.Scheme
}

// Resource ensures MachinePool owning AzureMachinePool CRs.
type Resource struct {
	ctrlClient client.Client
	logger     micrologger.Logger
	scheme     *runtime.Scheme
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Scheme == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Scheme must not be empty", config)
	}

	r := &Resource{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
		scheme:     config.Scheme,
	}

	return r, nil
}

// EnsureCreated ensures that OwnerReference is correctly set for
// infrastructure CR.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	machinePool, err := key.ToMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Ensuring %#q label and 'ownerReference' fields on MachinePool '%s/%s' and AzureMachinePool '%s/%s'", capi.ClusterLabelName, machinePool.Namespace, machinePool.Name, machinePool.Namespace, machinePool.Spec.Template.Spec.InfrastructureRef.Name)

	azureMachinePool := capzexp.AzureMachinePool{}
	err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: machinePool.Namespace, Name: machinePool.Spec.Template.Spec.InfrastructureRef.Name}, &azureMachinePool)
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "AzureMachinePool %s/%s was not found for MachinePool %#q, skipping setting owner reference", machinePool.Namespace, machinePool.Spec.Template.Spec.InfrastructureRef.Name, machinePool.Name)
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if !azureMachinePool.GetDeletionTimestamp().IsZero() {
		r.logger.Debugf(ctx, "AzureMachinePool is being deleted, skipping setting owner reference")
		return nil
	}

	if machinePool.Labels == nil {
		machinePool.Labels = make(map[string]string)
	}
	machinePool.Labels[capi.ClusterLabelName] = machinePool.Spec.ClusterName

	if azureMachinePool.Labels == nil {
		azureMachinePool.Labels = make(map[string]string)
	}
	azureMachinePool.Labels[capi.ClusterLabelName] = machinePool.Spec.ClusterName

	cluster, err := capiutil.GetClusterByName(ctx, r.ctrlClient, machinePool.Namespace, machinePool.Spec.ClusterName)
	if err != nil {
		return microerror.Mask(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		r.logger.Debugf(ctx, "Cluster is being deleted, skipping setting owner reference")
		return nil
	}

	// Set Cluster as owner of MachinePool
	machinePool.OwnerReferences = capiutil.EnsureOwnerRef(machinePool.OwnerReferences, metav1.OwnerReference{
		APIVersion: capi.GroupVersion.String(),
		Kind:       "Cluster",
		Name:       cluster.Name,
		UID:        cluster.UID,
	})

	err = r.ctrlClient.Update(ctx, &machinePool)
	if apierrors.IsConflict(err) {
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensured %s label and 'ownerReference' fields on MachinePool '%s/%s'", capi.ClusterLabelName, machinePool.Namespace, machinePool.Name))

	// Set MachinePool as owner of AzureMachinePool
	err = controllerutil.SetControllerReference(&machinePool, &azureMachinePool, r.scheme)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ctrlClient.Update(ctx, &azureMachinePool)
	if apierrors.IsConflict(err) {
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensured %s label and 'ownerReference' fields on AzureMachinePool '%s/%s'", capi.ClusterLabelName, machinePool.Namespace, machinePool.Spec.Template.Spec.InfrastructureRef.Name))

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

package machinepoolownerreference

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
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

	r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensuring %#q label and 'ownerReference' fields on MachinePool '%s/%s' and AzureMachinePool '%s/%s'", capiv1alpha3.ClusterLabelName, machinePool.Namespace, machinePool.Name, machinePool.Namespace, machinePool.Spec.Template.Spec.InfrastructureRef.Name))

	azureMachinePool := expcapzv1alpha3.AzureMachinePool{}
	err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: machinePool.Namespace, Name: machinePool.Spec.Template.Spec.InfrastructureRef.Name}, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	if machinePool.Labels == nil {
		machinePool.Labels = make(map[string]string)
	}
	machinePool.Labels[capiv1alpha3.ClusterLabelName] = machinePool.Spec.ClusterName

	if azureMachinePool.Labels == nil {
		azureMachinePool.Labels = make(map[string]string)
	}
	azureMachinePool.Labels[capiv1alpha3.ClusterLabelName] = machinePool.Spec.ClusterName

	cluster, err := capiutil.GetClusterByName(ctx, r.ctrlClient, machinePool.Namespace, machinePool.Spec.ClusterName)
	if err != nil {
		return microerror.Mask(err)
	}

	// Set Cluster as owner of MachinePool
	machinePool.OwnerReferences = capiutil.EnsureOwnerRef(machinePool.OwnerReferences, metav1.OwnerReference{
		APIVersion: capiv1alpha3.GroupVersion.String(),
		Kind:       "Cluster",
		Name:       cluster.Name,
		UID:        cluster.UID,
	})

	err = r.ctrlClient.Update(ctx, &machinePool)
	if apierrors.IsConflict(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "conflict trying to save object in k8s API concurrently", "stack", microerror.JSON(microerror.Mask(err)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensured %s label and 'ownerReference' fields on MachinePool '%s/%s'", capiv1alpha3.ClusterLabelName, machinePool.Namespace, machinePool.Name))

	// Set MachinePool as owner of AzureMachinePool
	err = controllerutil.SetControllerReference(&machinePool, &azureMachinePool, r.scheme)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ctrlClient.Update(ctx, &azureMachinePool)
	if apierrors.IsConflict(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "conflict trying to save object in k8s API concurrently", "stack", microerror.JSON(microerror.Mask(err)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensured %s label and 'ownerReference' fields on AzureMachinePool '%s/%s'", capiv1alpha3.ClusterLabelName, machinePool.Namespace, machinePool.Spec.Template.Spec.InfrastructureRef.Name))

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

package clusterownerreference

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "clusterownerreference"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
	Scheme     *runtime.Scheme
}

// Resource ensures Cluster owns AzureCluster and MachinePool.
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
	cluster, err := key.ToCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureAzureCluster(ctx, cluster)
	if IsCRBeingDeletedError(err) {
		r.logger.Debugf(ctx, "AzureCluster is being deleted, skipping setting owner references")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureMachinePools(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureControlPlane(ctx, &cluster)
	if err != nil {
		return microerror.Mask(err)
	}

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

func (r *Resource) ensureAzureCluster(ctx context.Context, cluster capi.Cluster) error {
	var err error
	r.logger.Debugf(ctx, "Ensuring %s label and 'ownerReference' fields on AzureCluster '%s/%s'", capi.ClusterLabelName, cluster.Namespace, cluster.Spec.InfrastructureRef.Name)

	azureCluster := capz.AzureCluster{}
	err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cluster.Namespace, Name: cluster.Spec.InfrastructureRef.Name}, &azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	if !azureCluster.GetDeletionTimestamp().IsZero() {
		return microerror.Mask(crBeingDeletedError)
	}

	if azureCluster.Labels == nil {
		azureCluster.Labels = make(map[string]string)
	}
	azureCluster.Labels[capi.ClusterLabelName] = cluster.Name

	// Set Cluster as owner of AzureCluster
	err = controllerutil.SetControllerReference(&cluster, &azureCluster, r.scheme)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ctrlClient.Update(ctx, &azureCluster)
	if apierrors.IsConflict(err) {
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Ensured %s label and 'ownerReference' fields on AzureCluster '%s/%s'", capi.ClusterLabelName, cluster.Namespace, cluster.Spec.InfrastructureRef.Name)

	if cluster.Spec.InfrastructureRef != nil && cluster.Spec.InfrastructureRef.APIVersion != "" {
		// Correct Cluster.Spec.InfrastructureRef with APIVersion is already set
		return nil
	}

	var infrastructureCRRef *corev1.ObjectReference
	{
		infrastructureCRRef, err = reference.GetReference(r.scheme, &azureCluster)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	cluster.Spec.InfrastructureRef = infrastructureCRRef

	err = r.ctrlClient.Update(ctx, &cluster)
	if apierrors.IsConflict(err) {
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Ensured 'Spec.InfrastructureRef' fields on Cluster '%s/%s'", cluster.Namespace, cluster.Name)
	return nil
}

func (r *Resource) ensureMachinePools(ctx context.Context, cluster capi.Cluster) error {
	var err error

	o := client.MatchingLabels{
		capi.ClusterLabelName: key.ClusterName(&cluster),
	}
	mpList := new(capiexp.MachinePoolList)
	err = r.ctrlClient.List(ctx, mpList, o)
	if err != nil {
		return microerror.Mask(err)
	}

	for i := range mpList.Items {
		machinePool := mpList.Items[i]

		if !machinePool.GetDeletionTimestamp().IsZero() {
			r.logger.Debugf(ctx, "MachinePool %#q is being deleted, skipping setting owner references", machinePool.Name)
			continue
		}

		r.logger.Debugf(ctx, "Ensuring %s label and 'ownerReference' fields on MachinePool '%s/%s'", capi.ClusterLabelName, machinePool.Namespace, machinePool.Name)

		if machinePool.Labels == nil {
			machinePool.Labels = make(map[string]string)
		}
		machinePool.Labels[capi.ClusterLabelName] = cluster.Name

		// Set Cluster as owner of MachinePool
		err = controllerutil.SetControllerReference(&cluster, &machinePool, r.scheme)
		if err != nil {
			return microerror.Mask(err)
		}

		err = r.ctrlClient.Update(ctx, &machinePool)
		if apierrors.IsConflict(err) {
			r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
			r.logger.Debugf(ctx, "cancelling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "Ensured %s label and 'ownerReference' fields on MachinePool '%s/%s'", capi.ClusterLabelName, machinePool.Namespace, machinePool.Name)
	}

	return nil
}

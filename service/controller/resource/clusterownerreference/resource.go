package clusterownerreference

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
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

// Resource manages Azure resource groups.
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
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureMachinePools(ctx, cluster)
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

func (r *Resource) ensureAzureCluster(ctx context.Context, cluster capiv1alpha3.Cluster) error {
	var err error
	r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensuring %s label and 'ownerReference' fields on AzureCluster '%s/%s'", capiv1alpha3.ClusterLabelName, cluster.Namespace, cluster.Spec.InfrastructureRef.Name))

	azureCluster := v1alpha3.AzureCluster{}
	err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cluster.Namespace, Name: cluster.Spec.InfrastructureRef.Name}, &azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	if azureCluster.Labels == nil {
		azureCluster.Labels = make(map[string]string)
	}
	azureCluster.Labels[capiv1alpha3.ClusterLabelName] = cluster.Name

	// Set Cluster as owner of AzureCluster
	err = controllerutil.SetControllerReference(&cluster, &azureCluster, r.scheme)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ctrlClient.Update(ctx, &azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensured %s label and 'ownerReference' fields on AzureCluster '%s/%s'", capiv1alpha3.ClusterLabelName, cluster.Namespace, cluster.Spec.InfrastructureRef.Name))
	return nil
}

func (r *Resource) ensureMachinePools(ctx context.Context, cluster capiv1alpha3.Cluster) error {
	var err error

	o := client.MatchingLabels{
		capiv1alpha3.ClusterLabelName: key.ClusterID(&cluster),
	}
	mpList := new(expcapiv1alpha3.MachinePoolList)
	err = r.ctrlClient.List(ctx, mpList, o)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, machinePool := range mpList.Items {
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensuring %s label and 'ownerReference' fields on MachinePool '%s/%s'", capiv1alpha3.ClusterLabelName, machinePool.Namespace, machinePool.Name))

		if machinePool.Labels == nil {
			machinePool.Labels = make(map[string]string)
		}
		machinePool.Labels[capiv1alpha3.ClusterLabelName] = cluster.Name

		// Set Cluster as owner of MachinePool
		err = controllerutil.SetControllerReference(&cluster, &machinePool, r.scheme)
		if err != nil {
			return microerror.Mask(err)
		}

		err = r.ctrlClient.Update(ctx, &machinePool)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "message", fmt.Sprintf("Ensured %s label and 'ownerReference' fields on MachinePool '%s/%s'", capiv1alpha3.ClusterLabelName, machinePool.Namespace, machinePool.Name))
	}

	return nil
}

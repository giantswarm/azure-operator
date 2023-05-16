package cloudconfigblob

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/client"
)

const (
	// Name is the identifier of the resource.
	Name = "cloudconfigblob"
)

type Config struct {
	ClientFactory client.OrganizationFactory
	CtrlClient    ctrlclient.Client
	Logger        micrologger.Logger
}

// Resource manages the blob saved in Azure Storage Account that contains the cloudconfig files to bootstrap our virtual machines.
type Resource struct {
	clientFactory client.OrganizationFactory
	ctrlClient    ctrlclient.Client
	logger        micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		clientFactory: config.ClientFactory,
		ctrlClient:    config.CtrlClient,
		logger:        config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

// getMachinePoolByName finds and return a MachinePool object using the specified params.
func (r *Resource) getMachinePoolByName(ctx context.Context, namespace, name string) (*capiexp.MachinePool, error) {
	machinePool := &capiexp.MachinePool{}
	objectKey := ctrlclient.ObjectKey{Name: name, Namespace: namespace}
	if err := r.ctrlClient.Get(ctx, objectKey, machinePool); err != nil {
		return nil, err
	}

	r.logger = r.logger.With("machinePool", machinePool.Name)

	return machinePool, nil
}

// getOwnerMachinePool returns the MachinePool object owning the current resource.
func (r *Resource) getOwnerMachinePool(ctx context.Context, obj metav1.ObjectMeta) (*capiexp.MachinePool, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind == "MachinePool" && ref.APIVersion == capiexp.GroupVersion.String() {
			return r.getMachinePoolByName(ctx, obj.Namespace, ref.Name)
		}
	}

	return nil, nil
}

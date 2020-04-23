package masters

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

// This transition function aims at detecting if the master VMSS needs to be migrated from CoreOS to flatcar.
func (r *Resource) checkFlatcarMigrationNeededTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	legacyExists, err := r.vmssExistsAndHasActiveInstance(ctx, key.MasterVMSSName(cr))
	if err != nil {
		return Empty, microerror.Mask(err)
	}

}

func (r *Resource) vmssExistsAndHasActiveInstance(ctx context.Context, vmssName string) (bool, error) {
	// TODO Check if the legacy master VMSS exists and has an active instance.
	return false, nil
}

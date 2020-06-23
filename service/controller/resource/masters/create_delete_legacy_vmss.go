package masters

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) deleteLegacyVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	// Delete the scaleset
	err = r.deleteScaleSet(ctx, cr, key.ResourceGroupName(cr), key.LegacyMasterVMSSName(cr))
	if IsScaleSetNotFound(err) {
		// Scale set not found, all good.
		return DeploymentCompleted, nil
	} else if err != nil {
		return Empty, microerror.Mask(err)
	}

	return UnblockAPICalls, nil
}

func (r *Resource) deleteScaleSet(ctx context.Context, customObject providerv1alpha1.AzureConfig, resourceGroup string, vmssName string) error {
	c, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(key.CredentialNamespace(customObject), key.CredentialName(customObject))
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.Delete(ctx, resourceGroup, vmssName)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

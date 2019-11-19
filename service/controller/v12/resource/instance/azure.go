package instance

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v12/key"
	"github.com/giantswarm/microerror"
)

func (r *Resource) reimageInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error)) error {
	if instance == nil {
		return nil
	}

	instanceName, err := instanceNameFunc(customObject, *instance.InstanceID)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be reimaged", instanceName))

	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	g := key.ResourceGroupName(customObject)
	s := deploymentNameFunc(customObject)
	ids := &compute.VirtualMachineScaleSetReimageParameters{
		InstanceIds: to.StringSlicePtr([]string{
			*instance.InstanceID,
		}),
	}
	res, err := c.Reimage(ctx, g, s, ids)
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = c.ReimageResponder(res.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be reimaged", instanceName))

	return nil
}

func (r *Resource) updateInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error)) error {
	if instance == nil {
		return nil
	}

	instanceName, err := instanceNameFunc(customObject, *instance.InstanceID)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be updated", instanceName))

	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	g := key.ResourceGroupName(customObject)
	s := deploymentNameFunc(customObject)
	ids := compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
		InstanceIds: to.StringSlicePtr([]string{
			*instance.InstanceID,
		}),
	}
	res, err := c.UpdateInstances(ctx, g, s, ids)
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = c.UpdateInstancesResponder(res.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be updated", instanceName))

	return nil
}

func (r *Resource) allInstances(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string) ([]compute.VirtualMachineScaleSetVM, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("looking for the scale set '%s'", deploymentNameFunc(customObject)))

	c, err := r.getVMsClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	g := key.ResourceGroupName(customObject)
	s := deploymentNameFunc(customObject)
	result, err := c.List(ctx, g, s, "", "", "")
	if IsScaleSetNotFound(err) {
		return nil, microerror.Mask(scaleSetNotFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	var instances []compute.VirtualMachineScaleSetVM

	for result.NotDone() {
		instances = append(instances, result.Values()...)

		err := result.Next()
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found the scale set '%s'", deploymentNameFunc(customObject)))

	return instances, nil
}

// firstInstanceInProgress returns the first instance in the list not having a
// final state. In case all instances are in a final state
// firstInstanceInProgress returns nil.
func firstInstanceInProgress(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM) *compute.VirtualMachineScaleSetVM {
	for _, v := range list {
		if v.ProvisioningState == nil || key.IsFinalProvisioningState(*v.ProvisioningState) {
			continue
		}

		return &v
	}

	return nil
}

// firstInstanceToReimage returns the first instance to be reimaged. The
// decision of reimaging an instance is done by comparing the desired version
// bundle version of the custom object and the current version bundle version of
// the instance's tags applied. In case all instances are reimaged
// firstInstanceToReimage return nil.
func firstInstanceToReimage(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error), versionValue map[string]string) (*compute.VirtualMachineScaleSetVM, error) {
	if versionValue == nil {
		return nil, microerror.Mask(versionBlobEmptyError)
	}

	for _, v := range list {
		desiredVersion := key.VersionBundleVersion(customObject)
		instanceName, err := instanceNameFunc(customObject, *v.InstanceID)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		instanceVersion, ok := versionValue[instanceName]
		if !ok {
			continue
		}
		if desiredVersion == instanceVersion {
			continue
		}

		return &v, nil
	}

	return nil, nil
}

// firstInstanceToUpdate return the first instance to be updated. The decision
// of updating an instance is done by checking if the latest scale set model is
// applied. In case all instances are updated firstInstanceToUpdate return nil.
func firstInstanceToUpdate(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM) *compute.VirtualMachineScaleSetVM {
	for _, v := range list {
		if *v.LatestModelApplied {
			continue
		}

		return &v
	}

	return nil
}

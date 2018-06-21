package endpoints

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

const (
	masterEndpointsName = "master"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Deleting the K8s namespace will take care of cleaning the endpoints.
	if key.IsDeleted(customObject) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "redirecting responsibility of deletion of endpoints to namespace termination")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

		return nil, nil
	}

	// We are only able to compute the desired state of the endpoints in case the
	// network interfaces are available. On cluster creation the resource
	// implementations are non blocking and every resource implementation has to
	// ensure its own requirements are met. So in case the network interfaces for
	// the virtual machines are not yet present, we cancel the resource and try
	// again on the next resync period.
	{
		interfacesClient, err := r.getInterfacesClient()
		if err != nil {
			return nil, microerror.Mask(err)
		}

		g := key.ClusterID(customObject)
		s := key.MasterVMSSName(customObject)
		_, err = interfacesClient.ListVirtualMachineScaleSetNetworkInterfaces(ctx, g, s)
		if IsNetworkInterfacesNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the network interfaces in the Azure API")
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

			return nil, nil
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Lookup the current state of the endpoints.
	var endpoints *corev1.Endpoints
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the master endpoints in the Kubernetes API")

		n := key.ClusterNamespace(customObject)
		manifest, err := r.k8sClient.CoreV1().Endpoints(n).Get(masterEndpointsName, apismetav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the master endpoints in the Kubernetes API")

			return nil, nil
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "found the master endpoints in the Kubernetes API")
			endpoints = manifest
		}
	}

	return endpoints, nil
}

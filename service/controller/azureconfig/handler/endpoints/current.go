package endpoints

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

const (
	masterEndpointsName = "master"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Deleting the K8s namespace will take care of cleaning the endpoints.
	if key.IsDeleted(&cr) {
		r.logger.Debugf(ctx, "redirecting deletion to namespace termination")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.Debugf(ctx, "canceling resource")

		return nil, nil
	}

	// We are only able to compute the desired state of the endpoints in case the
	// network interfaces are available. On cluster creation the resource
	// implementations are non blocking and every resource implementation has to
	// ensure its own requirements are met. So in case the network interfaces for
	// the virtual machines are not yet present, we cancel the resource and try
	// again on the next resync period.
	{
		interfacesClient, err := r.getInterfacesClient(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		g := key.ClusterID(&cr)
		s := key.MasterVMSSName(cr)
		_, err = interfacesClient.ListVirtualMachineScaleSetNetworkInterfaces(ctx, g, s)
		if IsNetworkInterfacesNotFound(err) {
			r.logger.Debugf(ctx, "did not find the network interfaces in the Azure API")
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.Debugf(ctx, "canceling resource")

			return nil, nil
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// Lookup the current state of the endpoints.
	var endpoints *corev1.Endpoints
	{
		r.logger.Debugf(ctx, "looking for the master endpoints in the Kubernetes API")

		n := key.ClusterNamespace(cr)
		manifest, err := r.k8sClient.CoreV1().Endpoints(n).Get(ctx, masterEndpointsName, apismetav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "did not find the master endpoints in the Kubernetes API")

			return nil, nil
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			r.logger.Debugf(ctx, "found the master endpoints in the Kubernetes API")
			endpoints = manifest
		}
	}

	return endpoints, nil
}

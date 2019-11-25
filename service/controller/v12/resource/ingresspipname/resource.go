// Package ingresspipname implements resource to manage migration of public IP
// resource name for public ingress loadbalancer. Originally the public IP was
// called dummy-pip, but when refactoring infrastructure so that k8s
// ingress-controller has annotation with this PIP name, it was also renamed
// more appropriately and in order to manage this with older clusters, the name
// must be kept along with migrations.
package ingresspipname

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/azure-operator/service/controller/v12/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v12/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
)

const (
	Name = "ingresspipnamev12"
)

type Config struct {
	G8sClient versioned.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	g8sClient versioned.Interface
	logger    micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	newResource := &Resource{
		g8sClient: config.G8sClient,
		logger:    config.Logger,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "finding if ingress PIP name must be set to status")

		if key.IngressPIPName(customObject) != "" {
			r.logger.LogCtx(ctx, "level", "debug", "message", "found out that ingress PIP name is already set to status")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

			return nil
		}
	}

	var ingressPIPName string
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "finding if ingress PIP has been already created")

		groupName := key.ClusterID(customObject)
		pipClient, err := r.getPIPClient(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		// First try to look for legacy PIP name (dummmy-pip).
		_, err = pipClient.Get(ctx, groupName, key.LegacyIngressLBPIPName, "")
		if IsPIPNotFound(err) {
			// This is ok. Maybe PIP is created with currently desired name?

			_, err = pipClient.Get(ctx, groupName, key.DefaultIngressPIPName(customObject), "")
			if IsPIPNotFound(err) {
				// This is ok as well. The given Public IP is not created yet.
			} else if err != nil {
				return microerror.Mask(err)
			}

			ingressPIPName = key.DefaultIngressPIPName(customObject)

		} else if err != nil {
			return microerror.Mask(err)
		} else {
			ingressPIPName = key.LegacyIngressLBPIPName
		}

		if ingressPIPName == "" {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find already created ingress PIP")
			ingressPIPName = key.DefaultIngressPIPName(customObject)
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("defaulting to %q", ingressPIPName))
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found ingress PIP with name %q", ingressPIPName))
		}
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "updating CR status with ingress PIP name")

		customObject.Status.Provider.Ingress.LoadBalancer.PublicIPName = ingressPIPName

		_, err := r.g8sClient.ProviderV1alpha1().AzureConfigs(customObject.Namespace).UpdateStatus(&customObject)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "updated CR status with ingress PIP name")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
	}

	return nil
}

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

func (r *Resource) getPIPClient(ctx context.Context) (*network.PublicIPAddressesClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.PublicIPAddressesClient, nil
}

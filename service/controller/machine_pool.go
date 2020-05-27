package controller

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

type MachinePoolConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type MachinePool struct {
	*controller.Controller
}

func NewMachinePool(config MachinePoolConfig) (*MachinePool, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}

	var err error

	var resourceSet *controller.ResourceSet
	{
		c := MachinePoolResourceSetConfig{
			Logger: config.Logger,
		}

		resourceSet, err = NewMachinePoolResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1alpha3.AzureMachinePool)
			},
			ResourceSets: []*controller.ResourceSet{
				resourceSet,
			},
			// Name is used to compute finalizer names. This results in something
			// like operatorkit.giantswarm.io/azure-operator-machine-pool-controller.
			Name: project.Name() + "-machine-pool-controller",
		}

		operatorkitController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return &MachinePool{Controller: operatorkitController}, nil
}

type MachinePoolResourceSetConfig struct {
	Logger micrologger.Logger
}

func NewMachinePoolResourceSet(config MachinePoolResourceSetConfig) (*controller.ResourceSet, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var err error

	handlesFunc := func(obj interface{}) bool {
		cr, err := key.ToAzureMachinePool(obj)
		if err != nil {
			config.Logger.Log("level", "warning", "message", fmt.Sprintf("invalid object: %s", err), "stack", fmt.Sprintf("%v", err)) // nolint: errcheck
			return false
		}

		if key.MachinePoolOperatorVersion(cr) == project.Version() {
			return true
		}

		return false
	}

	initCtxFunc := func(ctx context.Context, obj interface{}) (context.Context, error) {
		return ctx, nil
	}

	var resources []resource.Interface
	{
		c := retryresource.WrapConfig{
			Logger: config.Logger,
		}

		resources, err = retryresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		c := metricsresource.WrapConfig{}
		resources, err = metricsresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var resourceSet *controller.ResourceSet
	{
		c := controller.ResourceSetConfig{
			Handles:   handlesFunc,
			InitCtx:   initCtxFunc,
			Logger:    config.Logger,
			Resources: resources,
		}

		resourceSet, err = controller.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resourceSet, nil
}

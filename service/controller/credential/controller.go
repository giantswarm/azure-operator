package credential

import (
	"context"

	"github.com/giantswarm/certs/v3/pkg/certs"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/controller"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/retryresource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/service/controller/credential/handler/azureclusteridentity"

	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
)

type ControllerConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
	SentryDSN string
}

func NewController(config ControllerConfig) (*controller.Controller, error) {
	var err error

	var certsSearcher *certs.Searcher
	{
		c := certs.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		certsSearcher, err = certs.NewSearcher(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var resources []resource.Interface
	{
		resources, err = newCredentialResources(config, certsSearcher)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			InitCtx: func(ctx context.Context, obj interface{}) (context.Context, error) {
				return controllercontext.NewContext(ctx, controllercontext.Context{}), nil
			},
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			// Name is used to compute finalizer names. This results in something
			// like operatorkit.giantswarm.io/azure-operator-credential-controller.
			Name: project.Name() + "-credential-controller",
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1.Secret)
			},
			Resources: resources,
			Selector: labels.SelectorFromSet(map[string]string{
				label.App: "credentiald",
			}),
			SentryDSN: config.SentryDSN,
		}

		operatorkitController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return operatorkitController, nil
}

func newCredentialResources(config ControllerConfig, certsSearcher certs.Interface) ([]resource.Interface, error) {
	var err error

	var azureclusteridentityResource resource.Interface
	{
		c := azureclusteridentity.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		azureclusteridentityResource, err = azureclusteridentity.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		azureclusteridentityResource,
	}

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

	return resources, nil
}

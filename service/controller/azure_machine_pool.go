package controller

import (
	"context"
	"net"
	"time"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/pkg/credential"
	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/pkg/locker"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/cloudconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/ipam"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodepool"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodes"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

type AzureMachinePoolConfig struct {
	Azure                     setting.Azure
	CredentialProvider        credential.Provider
	GSClientCredentialsConfig auth.ClientCredentialsConfig
	GuestSubnetMaskBits       int
	InstallationName          string
	IPAMNetworkRange          net.IPNet
	K8sClient                 k8sclient.Interface
	Locker                    locker.Interface
	Logger                    micrologger.Logger
	RegistryDomain            string
	SentryDSN                 string
	VMSSCheckWorkers          int
	VMSSMSIEnabled            bool
}

type AzureMachinePool struct {
	*controller.Controller
}

func NewAzureMachinePool(config AzureMachinePoolConfig) (*AzureMachinePool, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}

	var err error

	var resources []resource.Interface
	{
		resources, err = NewAzureMachinePoolResourceSet(config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			InitCtx: func(ctx context.Context, obj interface{}) (context.Context, error) {
				return ctx, nil
			},
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			// Name is used to compute finalizer names. This results in something
			// like operatorkit.giantswarm.io/azure-operator-machine-pool-controller.
			Name: project.Name() + "-machine-pool-controller",
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1alpha3.AzureMachinePool)
			},
			Resources: resources,
			Selector: labels.SelectorFromSet(map[string]string{
				label.OperatorVersion: project.Version(),
			}),
			SentryDSN: config.SentryDSN,
		}

		operatorkitController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return &AzureMachinePool{Controller: operatorkitController}, nil
}

func NewAzureMachinePoolResourceSet(config AzureMachinePoolConfig) ([]resource.Interface, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var err error

	var clientFactory *client.Factory
	{
		c := client.FactoryConfig{
			CacheDuration:      30 * time.Minute,
			CredentialProvider: config.CredentialProvider,
			Logger:             config.Logger,
		}

		clientFactory, err = client.NewFactory(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var newDebugger *debugger.Debugger
	{
		c := debugger.Config{
			Logger: config.Logger,
		}

		newDebugger, err = debugger.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var iwd vmsscheck.InstanceWatchdog
	{
		c := vmsscheck.Config{
			Logger:     config.Logger,
			NumWorkers: config.VMSSCheckWorkers,
		}

		var err error
		iwd, err = vmsscheck.NewInstanceWatchdog(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	nodesConfig := nodes.Config{
		Debugger:  newDebugger,
		G8sClient: config.K8sClient.G8sClient(),
		K8sClient: config.K8sClient.K8sClient(),
		Logger:    config.Logger,

		Azure:            config.Azure,
		ClientFactory:    clientFactory,
		InstanceWatchdog: iwd,
	}

	var instanceResource resource.Interface
	{
		c := nodepool.Config{
			Config:                    nodesConfig,
			CredentialProvider:        config.CredentialProvider,
			CtrlClient:                config.K8sClient.CtrlClient(),
			GSClientCredentialsConfig: config.GSClientCredentialsConfig,
		}

		instanceResource, err = nodepool.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clusterChecker *ipam.AzureMachinePoolChecker
	{
		c := ipam.AzureMachinePoolCheckerConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		clusterChecker, err = ipam.NewAzureMachinePoolChecker(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var azureMachinePoolPersister *ipam.AzureMachinePoolPersister
	{
		c := ipam.AzureMachinePoolPersisterConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		azureMachinePoolPersister, err = ipam.NewAzureMachinePoolPersister(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var subnetCollector *ipam.SubnetCollector
	{
		c := ipam.SubnetCollectorConfig{
			CredentialProvider: config.CredentialProvider,
			K8sClient:          config.K8sClient,
			InstallationName:   config.InstallationName,
			Logger:             config.Logger,

			NetworkRange: config.IPAMNetworkRange,
		}

		subnetCollector, err = ipam.NewSubnetCollector(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var ipamResource resource.Interface
	{
		c := ipam.Config{
			Checker:   clusterChecker,
			Collector: subnetCollector,
			Locker:    config.Locker,
			Logger:    config.Logger,
			Persister: azureMachinePoolPersister,

			AllocatedSubnetMaskBits: config.GuestSubnetMaskBits,
			NetworkRange:            config.IPAMNetworkRange,
		}

		ipamResource, err = ipam.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var certsSearcher *certs.Searcher
	{
		c := certs.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			WatchTimeout: 5 * time.Second,
		}

		certsSearcher, err = certs.NewSearcher(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var cloudConfigResource resource.Interface
	{
		c := cloudconfig.Config{
			AzureClientsFactory: clientFactory,
			CertsSearcher:       certsSearcher,
			CtrlClient:          config.K8sClient.CtrlClient(),
			G8sClient:           config.K8sClient.G8sClient(),
			K8sClient:           config.K8sClient.K8sClient(),
			Logger:              config.Logger,
			RegistryDomain:      config.RegistryDomain,
		}

		cloudconfigObject, err := cloudconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		cloudConfigResource, err = toCRUDResource(config.Logger, cloudconfigObject)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		cloudConfigResource,
		ipamResource,
		instanceResource,
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

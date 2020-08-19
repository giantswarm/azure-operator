package controller

import (
	"context"
	"net"
	"time"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/certs/v2/pkg/certs"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"
	"github.com/giantswarm/randomkeys"
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
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/azureconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/cloudconfigblob"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/ipam"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodepool"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodes"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/spark"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

type AzureMachinePoolConfig struct {
	APIServerSecurePort       int
	Azure                     setting.Azure
	Calico                    azureconfig.CalicoConfig
	ClusterIPRange            string
	CredentialProvider        credential.Provider
	EtcdPrefix                string
	GSClientCredentialsConfig auth.ClientCredentialsConfig
	GuestSubnetMaskBits       int
	Ignition                  setting.Ignition
	InstallationName          string
	IPAMNetworkRange          net.IPNet
	K8sClient                 k8sclient.Interface
	Locker                    locker.Interface
	Logger                    micrologger.Logger
	OIDC                      setting.OIDC
	RegistryDomain            string
	SentryDSN                 string
	SSHUserList               string
	SSOPublicKey              string
	VMSSCheckWorkers          int
	VMSSMSIEnabled            bool
}

func NewAzureMachinePool(config AzureMachinePoolConfig) (*controller.Controller, error) {
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
			Name:      project.Name() + "-azure-machine-pool-controller",
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

	return operatorkitController, nil
}

func NewAzureMachinePoolResourceSet(config AzureMachinePoolConfig) ([]resource.Interface, error) {
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

	var randomkeysSearcher *randomkeys.Searcher
	{
		c := randomkeys.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		randomkeysSearcher, err = randomkeys.NewSearcher(c)
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

	var nodepoolResource resource.Interface
	{
		c := nodepool.Config{
			Config:                    nodesConfig,
			CredentialProvider:        config.CredentialProvider,
			CtrlClient:                config.K8sClient.CtrlClient(),
			GSClientCredentialsConfig: config.GSClientCredentialsConfig,
		}

		nodepoolResource, err = nodepool.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var subnetChecker *ipam.AzureMachinePoolSubnetChecker
	{
		c := ipam.AzureMachinePoolSubnetCheckerConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		subnetChecker, err = ipam.NewAzureMachinePoolSubnetChecker(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var subnetPersister *ipam.AzureMachinePoolSubnetPersister
	{
		c := ipam.AzureMachinePoolSubnetPersisterConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		subnetPersister, err = ipam.NewAzureMachinePoolSubnetPersister(c)
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
			Checker:   subnetChecker,
			Collector: subnetCollector,
			Locker:    config.Locker,
			Logger:    config.Logger,
			Persister: subnetPersister,

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

	var sparkResource resource.Interface
	{
		c := spark.Config{
			APIServerSecurePort: config.APIServerSecurePort,
			Azure:               config.Azure,
			Calico:              config.Calico,
			CertsSearcher:       certsSearcher,
			ClusterIPRange:      config.ClusterIPRange,
			EtcdPrefix:          config.EtcdPrefix,
			CredentialProvider:  config.CredentialProvider,
			CtrlClient:          config.K8sClient.CtrlClient(),
			Ignition:            config.Ignition,
			Logger:              config.Logger,
			OIDC:                config.OIDC,
			RandomKeysSearcher:  randomkeysSearcher,
			RegistryDomain:      config.RegistryDomain,
			SSHUserList:         config.SSHUserList,
			SSOPublicKey:        config.SSOPublicKey,
		}

		sparkResource, err = spark.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var cloudconfigblobResource resource.Interface
	{
		c := cloudconfigblob.Config{
			ClientFactory: clientFactory,
			CtrlClient:    config.K8sClient.CtrlClient(),
			Logger:        config.Logger,
		}

		cloudconfigblobResource, err = cloudconfigblob.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		sparkResource,
		cloudconfigblobResource,
		ipamResource,
		nodepoolResource,
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

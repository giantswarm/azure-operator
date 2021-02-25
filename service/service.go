package service

import (
	"context"
	"fmt"
	"net"
	"sync"

	corev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	exporterkitcollector "github.com/giantswarm/exporterkit/collector"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v5/pkg/k8srestconfig"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	operatorkitcontroller "github.com/giantswarm/operatorkit/v4/pkg/controller"
	"github.com/giantswarm/versionbundle"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v5/azureclient/basicfactory"
	"github.com/giantswarm/azure-operator/v5/azureclient/credentialprovider"
	"github.com/giantswarm/azure-operator/v5/azureclient/credentialsawarefactory"
	"github.com/giantswarm/azure-operator/v5/flag"
	"github.com/giantswarm/azure-operator/v5/pkg/employees"
	"github.com/giantswarm/azure-operator/v5/pkg/locker"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/collector"
	"github.com/giantswarm/azure-operator/v5/service/controller/azurecluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig"
	"github.com/giantswarm/azure-operator/v5/service/controller/azuremachine"
	"github.com/giantswarm/azure-operator/v5/service/controller/azuremachinepool"
	"github.com/giantswarm/azure-operator/v5/service/controller/cluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/machinepool"
	"github.com/giantswarm/azure-operator/v5/service/controller/setting"
	"github.com/giantswarm/azure-operator/v5/service/controller/unhealthynode"
)

// Config represents the configuration used to create a new service.
type Config struct {
	Logger micrologger.Logger

	Flag  *flag.Flag
	Viper *viper.Viper

	Description    string
	GitCommit      string
	ProjectName    string
	RegistryDomain string
	Source         string
	Version        string
}

type Service struct {
	Version *version.Service

	bootOnce          sync.Once
	operatorCollector *exporterkitcollector.Set
	controllers       []*operatorkitcontroller.Controller
}

// New creates a new configured service object.
func New(config Config) (*Service, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Flag must not be empty", config)
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Viper must not be empty", config)
	}
	if config.Description == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Description must not be empty", config)
	}
	if config.GitCommit == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.GitCommit must not be empty", config)
	}
	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ProjectName must not be empty", config)
	}
	if config.Source == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Source must not be empty", config)
	}

	var err error

	resourceGroup := config.Viper.GetString(config.Flag.Service.Azure.HostCluster.ResourceGroup)
	if resourceGroup == "" {
		resourceGroup = config.Viper.GetString(config.Flag.Service.Installation.Name)
	}

	virtualNetwork := config.Viper.GetString(config.Flag.Service.Azure.HostCluster.VirtualNetwork)
	if virtualNetwork == "" {
		virtualNetwork = resourceGroup
	}

	virtualNetworkGateway := config.Viper.GetString(config.Flag.Service.Azure.HostCluster.VirtualNetworkGateway)
	if virtualNetworkGateway == "" {
		virtualNetworkGateway = fmt.Sprintf("%s-%s", resourceGroup, "vpn-gateway")
	}

	azure := setting.Azure{
		EnvironmentName: config.Viper.GetString(config.Flag.Service.Azure.EnvironmentName),
		HostCluster: setting.AzureHostCluster{
			CIDR:                  config.Viper.GetString(config.Flag.Service.Azure.HostCluster.CIDR),
			ResourceGroup:         resourceGroup,
			VirtualNetwork:        virtualNetwork,
			VirtualNetworkGateway: virtualNetworkGateway,
		},
		MSI: setting.AzureMSI{
			Enabled: config.Viper.GetBool(config.Flag.Service.Azure.MSI.Enabled),
		},
		Location: config.Viper.GetString(config.Flag.Service.Azure.Location),
	}

	Ignition := setting.Ignition{
		Path:       config.Viper.GetString(config.Flag.Service.Tenant.Ignition.Path),
		Debug:      config.Viper.GetBool(config.Flag.Service.Tenant.Ignition.Debug.Enabled),
		LogsPrefix: config.Viper.GetString(config.Flag.Service.Tenant.Ignition.Debug.LogsPrefix),
		LogsToken:  config.Viper.GetString(config.Flag.Service.Tenant.Ignition.Debug.LogsToken),
	}

	OIDC := setting.OIDC{
		ClientID:      config.Viper.GetString(config.Flag.Service.Installation.Tenant.Kubernetes.API.Auth.Provider.OIDC.ClientID),
		IssuerURL:     config.Viper.GetString(config.Flag.Service.Installation.Tenant.Kubernetes.API.Auth.Provider.OIDC.IssuerURL),
		UsernameClaim: config.Viper.GetString(config.Flag.Service.Installation.Tenant.Kubernetes.API.Auth.Provider.OIDC.UsernameClaim),
		GroupsClaim:   config.Viper.GetString(config.Flag.Service.Installation.Tenant.Kubernetes.API.Auth.Provider.OIDC.GroupsClaim),
	}

	sentryDSN := config.Viper.GetString(config.Flag.Service.Sentry.DSN)

	var restConfig *rest.Config
	{
		c := k8srestconfig.Config{
			Logger: config.Logger,

			Address:    config.Viper.GetString(config.Flag.Service.Kubernetes.Address),
			InCluster:  config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster),
			KubeConfig: config.Viper.GetString(config.Flag.Service.Kubernetes.KubeConfig),
			TLS: k8srestconfig.ConfigTLS{
				CAFile:  config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile),
				CrtFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile),
				KeyFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile),
			},
		}

		restConfig, err = k8srestconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		restConfig.UserAgent = fmt.Sprintf("%s/%s", project.Name(), project.Version())
	}

	var k8sClient *k8sclient.Clients
	{
		address := config.Viper.GetString(config.Flag.Service.Kubernetes.Address)
		inCluster := config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster)
		kubeConfigPath := config.Viper.GetString(config.Flag.Service.Kubernetes.KubeConfigPath)

		defined := 0
		if address != "" {
			defined++
		}
		if inCluster {
			defined++
		}
		if kubeConfigPath != "" {
			defined++
		}

		if defined == 0 {
			return nil, microerror.Maskf(invalidConfigError, "address or inCluster or kubeConfigPath must be defined")
		}
		if defined > 1 {
			return nil, microerror.Maskf(invalidConfigError, "address and inCluster and kubeConfigPath must not be defined at the same time")
		}

		var restConfig *rest.Config
		if kubeConfigPath == "" {
			restConfig, err = buildK8sRestConfig(config)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		}

		c := k8sclient.ClientsConfig{
			Logger: config.Logger,
			SchemeBuilder: k8sclient.SchemeBuilder{
				corev1alpha1.AddToScheme,
				providerv1alpha1.AddToScheme,
				releasev1alpha1.AddToScheme,
				capiv1alpha3.AddToScheme,
				capzv1alpha3.AddToScheme,
				expcapiv1alpha3.AddToScheme,
				expcapzv1alpha3.AddToScheme,
			},

			KubeConfigPath: kubeConfigPath,
			RestConfig:     restConfig,
		}

		k8sClient, err = k8sclient.NewClients(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var debugSettings setting.Debug
	{
		debugSettings = setting.Debug{
			InsecureStorageAccount: config.Viper.GetBool(config.Flag.Service.Debug.InsecureStorageAccount),
		}
	}

	var kubeLockLocker locker.Interface
	{
		c := locker.KubeLockLockerConfig{
			Logger:     config.Logger,
			RestConfig: restConfig,
		}

		kubeLockLocker, err = locker.NewKubeLockLocker(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var ipamNetworkRange net.IPNet
	{
		_, ipnet, err := net.ParseCIDR(config.Viper.GetString(config.Flag.Service.Installation.Guest.IPAM.Network.CIDR))
		if err != nil {
			return nil, microerror.Mask(err)
		}
		ipamNetworkRange = *ipnet
	}

	var reservedCIDRs []net.IPNet
	{
		_, ipnet, err := net.ParseCIDR(config.Viper.GetString(config.Flag.Service.Azure.HostCluster.CIDR))
		if err != nil {
			return nil, microerror.Mask(err)
		}
		reservedCIDRs = append(reservedCIDRs, *ipnet)
	}

	var azureClientFactory *basicfactory.BasicFactory
	{
		azureClientFactoryConfig := basicfactory.Config{
			Logger:           config.Logger,
			MetricsCollector: nil,
			PartnerID:        config.Viper.GetString(config.Flag.Service.Azure.PartnerID),
		}

		azureClientFactory, err = basicfactory.NewAzureClientFactory(azureClientFactoryConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// This factory will be used when creating AzureClients for Management Clusters.
	var mcAzureClientFactory credentialsawarefactory.Interface
	{
		mcCredentialsProvider, err := credentialprovider.NewCLIFlagsCredentialProvider(
			credentialprovider.CLIFlagsCredentialProviderConfig{
				CtrlClient:                      k8sClient.CtrlClient(),
				Logger:                          config.Logger,
				ManagementClusterClientID:       config.Viper.GetString(config.Flag.Service.Azure.ClientID),
				ManagementClusterClientSecret:   config.Viper.GetString(config.Flag.Service.Azure.ClientSecret),
				ManagementClusterSubscriptionID: config.Viper.GetString(config.Flag.Service.Azure.SubscriptionID),
				ManagementClusterTenantID:       config.Viper.GetString(config.Flag.Service.Azure.TenantID),
			},
		)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		mcAzureClientFactory, err = credentialsawarefactory.NewCredentialsAwareClientFactory(mcCredentialsProvider, *azureClientFactory)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var wcAzureClientFactory credentialsawarefactory.Interface
	{
		wcCredentialProvider, err := credentialprovider.NewK8sSecretCredentialProvider(credentialprovider.K8sSecretCredentialProviderConfig{
			CtrlClient: k8sClient.CtrlClient(),
			Logger:     config.Logger,
			MCTenantID: config.Viper.GetString(config.Flag.Service.Azure.TenantID),
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}

		wcAzureClientFactory, err = credentialsawarefactory.NewCredentialsAwareClientFactory(wcCredentialProvider, *azureClientFactory)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var azureCollector collector.AzureAPIMetrics
	var collectorSet *exporterkitcollector.Set
	{
		azureAPIMetricsCollector, err := collector.NewAzureAPIMetricsCollector(collector.Config{Logger: config.Logger})
		if err != nil {
			return nil, microerror.Mask(err)
		}

		azureCollector = azureAPIMetricsCollector

		c := exporterkitcollector.SetConfig{
			Collectors: []exporterkitcollector.Interface{
				azureAPIMetricsCollector,
			},
			Logger: config.Logger,
		}

		collectorSet, err = exporterkitcollector.NewSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var controllers []*operatorkitcontroller.Controller

	var azureClusterController *operatorkitcontroller.Controller
	{
		c := azurecluster.ControllerConfig{
			WCAzureClientFactory: wcAzureClientFactory,
			K8sClient:            k8sClient,
			Logger:               config.Logger,

			Flag:  config.Flag,
			Viper: config.Viper,

			Azure:                 azure,
			AzureMetricsCollector: azureCollector,
			Ignition:              Ignition,
			OIDC:                  OIDC,
			InstallationName:      config.Viper.GetString(config.Flag.Service.Installation.Name),
			ProjectName:           config.ProjectName,
			RegistryDomain:        config.Viper.GetString(config.Flag.Service.Registry.Domain),
			SSOPublicKey:          config.Viper.GetString(config.Flag.Service.Tenant.SSH.SSOPublicKey),

			SentryDSN: sentryDSN,
		}

		azureClusterController, err = azurecluster.NewController(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		controllers = append(controllers, azureClusterController)
	}

	var sshUserList employees.SSHUserList
	{
		str := config.Viper.GetString(config.Flag.Service.Cluster.Kubernetes.SSH.UserList)

		sshUserList, err = employees.FromDraughtsmanString(str)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var azureConfigController *operatorkitcontroller.Controller
	{
		c := azureconfig.ControllerConfig{
			Azure:                 azure,
			AzureMetricsCollector: azureCollector,
			MCAzureClientFactory:  mcAzureClientFactory,
			WCAzureClientFactory:  wcAzureClientFactory,
			ClusterVNetMaskBits:   config.Viper.GetInt(config.Flag.Service.Installation.Guest.IPAM.Network.SubnetMaskBits),
			DockerhubToken:        config.Viper.GetString(config.Flag.Service.Registry.DockerhubToken),
			Ignition:              Ignition,
			InstallationName:      config.Viper.GetString(config.Flag.Service.Installation.Name),
			IPAMNetworkRange:      ipamNetworkRange,
			IPAMReservedCIDRs:     reservedCIDRs,
			K8sClient:             k8sClient,
			Locker:                kubeLockLocker,
			Logger:                config.Logger,
			OIDC:                  OIDC,
			ProjectName:           config.ProjectName,
			RegistryDomain:        config.Viper.GetString(config.Flag.Service.Registry.Domain),
			RegistryMirrors:       config.Viper.GetStringSlice(config.Flag.Service.Registry.Mirrors),
			SentryDSN:             sentryDSN,
			SSHUserList:           sshUserList,
			SSOPublicKey:          config.Viper.GetString(config.Flag.Service.Tenant.SSH.SSOPublicKey),
			Debug:                 debugSettings,
		}

		azureConfigController, err = azureconfig.NewController(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		controllers = append(controllers, azureConfigController)
	}

	var azureMachinePoolController *operatorkitcontroller.Controller
	{
		c := azuremachinepool.ControllerConfig{
			APIServerSecurePort:   config.Viper.GetInt(config.Flag.Service.Cluster.Kubernetes.API.SecurePort),
			Azure:                 azure,
			AzureMetricsCollector: azureCollector,
			MCAzureClientFactory:  mcAzureClientFactory,
			WCAzureClientFactory:  wcAzureClientFactory,
			CalicoCIDRSize:        config.Viper.GetInt(config.Flag.Service.Cluster.Calico.CIDR),
			CalicoMTU:             config.Viper.GetInt(config.Flag.Service.Cluster.Calico.MTU),
			CalicoSubnet:          config.Viper.GetString(config.Flag.Service.Cluster.Calico.Subnet),
			ClusterIPRange:        config.Viper.GetString(config.Flag.Service.Cluster.Kubernetes.API.ClusterIPRange),
			DockerhubToken:        config.Viper.GetString(config.Flag.Service.Registry.DockerhubToken),
			EtcdPrefix:            config.Viper.GetString(config.Flag.Service.Cluster.Etcd.Prefix),
			Ignition:              Ignition,
			InstallationName:      config.Viper.GetString(config.Flag.Service.Installation.Name),
			K8sClient:             k8sClient,
			Locker:                kubeLockLocker,
			Logger:                config.Logger,
			OIDC:                  OIDC,
			RegistryDomain:        config.Viper.GetString(config.Flag.Service.Registry.Domain),
			SentryDSN:             sentryDSN,
			SSHUserList:           sshUserList,
			SSOPublicKey:          config.Viper.GetString(config.Flag.Service.Tenant.SSH.SSOPublicKey),
			VMSSMSIEnabled:        config.Viper.GetBool(config.Flag.Service.Azure.MSI.Enabled),
		}

		azureMachinePoolController, err = azuremachinepool.NewController(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		controllers = append(controllers, azureMachinePoolController)
	}

	var azureMachineController *operatorkitcontroller.Controller
	{
		c := azuremachine.ControllerConfig{
			AzureMetricsCollector: azureCollector,
			MCAzureClientFactory:  mcAzureClientFactory,
			WCAzureClientFactory:  wcAzureClientFactory,
			K8sClient:             k8sClient,
			Logger:                config.Logger,
			SentryDSN:             sentryDSN,
		}

		azureMachineController, err = azuremachine.NewController(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		controllers = append(controllers, azureMachineController)
	}

	var clusterController *operatorkitcontroller.Controller
	{
		c := cluster.ControllerConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,
			SentryDSN: sentryDSN,
		}

		clusterController, err = cluster.NewController(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		controllers = append(controllers, clusterController)
	}

	var machinePoolController *operatorkitcontroller.Controller
	{
		c := machinepool.ControllerConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,
			SentryDSN: sentryDSN,
		}

		machinePoolController, err = machinepool.NewController(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		controllers = append(controllers, machinePoolController)
	}

	var terminateUnhealthyNodeController *operatorkitcontroller.Controller
	{
		c := unhealthynode.ControllerConfig{
			AzureMetricsCollector: azureCollector,
			MCAzureClientFactory:  mcAzureClientFactory,
			WCAzureClientFactory:  wcAzureClientFactory,
			K8sClient:             k8sClient,
			Logger:                config.Logger,
			SentryDSN:             sentryDSN,
		}

		terminateUnhealthyNodeController, err = unhealthynode.NewController(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		controllers = append(controllers, terminateUnhealthyNodeController)
	}

	var versionService *version.Service
	{
		c := version.Config{
			Description:    config.Description,
			GitCommit:      config.GitCommit,
			Name:           config.ProjectName,
			Source:         config.Source,
			Version:        config.Version,
			VersionBundles: []versionbundle.Bundle{project.NewVersionBundle()},
		}

		versionService, err = version.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	s := &Service{
		bootOnce:          sync.Once{},
		controllers:       controllers,
		operatorCollector: collectorSet,
		Version:           versionService,
	}

	return s, nil
}

// nolint: errcheck
func (s *Service) Boot(ctx context.Context) {
	s.bootOnce.Do(func() {
		for _, ctrl := range s.controllers {
			go ctrl.Boot(ctx)
		}

		go s.operatorCollector.Boot(context.Background())
	})
}

func buildK8sRestConfig(config Config) (*rest.Config, error) {
	c := k8srestconfig.Config{
		Logger: config.Logger,

		Address:    config.Viper.GetString(config.Flag.Service.Kubernetes.Address),
		InCluster:  config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster),
		KubeConfig: config.Viper.GetString(config.Flag.Service.Kubernetes.KubeConfig),
		TLS: k8srestconfig.ConfigTLS{
			CAFile:  config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile),
			CrtFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile),
			KeyFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile),
		},
	}

	restConfig, err := k8srestconfig.New(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return restConfig, nil
}

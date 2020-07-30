package service

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v3/pkg/k8srestconfig"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/statusresource"
	"github.com/giantswarm/versionbundle"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/flag"
	"github.com/giantswarm/azure-operator/v4/pkg/credential"
	"github.com/giantswarm/azure-operator/v4/pkg/locker"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
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

	azureClusterController  *controller.AzureCluster
	bootOnce                sync.Once
	clusterController       *controller.Cluster
	machinePoolController   *controller.MachinePool
	statusResourceCollector *statusresource.CollectorSet
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

	var azureClusterController *controller.AzureCluster
	{
		c := controller.AzureClusterConfig{
			K8sClient: k8sClient,
			Logger:    config.Logger,

			Flag:  config.Flag,
			Viper: config.Viper,

			Azure:            azure,
			Ignition:         Ignition,
			OIDC:             OIDC,
			InstallationName: config.Viper.GetString(config.Flag.Service.Installation.Name),
			ProjectName:      config.ProjectName,
			RegistryDomain:   config.Viper.GetString(config.Flag.Service.Registry.Domain),
			SSOPublicKey:     config.Viper.GetString(config.Flag.Service.Tenant.SSH.SSOPublicKey),
			VMSSCheckWorkers: config.Viper.GetInt(config.Flag.Service.Azure.VMSSCheckWorkers),

			SentryDSN: sentryDSN,
		}

		azureClusterController, err = controller.NewAzureCluster(c)
		if err != nil {
			return nil, microerror.Mask(err)
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

	// These credentials will be used when creating AzureClients for Control Plane clusters.
	gsClientCredentialsConfig, err := credential.NewAzureCredentials(
		config.Viper.GetString(config.Flag.Service.Azure.ClientID),
		config.Viper.GetString(config.Flag.Service.Azure.ClientSecret),
		config.Viper.GetString(config.Flag.Service.Azure.TenantID),
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	credentialProvider := credential.NewK8SCredentialProvider(k8sClient, config.Viper.GetString(config.Flag.Service.Azure.TenantID))

	var clusterController *controller.Cluster
	{
		cpAzureClientSet, err := NewCPAzureClientSet(config, gsClientCredentialsConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		c := controller.ClusterConfig{
			Azure:                     azure,
			CredentialProvider:        credentialProvider,
			CPAzureClientSet:          cpAzureClientSet,
			GSClientCredentialsConfig: gsClientCredentialsConfig,
			GuestSubnetMaskBits:       config.Viper.GetInt(config.Flag.Service.Installation.Guest.IPAM.Network.SubnetMaskBits),
			Ignition:                  Ignition,
			InstallationName:          config.Viper.GetString(config.Flag.Service.Installation.Name),
			IPAMNetworkRange:          ipamNetworkRange,
			K8sClient:                 k8sClient,
			Locker:                    kubeLockLocker,
			Logger:                    config.Logger,
			OIDC:                      OIDC,
			ProjectName:               config.ProjectName,
			RegistryDomain:            config.Viper.GetString(config.Flag.Service.Registry.Domain),
			RegistryMirrors:           config.Viper.GetStringSlice(config.Flag.Service.Registry.Mirrors),
			SentryDSN:                 sentryDSN,
			SSOPublicKey:              config.Viper.GetString(config.Flag.Service.Tenant.SSH.SSOPublicKey),
			VMSSCheckWorkers:          config.Viper.GetInt(config.Flag.Service.Azure.VMSSCheckWorkers),
			Debug:                     debugSettings,
		}

		clusterController, err = controller.NewCluster(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var machinePoolController *controller.MachinePool
	{
		c := controller.MachinePoolConfig{
			CredentialProvider:        credentialProvider,
			GSClientCredentialsConfig: gsClientCredentialsConfig,
			GuestSubnetMaskBits:       config.Viper.GetInt(config.Flag.Service.Installation.Guest.IPAM.Network.SubnetMaskBits),
			InstallationName:          config.Viper.GetString(config.Flag.Service.Installation.Name),
			IPAMNetworkRange:          ipamNetworkRange,
			K8sClient:                 k8sClient,
			Locker:                    kubeLockLocker,
			Logger:                    config.Logger,
			RegistryDomain:            config.Viper.GetString(config.Flag.Service.Registry.Domain),
			SentryDSN:                 sentryDSN,
		}

		machinePoolController, err = controller.NewMachinePool(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var statusResourceCollector *statusresource.CollectorSet
	{
		c := statusresource.CollectorSetConfig{
			Logger:  config.Logger,
			Watcher: k8sClient.G8sClient().ProviderV1alpha1().AzureConfigs("").Watch,
		}

		statusResourceCollector, err = statusresource.NewCollectorSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
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
		Version: versionService,

		azureClusterController:  azureClusterController,
		bootOnce:                sync.Once{},
		clusterController:       clusterController,
		machinePoolController:   machinePoolController,
		statusResourceCollector: statusResourceCollector,
	}

	return s, nil
}

func (s *Service) Boot(ctx context.Context) {
	s.bootOnce.Do(func() {
		go s.statusResourceCollector.Boot(ctx) // nolint: errcheck

		go s.azureClusterController.Boot(ctx)
		go s.clusterController.Boot(ctx)
		go s.machinePoolController.Boot(ctx)
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

// NewCPAzureClientSet return an Azure client set configured for the Control Plane cluster.
func NewCPAzureClientSet(config Config, gsClientCredentialsConfig auth.ClientCredentialsConfig) (*client.AzureClientSet, error) {
	cpTenantID := config.Viper.GetString(config.Flag.Service.Azure.HostCluster.Tenant.TenantID)
	if cpTenantID != "" {
		// We want the code to work both when using Single Tenant Service Principal and Multi Tenant Service Principal.
		// We only add the CP Tenant ID as auxiliary id if an explicit CP Tenant ID has been passed.
		gsClientCredentialsConfig.AuxTenants = append(gsClientCredentialsConfig.AuxTenants, cpTenantID)
	}

	cpSubscriptionID := config.Viper.GetString(config.Flag.Service.Azure.HostCluster.Tenant.SubscriptionID)
	if cpSubscriptionID == "" {
		cpSubscriptionID = config.Viper.GetString(config.Flag.Service.Azure.SubscriptionID)
	}

	cpPartnerID := config.Viper.GetString(config.Flag.Service.Azure.HostCluster.Tenant.PartnerID)
	if cpPartnerID == "" {
		cpPartnerID = config.Viper.GetString(config.Flag.Service.Azure.PartnerID)
	}

	return client.NewAzureClientSet(gsClientCredentialsConfig, cpSubscriptionID, cpPartnerID)
}

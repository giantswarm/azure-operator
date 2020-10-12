package setup

import (
	corev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/apprclient/v2"
	"github.com/giantswarm/e2e-harness/v2/pkg/framework"
	"github.com/giantswarm/e2e-harness/v2/pkg/release"
	e2eclientsazure "github.com/giantswarm/e2eclients/azure"
	e2esetupenv "github.com/giantswarm/e2esetup/v2/chart/env"
	"github.com/giantswarm/helmclient/v2/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v5/e2e/env"
)

const (
	namespace    = "giantswarm"
	organization = "giantswarm"
	quayAddress  = "https://quay.io"
)

type LogAnalyticsConfig struct {
	WorkspaceID string
	SharedKey   string
}

type Config struct {
	AzureClient        *e2eclientsazure.Client
	Guest              *framework.Guest
	HelmClient         helmclient.Interface
	Host               *framework.Host
	K8s                *k8sclient.Setup
	K8sClients         k8sclient.Interface
	Logger             micrologger.Logger
	Release            *release.Release
	LogAnalyticsConfig LogAnalyticsConfig
}

func NewConfig() (Config, error) {
	var err error

	var azureClient *e2eclientsazure.Client
	{
		azureClient, err = e2eclientsazure.NewClient()
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var logger micrologger.Logger
	{
		c := micrologger.Config{}

		logger, err = micrologger.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var apprClient *apprclient.Client
	{
		c := apprclient.Config{
			Logger: logger,

			Address:      quayAddress,
			Organization: organization,
		}

		apprClient, err = apprclient.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var cpK8sClients *k8sclient.Clients
	{
		c := k8sclient.ClientsConfig{
			Logger: logger,
			SchemeBuilder: k8sclient.SchemeBuilder{
				corev1alpha1.AddToScheme,
				providerv1alpha1.AddToScheme,
				releasev1alpha1.AddToScheme,
				capiv1alpha3.AddToScheme,
				capzv1alpha3.AddToScheme,
				expcapiv1alpha3.AddToScheme,
				expcapzv1alpha3.AddToScheme,
			},

			KubeConfigPath: e2esetupenv.KubeConfigPath(),
		}

		cpK8sClients, err = k8sclient.NewClients(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var guest *framework.Guest
	{
		c := framework.GuestConfig{
			Logger:        logger,
			HostK8sClient: cpK8sClients.K8sClient(),

			ClusterID:    env.ClusterID(),
			CommonDomain: env.CommonDomain(),
		}

		guest, err = framework.NewGuest(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var host *framework.Host
	{
		c := framework.HostConfig{
			Logger: logger,

			ClusterID: env.ClusterID(),
		}

		host, err = framework.NewHost(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var k8sSetup *k8sclient.Setup
	{
		c := k8sclient.SetupConfig{
			Clients: cpK8sClients,
			Logger:  logger,
		}

		k8sSetup, err = k8sclient.NewSetup(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var helmClient *helmclient.Client
	{
		c := helmclient.Config{
			Logger:    logger,
			K8sClient: cpK8sClients,
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var newRelease *release.Release
	{
		c := release.Config{
			ApprClient: apprClient,
			ExtClient:  host.ExtClient(),
			G8sClient:  host.G8sClient(),
			HelmClient: helmClient,
			K8sClient:  cpK8sClients.K8sClient(),
			Logger:     logger,

			Namespace: namespace,
		}

		newRelease, err = release.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var logAnalyticsConfig LogAnalyticsConfig
	{
		logAnalyticsConfig = LogAnalyticsConfig{
			WorkspaceID: env.LogAnalyticsWorkspaceID(),
			SharedKey:   env.LogAnalyticsSharedKey(),
		}
	}

	c := Config{
		AzureClient:        azureClient,
		Guest:              guest,
		HelmClient:         helmClient,
		Host:               host,
		K8s:                k8sSetup,
		K8sClients:         cpK8sClients,
		Logger:             logger,
		Release:            newRelease,
		LogAnalyticsConfig: logAnalyticsConfig,
	}

	return c, nil
}

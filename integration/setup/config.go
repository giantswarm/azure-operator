package setup

import (
	"github.com/giantswarm/apprclient"
	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/e2e-harness/pkg/release"
	e2eclientsazure "github.com/giantswarm/e2eclients/azure"
	e2esetupenv "github.com/giantswarm/e2esetup/chart/env"
	"github.com/giantswarm/helmclient"
	k8sclientlegacy "github.com/giantswarm/k8sclient/v2/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v4/integration/env"
)

const (
	namespace    = "giantswarm"
	organization = "giantswarm"
	quayAddress  = "https://quay.io"
)

type Config struct {
	AzureClient      *e2eclientsazure.Client
	Guest            *framework.Guest
	HelmClient       helmclient.Interface
	Host             *framework.Host
	K8s              *k8sclient.Setup
	LegacyK8sClients k8sclientlegacy.Interface
	K8sClients       k8sclient.Interface
	Logger           micrologger.Logger
	Release          *release.Release
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

			KubeConfigPath: e2esetupenv.KubeConfigPath(),
		}

		cpK8sClients, err = k8sclient.NewClients(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var legacyCPK8sClients *k8sclientlegacy.Clients
	{
		c := k8sclientlegacy.ClientsConfig{
			Logger: logger,

			KubeConfigPath: e2esetupenv.KubeConfigPath(),
		}

		legacyCPK8sClients, err = k8sclientlegacy.NewClients(c)
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

	c := Config{
		AzureClient:      azureClient,
		Guest:            guest,
		HelmClient:       helmClient,
		Host:             host,
		K8s:              k8sSetup,
		K8sClients:       cpK8sClients,
		LegacyK8sClients: legacyCPK8sClients,
		Logger:           logger,
		Release:          newRelease,
	}

	return c, nil
}

package collector

import (
	"github.com/giantswarm/exporterkit/collector"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

type SetConfig struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
	Watcher   func(opts metav1.ListOptions) (watch.Interface, error)

	AzureSetting             setting.Azure
	HostAzureClientSetConfig client.AzureClientSetConfig
}

// Set is basically only a wrapper for the operator's collector implementations.
// It eases the iniitialization and prevents some weird import mess so we do not
// have to alias packages.
type Set struct {
	*collector.Set
}

func NewSet(config SetConfig) (*Set, error) {
	var err error

	var deploymentCollector *Deployment
	{
		c := DeploymentConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			Watcher:   config.Watcher,

			EnvironmentName: config.AzureSetting.Cloud,
		}

		deploymentCollector, err = NewDeployment(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var resourceGroupCollector *ResourceGroup
	{
		c := ResourceGroupConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			EnvironmentName: config.AzureSetting.Cloud,
		}

		resourceGroupCollector, err = NewResourceGroup(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var vpnConnectionCollector *VPNConnection
	{
		c := VPNConnectionConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			AzureSetting:             config.AzureSetting,
			HostAzureClientSetConfig: config.HostAzureClientSetConfig,
		}

		vpnConnectionCollector, err = NewVPNConnection(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var collectorSet *collector.Set
	{
		c := collector.SetConfig{
			Collectors: []collector.Interface{
				deploymentCollector,
				resourceGroupCollector,
				vpnConnectionCollector,
			},
			Logger: config.Logger,
		}

		collectorSet, err = collector.NewSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	s := &Set{
		Set: collectorSet,
	}

	return s, nil
}

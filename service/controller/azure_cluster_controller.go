package controller

import (
	"context"

	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/flag"
	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/azureclusterconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/azureconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/release"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

type AzureClusterConfig struct {
	InstallationName string
	K8sClient        k8sclient.Interface
	Logger           micrologger.Logger

	Flag  *flag.Flag
	Viper *viper.Viper

	Azure            setting.Azure
	CPAzureClientSet client.AzureClientSet
	ProjectName      string
	RegistryDomain   string

	Ignition         setting.Ignition
	OIDC             setting.OIDC
	SSOPublicKey     string
	TemplateVersion  string
	VMSSCheckWorkers int

	SentryDSN string
}

func NewAzureCluster(config AzureClusterConfig) (*controller.Controller, error) {
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
		resources, err = newAzureClusterResources(config, certsSearcher)
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
			// like operatorkit.giantswarm.io/azure-operator-azurecluster-controller.
			Name: project.Name() + "-azurecluster-controller",
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1alpha3.AzureCluster)
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

func newAzureClusterResources(config AzureClusterConfig, certsSearcher certs.Interface) ([]resource.Interface, error) {
	var err error

	var azureClusterConfigResource *azureclusterconfig.Resource
	{
		c := azureclusterconfig.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		azureClusterConfigResource, err = azureclusterconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var azureConfigResource *azureconfig.Resource
	{
		c := azureconfig.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,

			APIServerSecurePort: config.Viper.GetInt(config.Flag.Service.Cluster.Kubernetes.API.SecurePort),
			Calico: azureconfig.CalicoConfig{
				CIDRSize: config.Viper.GetInt(config.Flag.Service.Cluster.Calico.CIDR),
				MTU:      config.Viper.GetInt(config.Flag.Service.Cluster.Calico.MTU),
				Subnet:   config.Viper.GetString(config.Flag.Service.Cluster.Calico.Subnet),
			},
			ClusterIPRange:                 config.Viper.GetString(config.Flag.Service.Cluster.Kubernetes.API.ClusterIPRange),
			EtcdPrefix:                     config.Viper.GetString(config.Flag.Service.Cluster.Etcd.Prefix),
			ManagementClusterResourceGroup: config.Viper.GetString(config.Flag.Service.Azure.HostCluster.ResourceGroup),
			SSHUserList:                    config.Viper.GetString(config.Flag.Service.Cluster.Kubernetes.SSH.UserList),
		}

		azureConfigResource, err = azureconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var releaseResource resource.Interface
	{
		c := release.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		releaseResource, err = release.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		releaseResource,
		azureClusterConfigResource,
		azureConfigResource,
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

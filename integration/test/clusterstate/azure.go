// +build k8srequired

package clusterstate

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	azureclient "github.com/giantswarm/e2eclients/azure"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	VirtualMachineSize = "Standard_D4s_v3"
)

type ProviderConfig struct {
	AzureClient *azureclient.Client
	G8sClient   versioned.Interface
	Logger      micrologger.Logger

	ClusterID string
}

type Provider struct {
	azureClient *azureclient.Client
	g8sClient   versioned.Interface
	logger      micrologger.Logger

	clusterID string
}

func NewProvider(config ProviderConfig) (*Provider, error) {
	if config.AzureClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureClient must not be empty", config)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ClusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClusterID must not be empty", config)
	}

	p := &Provider{
		azureClient: config.AzureClient,
		g8sClient:   config.G8sClient,
		logger:      config.Logger,

		clusterID: config.ClusterID,
	}

	return p, nil
}

func (p *Provider) RebootMaster() error {
	resourceGroupName := p.clusterID
	scaleSetName := fmt.Sprintf("%s-master", p.clusterID)

	scaleSetVMs, err := p.azureClient.VirtualMachineScaleSetVMsClient.List(context.TODO(), resourceGroupName, scaleSetName, "", "", "")
	if err != nil {
		return microerror.Mask(err)
	}

	vmList := scaleSetVMs.Values()
	if len(vmList) == 0 {
		return microerror.Maskf(notFoundError, "scale set '%s' has no vms", scaleSetName)
	} else if len(vmList) > 1 {
		return microerror.Maskf(tooManyResultsError, "scale set '%s' has %d vms", scaleSetName, len(vmList))
	}

	instanceID := vmList[0].InstanceID
	instanceIDs := &compute.VirtualMachineScaleSetVMInstanceIDs{
		InstanceIds: to.StringSlicePtr([]string{
			*instanceID,
		}),
	}
	_, err = p.azureClient.VirtualMachineScaleSetsClient.Restart(context.TODO(), resourceGroupName, scaleSetName, instanceIDs)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (p *Provider) ReplaceMaster() error {
	customObject, err := p.g8sClient.ProviderV1alpha1().AzureConfigs("default").Get(p.clusterID, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	// Change virtual machine size to trigger replacement of existing master node.
	customObject.Spec.Azure.Masters[0].VMSize = VirtualMachineSize

	_, err = p.g8sClient.ProviderV1alpha1().AzureConfigs("default").Update(customObject)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

package deployment

import (
	"context"
	"io/ioutil"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/markbates/pkger"

	"github.com/giantswarm/azure-operator/v3/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v3/service/controller/key"
)

func (r Resource) newDeployment(ctx context.Context, customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	defaultParams := map[string]interface{}{
		"blobContainerName":       key.BlobContainerName(),
		"calicoSubnetCidr":        cc.AzureNetwork.Calico.String(),
		"clusterID":               key.ClusterID(customObject),
		"dnsZones":                key.DNSZones(customObject),
		"hostClusterCidr":         r.azure.HostCluster.CIDR,
		"kubernetesAPISecurePort": key.APISecurePort(customObject),
		"masterSubnetCidr":        cc.AzureNetwork.Master.String(),
		"storageAccountName":      key.StorageAccountName(customObject),
		"virtualNetworkCidr":      key.VnetCIDR(customObject),
		"virtualNetworkName":      key.VnetName(customObject),
		"vnetGatewaySubnetName":   key.VNetGatewaySubnetName(),
		"vpnSubnetCidr":           cc.AzureNetwork.VPN.String(),
		"workerSubnetCidr":        cc.AzureNetwork.Worker.String(),
	}

	f, err := pkger.Open("/service/controller/resource/deployment/template/main.json")
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			Template:   string(b),
		},
	}

	return d, nil
}

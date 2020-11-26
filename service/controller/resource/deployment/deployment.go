package deployment

import (
	"context"
	"fmt"
	"net"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/deployment/template"
	"github.com/giantswarm/azure-operator/v5/service/network"
)

func (r Resource) newDeployment(ctx context.Context, customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	// The VPN subnet is not persisted in the AzureConfig so I have to compute it now.
	// This is suboptimal, but will not be needed anymore once we switch to vnet peering
	// and that will hopefully happen soon.
	vpnSubnet, err := getVPNSubnet(customObject)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	controlPlaneWorkerSubnetID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s_worker_subnet",
		r.controlPlaneSubscriptionID,
		r.installationName,
		r.azure.HostCluster.VirtualNetwork,
		r.installationName,
	)

	controlPlanePublicIPs, err := r.getControlPlanePublicIPs(ctx)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	defaultParams := map[string]interface{}{
		"blobContainerName":          key.BlobContainerName(),
		"calicoSubnetCidr":           key.CalicoCIDR(customObject),
		"controlPlaneWorkerSubnetID": controlPlaneWorkerSubnetID,
		"clusterID":                  key.ClusterID(&customObject),
		"dnsZones":                   key.DNSZones(customObject),
		"hostClusterCidr":            r.azure.HostCluster.CIDR,
		"hostPublicIPs":              controlPlanePublicIPs,
		"insecureStorageAccount":     r.debug.InsecureStorageAccount,
		"kubernetesAPISecurePort":    key.APISecurePort(customObject),
		"masterSubnetCidr":           key.MastersSubnetCIDR(customObject),
		"storageAccountName":         key.StorageAccountName(&customObject),
		"virtualNetworkCidr":         key.VnetCIDR(customObject),
		"virtualNetworkName":         key.VnetName(customObject),
		"vnetGatewaySubnetName":      key.VNetGatewaySubnetName(),
		"vpnSubnetCidr":              vpnSubnet.String(),
		"workerSubnetCidr":           key.WorkersSubnetCIDR(customObject),
	}

	armTemplate, err := template.GetARMTemplate()
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			Template:   armTemplate,
		},
	}

	return d, nil
}

// This function retrieves the public IP address for CP masters and workers, as a comma separated list.
func (r Resource) getControlPlanePublicIPs(ctx context.Context) ([]string, error) {
	allPublicIPs, err := r.azureClientSet.PublicIpAddressesClient.ListComplete(ctx, r.installationName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var ips []string
	for allPublicIPs.NotDone() {
		ip := allPublicIPs.Value()
		// Masters use the API LB as egress gateway, the workers use the ingress LB.
		if ip.Name != nil && *ip.Name == fmt.Sprintf("%s_ingress_ip", r.installationName) || *ip.Name == fmt.Sprintf("%s_api_ip", r.installationName) {
			ips = append(ips, *ip.IPAddress)
		}
		err := allPublicIPs.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return ips, nil
}

func getVPNSubnet(customObject providerv1alpha1.AzureConfig) (*net.IPNet, error) {
	_, netw, err := net.ParseCIDR(customObject.Spec.Azure.VirtualNetwork.CIDR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	subnets, err := network.Compute(*netw)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &subnets.VPN, nil
}

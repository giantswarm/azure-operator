package deployment

import (
	"context"
	"fmt"
	"net"
	"strings"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/deployment/template"
	"github.com/giantswarm/azure-operator/v4/service/network"
)

const (
	IsInitialProvisioning = "Yes"
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

	initialProvisioning, err := r.initialProvisioning(ctx, customObject)
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
		"initialProvisioning":        initialProvisioning,
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

func (r *Resource) initialProvisioning(ctx context.Context, customObject providerv1alpha1.AzureConfig) (string, error) {
	subnetsClient, err := r.clientFactory.GetSubnetsClient(customObject.Spec.Azure.CredentialSecret.Namespace, customObject.Spec.Azure.CredentialSecret.Name)
	if err != nil {
		return IsInitialProvisioning, microerror.Mask(err)
	}

	result, err := subnetsClient.ListComplete(ctx, key.ClusterID(&customObject), key.VnetName(customObject))
	if IsNotFound(err) {
		return IsInitialProvisioning, nil
	} else if err != nil {
		return IsInitialProvisioning, microerror.Mask(err)
	}

	expectedSubnets := 0
	for result.NotDone() {
		subnet := result.Value()
		if strings.Contains(*subnet.Name, key.VNetGatewaySubnetName()) {
			expectedSubnets++
		} else if strings.Contains(*subnet.Name, "MasterSubnet") {
			expectedSubnets++
		} else if strings.Contains(*subnet.Name, "WorkerSubnet") {
			expectedSubnets++
		}

		err = result.NextWithContext(ctx)
		if err != nil {
			return IsInitialProvisioning, microerror.Mask(err)
		}
	}

	initialProvisioning := "No"
	if expectedSubnets != 3 {
		initialProvisioning = IsInitialProvisioning
	}

	return initialProvisioning, nil
}

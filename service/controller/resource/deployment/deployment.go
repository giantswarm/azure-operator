package deployment

import (
	"context"
	"fmt"
	"net"
	"strconv"

	azurenetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/deployment/template"
	"github.com/giantswarm/azure-operator/v4/service/network"
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

	masterSecurityRules, err := r.getMasterSecurityRules(ctx, customObject)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	workersSecurityRules, err := r.getWorkersSecurityRules(ctx, customObject)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	defaultParams := map[string]interface{}{
		"blobContainerName":          key.BlobContainerName(),
		"calicoSubnetCidr":           key.CalicoCIDR(customObject),
		"controlPlaneWorkerSubnetID": controlPlaneWorkerSubnetID,
		"clusterID":                  key.ClusterID(&customObject),
		"dnsZones":                   key.DNSZones(customObject),
		"masterSubnetCidr":           key.MastersSubnetCIDR(customObject),
		"storageAccountName":         key.StorageAccountName(&customObject),
		"virtualNetworkCidr":         key.VnetCIDR(customObject),
		"virtualNetworkName":         key.VnetName(customObject),
		"vnetGatewaySubnetName":      key.VNetGatewaySubnetName(),
		"vpnSubnetCidr":              vpnSubnet.String(),
		"workerSubnetCidr":           key.WorkersSubnetCIDR(customObject),
		"masterSecurityRules":        masterSecurityRules,
		"workersSecurityRules":       workersSecurityRules,
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

func (r Resource) getMasterSecurityRules(ctx context.Context, customObject providerv1alpha1.AzureConfig) ([]azurenetwork.SecurityRule, error) {
	defaultRules := []azurenetwork.SecurityRule{
		{
			Name: to.StringPtr("defaultInboundRule"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Default rule that denies any inbound traffic."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				SourceAddressPrefix:      to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   "Deny",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(4096),
			},
		},
		{
			Name: to.StringPtr("defaultOutboundRule"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Default rule that allows any outbound traffic."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				SourceAddressPrefix:      to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   "Allow",
				Direction:                "Outbound",
				Priority:                 to.Int32Ptr(4095),
			},
		},
		{
			Name: to.StringPtr("defaultInClusterRule"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Default rule that allows any traffic within the master subnet."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				SourceAddressPrefix:      to.StringPtr(key.MastersSubnetCIDR(customObject)),
				DestinationAddressPrefix: to.StringPtr(key.VnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(4094),
			},
		},
		{
			Name: to.StringPtr("sshHostClusterToMasterSubnetRule"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow the host cluster machines to reach this guest cluster master subnet."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("22"),
				SourceAddressPrefix:      to.StringPtr(r.azure.HostCluster.CIDR),
				DestinationAddressPrefix: to.StringPtr(key.MastersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(4093),
			},
		},
		{
			Name: to.StringPtr("apiLoadBalancerRule"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow anyone to reach the kubernetes API."),
				Protocol:                 "tcp",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr(strconv.Itoa(key.APISecurePort(customObject))),
				SourceAddressPrefix:      to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr(key.MastersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3903),
			},
		},
		{
			Name: to.StringPtr("allowEtcdLoadBalancer"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow traffic from LB to master instance."),
				Protocol:                 "tcp",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("2379"),
				SourceAddressPrefix:      to.StringPtr("AzureLoadBalancer"),
				DestinationAddressPrefix: to.StringPtr(key.MastersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3902),
			},
		},
		{
			Name: to.StringPtr("etcdLoadBalancerRuleHost"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow host cluster to reach the etcd loadbalancer."),
				Protocol:                 "tcp",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("2379"),
				SourceAddressPrefix:      to.StringPtr(r.azure.HostCluster.CIDR),
				DestinationAddressPrefix: to.StringPtr(key.MastersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3901),
			},
		},
		{
			Name: to.StringPtr("etcdLoadBalancerRuleCluster"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow cluster subnet to reach the etcd loadbalancer."),
				Protocol:                 "tcp",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("2379"),
				SourceAddressPrefix:      to.StringPtr(key.VnetCIDR(customObject)),
				DestinationAddressPrefix: to.StringPtr(key.MastersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3900),
			},
		},
		{
			Name: to.StringPtr("allowWorkerSubnet"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow the worker machines to reach the master machines on any ports."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				SourceAddressPrefix:      to.StringPtr(key.WorkersSubnetCIDR(customObject)),
				DestinationAddressPrefix: to.StringPtr(key.VnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3800),
			},
		},
		{
			Name: to.StringPtr("allowCalicoSubnet"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow pods to reach the master machines on any ports."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				SourceAddressPrefix:      to.StringPtr(key.CalicoCIDR(customObject)),
				DestinationAddressPrefix: to.StringPtr(key.VnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3700),
			},
		},
		{
			Name: to.StringPtr("allowCadvisor"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow host cluster Prometheus to reach Cadvisors."),
				Protocol:                 "tcp",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("4194"),
				SourceAddressPrefix:      to.StringPtr(r.azure.HostCluster.CIDR),
				DestinationAddressPrefix: to.StringPtr(key.MastersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3500),
			},
		},
		{
			Name: to.StringPtr("allowKubelet"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow host cluster Prometheus to reach Kubelets."),
				Protocol:                 "tcp",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("10250"),
				SourceAddressPrefix:      to.StringPtr(r.azure.HostCluster.CIDR),
				DestinationAddressPrefix: to.StringPtr(key.MastersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3501),
			},
		},
		{
			Name: to.StringPtr("allowNodeExporter"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow host cluster Prometheus to reach node-exporters."),
				Protocol:                 "tcp",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("10300"),
				SourceAddressPrefix:      to.StringPtr(r.azure.HostCluster.CIDR),
				DestinationAddressPrefix: to.StringPtr(key.MastersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3502),
			},
		},
	}

	securityRulesClient, err := r.clientFactory.GetSecurityRulesClient(customObject.Spec.Azure.CredentialSecret.Namespace, customObject.Spec.Azure.CredentialSecret.Name)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	iterator, err := securityRulesClient.ListComplete(ctx, customObject.Name, key.MasterSecurityGroupName(customObject))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for iterator.NotDone() {
		rule := iterator.Value()

		// Check if rule is a default one by comparing the name.
		isDefault := false
		{
			fmt.Printf("Checking if rule %s is a default one\n", *rule.Name)
			for _, candidate := range defaultRules {
				if *rule.Name == *candidate.Name {
					isDefault = true
					fmt.Printf("Rule %s is a default one\n", *rule.Name)
					break
				}
				fmt.Printf("   %s != %s\n", *rule.Name, *candidate.Name)
			}
		}

		if !isDefault {
			fmt.Printf("Rule %s is NOT a default one\n", *rule.Name)
			// Rule is not a default one, we want to keep it.
			defaultRules = append(defaultRules, rule)
		}

		err = iterator.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return defaultRules, nil
}

func (r Resource) getWorkersSecurityRules(ctx context.Context, customObject providerv1alpha1.AzureConfig) ([]azurenetwork.SecurityRule, error) {
	defaultRules := []azurenetwork.SecurityRule{
		{
			Name: to.StringPtr("defaultInboundRule"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Default rule that denies any inbound traffic."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				SourceAddressPrefix:      to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   "Deny",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(4096),
			},
		},
		{
			Name: to.StringPtr("defaultOutboundRule"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Default rule that allows any outbound traffic."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				SourceAddressPrefix:      to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   "Allow",
				Direction:                "Outbound",
				Priority:                 to.Int32Ptr(4095),
			},
		},
		{
			Name: to.StringPtr("defaultInClusterRule"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Default rule that allows any traffic within the worker subnet."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				SourceAddressPrefix:      to.StringPtr(key.WorkersSubnetCIDR(customObject)),
				DestinationAddressPrefix: to.StringPtr(key.VnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(4094),
			},
		},
		{
			Name: to.StringPtr("sshHostClusterToWorkerSubnetRule"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow the host cluster machines to reach this guest cluster worker subnet."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("22"),
				SourceAddressPrefix:      to.StringPtr(r.azure.HostCluster.CIDR),
				DestinationAddressPrefix: to.StringPtr(key.WorkersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(4093),
			},
		},
		{
			Name: to.StringPtr("azureLoadBalancerHealthChecks"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow Azure Load Balancer health checks."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				SourceAddressPrefix:      to.StringPtr("AzureLoadBalancer"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(4000),
			},
		},
		{
			Name: to.StringPtr("allowMasterSubnet"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow the master machines to reach the worker machines on any ports."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				SourceAddressPrefix:      to.StringPtr(key.MastersSubnetCIDR(customObject)),
				DestinationAddressPrefix: to.StringPtr(key.VnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3700),
			},
		},
		{
			Name: to.StringPtr("allowCalicoSubnet"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow pods to reach the worker machines on any ports."),
				Protocol:                 "*",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				SourceAddressPrefix:      to.StringPtr(key.CalicoCIDR(customObject)),
				DestinationAddressPrefix: to.StringPtr(key.VnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3600),
			},
		},
		{
			Name: to.StringPtr("allowCadvisor"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow host cluster Prometheus to reach Cadvisors."),
				Protocol:                 "tcp",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("4194"),
				SourceAddressPrefix:      to.StringPtr(r.azure.HostCluster.CIDR),
				DestinationAddressPrefix: to.StringPtr(key.WorkersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3500),
			},
		},
		{
			Name: to.StringPtr("allowKubelet"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow host cluster Prometheus to reach Kubelets."),
				Protocol:                 "tcp",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("10250"),
				SourceAddressPrefix:      to.StringPtr(r.azure.HostCluster.CIDR),
				DestinationAddressPrefix: to.StringPtr(key.WorkersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3501),
			},
		},
		{
			Name: to.StringPtr("allowNodeExporter"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow host cluster Prometheus to reach node-exporters."),
				Protocol:                 "tcp",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("10300"),
				SourceAddressPrefix:      to.StringPtr(r.azure.HostCluster.CIDR),
				DestinationAddressPrefix: to.StringPtr(key.WorkersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3502),
			},
		},
		{
			Name: to.StringPtr("allowKubeStateMetrics"),
			SecurityRulePropertiesFormat: &azurenetwork.SecurityRulePropertiesFormat{
				Description:              to.StringPtr("Allow host cluster Prometheus to reach kube-state-metrics."),
				Protocol:                 "tcp",
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("10301"),
				SourceAddressPrefix:      to.StringPtr(r.azure.HostCluster.CIDR),
				DestinationAddressPrefix: to.StringPtr(key.WorkersSubnetCIDR(customObject)),
				Access:                   "Allow",
				Direction:                "Inbound",
				Priority:                 to.Int32Ptr(3503),
			},
		},
	}

	securityRulesClient, err := r.clientFactory.GetSecurityRulesClient(customObject.Spec.Azure.CredentialSecret.Namespace, customObject.Spec.Azure.CredentialSecret.Name)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	iterator, err := securityRulesClient.ListComplete(ctx, customObject.Name, key.WorkerSecurityGroupName(customObject))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for iterator.NotDone() {
		rule := iterator.Value()

		// Check if rule is a default one by comparing the name.
		isDefault := false
		{
			fmt.Printf("Checking if rule %s is a default one\n", *rule.Name)
			for _, candidate := range defaultRules {
				if *rule.Name == *candidate.Name {
					isDefault = true
					fmt.Printf("Rule %s is a default one\n", *rule.Name)
					break
				}
				fmt.Printf("   %s != %s\n", *rule.Name, *candidate.Name)
			}
		}

		if !isDefault {
			fmt.Printf("Rule %s is NOT a default one\n", *rule.Name)
			// Rule is not a default one, we want to keep it.
			defaultRules = append(defaultRules, rule)
		}

		err = iterator.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return defaultRules, nil
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

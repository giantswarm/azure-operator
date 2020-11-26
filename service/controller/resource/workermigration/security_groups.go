package workermigration

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/workermigration/internal/azure"
)

// Security group rules that need destination CIDR update from built-in worker subnet to VNET CIDR.
var workerSecurityGroupRulesToUpdate = []string{
	"allowCadvisor",
	"allowKubelet",
	"allowNodeExporter",
	"allowKubeStateMetrics",
	"defaultInClusterRule",
	"sshHostClusterToWorkerSubnetRule",
}

func (r *Resource) ensureSecurityGroupRulesUpdated(ctx context.Context, cr providerv1alpha1.AzureConfig, azureAPI azure.API) error {
	var err error
	var securityGroups azure.SecurityGroups
	{
		securityGroups, err = azureAPI.ListNetworkSecurityGroups(ctx, key.ResourceGroupName(cr))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	for _, sg := range securityGroups {
		if sg.Name != nil && *sg.Name == key.MasterSecurityGroupName(cr) {
			err := r.ensureMasterEtcdLBSourcePrefixesUpdated(ctx, cr, azureAPI, sg)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		if sg.Name != nil && *sg.Name == key.WorkerSecurityGroupName(cr) {
			err := r.ensureWorkerNetworkRuleCIDRUpdated(ctx, cr, azureAPI, sg, workerSecurityGroupRulesToUpdate)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	return nil
}

func (r *Resource) ensureMasterEtcdLBSourcePrefixesUpdated(ctx context.Context, cr providerv1alpha1.AzureConfig, azureAPI azure.API, securityGroup network.SecurityGroup) error {
	if securityGroup.SecurityGroupPropertiesFormat == nil || securityGroup.SecurityGroupPropertiesFormat.SecurityRules == nil {
		return microerror.Maskf(executionFailedError, "security group rules are missing from Azure API response")
	}

	var err error
	var publicIPs []string
	{
		publicIPs, err = azureAPI.ListPublicIPs(ctx, r.installationName)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var needUpdate bool
	for i, rule := range *securityGroup.SecurityRules {
		if rule.Name != nil && *rule.Name == "etcdLoadBalancerRuleHost" {
			if rule.SourceAddressPrefix == nil || len(*rule.SourceAddressPrefix) > 0 {
				(*securityGroup.SecurityRules)[i].SourceAddressPrefix = nil
				(*securityGroup.SecurityRules)[i].SourceAddressPrefixes = &publicIPs
				needUpdate = true
			}
		}
	}

	if needUpdate {
		err := azureAPI.CreateOrUpdateNetworkSecurityGroup(ctx, key.ResourceGroupName(cr), *securityGroup.Name, securityGroup)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) ensureWorkerNetworkRuleCIDRUpdated(ctx context.Context, cr providerv1alpha1.AzureConfig, azureAPI azure.API, securityGroup network.SecurityGroup, rulesToUpdate []string) error {
	if securityGroup.SecurityGroupPropertiesFormat == nil || securityGroup.SecurityGroupPropertiesFormat.SecurityRules == nil {
		return microerror.Maskf(executionFailedError, "security group rules are missing from Azure API response")
	}

	var needUpdate bool
	for i, rule := range *securityGroup.SecurityRules {
		if rule.Name != nil && contains(rulesToUpdate, *rule.Name) {
			if rule.SourceAddressPrefix == nil || *rule.SourceAddressPrefix == key.WorkersSubnetCIDR(cr) {
				(*securityGroup.SecurityRules)[i].SourceAddressPrefix = to.StringPtr(key.VnetCIDR(cr))
				needUpdate = true
			}

			if rule.DestinationAddressPrefix == nil || *rule.DestinationAddressPrefix == key.WorkersSubnetCIDR(cr) {
				(*securityGroup.SecurityRules)[i].DestinationAddressPrefix = to.StringPtr(key.VnetCIDR(cr))
				needUpdate = true
			}
		}
	}

	if needUpdate {
		err := azureAPI.CreateOrUpdateNetworkSecurityGroup(ctx, key.ResourceGroupName(cr), *securityGroup.Name, securityGroup)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}

	return false
}

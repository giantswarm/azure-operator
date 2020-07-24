package deployment

import (
	"context"
	"fmt"
	"strconv"

	azurenetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r Resource) getMasterSecurityRules(ctx context.Context, customObject providerv1alpha1.AzureConfig) ([]azurenetwork.SecurityRule, error) {
	return r.getCustomSecurityRules(ctx, customObject, r.getDefaultMasterSecurityRules(customObject), key.MasterSecurityGroupName(customObject))
}

func (r Resource) getWorkersSecurityRules(ctx context.Context, customObject providerv1alpha1.AzureConfig) ([]azurenetwork.SecurityRule, error) {
	return r.getCustomSecurityRules(ctx, customObject, r.getDefaultWorkersSecurityRules(customObject), key.WorkerSecurityGroupName(customObject))
}

func (r Resource) getCustomSecurityRules(ctx context.Context, customObject providerv1alpha1.AzureConfig, defaultRules []azurenetwork.SecurityRule, sgName string) ([]azurenetwork.SecurityRule, error) {
	securityRulesClient, err := r.clientFactory.GetSecurityRulesClient(customObject.Spec.Azure.CredentialSecret.Namespace, customObject.Spec.Azure.CredentialSecret.Name)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	iterator, err := securityRulesClient.ListComplete(ctx, customObject.Name, sgName)
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("SecurityGroup %s not found: unable to check for existing security rules.", sgName))
		return defaultRules, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	for iterator.NotDone() {
		rule := iterator.Value()

		// Check if rule is a default one by comparing the name.
		isDefault := false
		{
			for _, candidate := range defaultRules {
				if *rule.Name == *candidate.Name {
					isDefault = true
					break
				}
			}
		}

		if !isDefault {
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

func (r Resource) getDefaultMasterSecurityRules(customObject providerv1alpha1.AzureConfig) []azurenetwork.SecurityRule {
	return []azurenetwork.SecurityRule{
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
}

func (r Resource) getDefaultWorkersSecurityRules(customObject providerv1alpha1.AzureConfig) []azurenetwork.SecurityRule {
	return []azurenetwork.SecurityRule{
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
}

package masters

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

const (
	temporarySecurityRuleName = "temporailyBlockApiAccess"
)

func (r *Resource) blockAPICallsTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	exists, err := r.securityRuleExists(ctx, key.ResourceGroupName(cr), key.MasterSecurityGroupName(cr), temporarySecurityRuleName)
	if err != nil {
		// In case of error just retry.
		return currentState, microerror.Mask(err)
	}

	if !exists {
		// Create security rule
		err = r.createSecurityRule(ctx, key.ResourceGroupName(cr), key.MasterSecurityGroupName(cr), temporarySecurityRuleName)
		if err != nil {
			// In case of error just retry.
			return currentState, microerror.Mask(err)
		}
	}

	return DeploymentUninitialized, nil
}

func (r *Resource) unblockAPICallsTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	// Delete security rule
	err = r.deleteSecurityRule(ctx, key.ResourceGroupName(cr), key.MasterSecurityGroupName(cr), temporarySecurityRuleName)
	if IsNotFound(err) {
		// Rule not exists, ok to continue.
		return DeploymentCompleted, nil
	} else if err != nil {
		// In case of error just retry.
		return currentState, microerror.Mask(err)
	}

	return DeploymentCompleted, nil
}

func (r *Resource) createSecurityRule(ctx context.Context, resourceGroup string, securityGroupname string, securityRuleName string) error {
	c, err := r.getSecurityRulesClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.CreateOrUpdate(context.Background(), resourceGroup, securityGroupname, securityRuleName, network.SecurityRule{
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access:                   network.SecurityRuleAccessDeny,
			Description:              to.StringPtr("Temporarily block API access during flatcar migration"),
			DestinationPortRange:     to.StringPtr("443"),
			DestinationAddressPrefix: to.StringPtr("*"),
			Direction:                network.SecurityRuleDirectionInbound,
			Protocol:                 network.SecurityRuleProtocolAsterisk,
			Priority:                 to.Int32Ptr(3000),
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
		},
	})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) securityRuleExists(ctx context.Context, resourceGroup string, securityGroupname string, securityRuleName string) (bool, error) {
	c, err := r.getSecurityRulesClient(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	_, err = c.Get(context.Background(), resourceGroup, securityGroupname, securityRuleName)
	if IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func (r *Resource) deleteSecurityRule(ctx context.Context, resourceGroup string, securityGroupname string, securityRuleName string) error {
	c, err := r.getSecurityRulesClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.Delete(context.Background(), resourceGroup, securityGroupname, securityRuleName)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

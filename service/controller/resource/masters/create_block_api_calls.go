package masters

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	temporarySecurityRuleName = "temporaryFlatcarMigration"
)

func (r *Resource) blockAPICallsTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	workersRule := network.SecurityRule{
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access:                   network.SecurityRuleAccessDeny,
			Description:              to.StringPtr("Temporarily block API access from workers during flatcar migration"),
			DestinationPortRange:     to.StringPtr("443"),
			DestinationAddressPrefix: to.StringPtr("*"),
			Direction:                network.SecurityRuleDirectionOutbound,
			Protocol:                 network.SecurityRuleProtocolAsterisk,
			Priority:                 to.Int32Ptr(3000),
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
		},
	}

	mastersRule := network.SecurityRule{
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access:                   network.SecurityRuleAccessDeny,
			Description:              to.StringPtr("Temporarily block internet access to masters during flatcar migration"),
			DestinationPortRange:     to.StringPtr("*"),
			DestinationAddressPrefix: to.StringPtr("*"),
			Direction:                network.SecurityRuleDirectionOutbound,
			Protocol:                 network.SecurityRuleProtocolAsterisk,
			Priority:                 to.Int32Ptr(3000),
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
		},
	}

	workerFound, err := r.ensureSecurityRule(ctx, key.ResourceGroupName(cr), key.WorkerSecurityGroupName(cr), temporarySecurityRuleName, workersRule)
	if err != nil {
		return "", microerror.Mask(err)
	}

	masterFound, err := r.ensureSecurityRule(ctx, key.ResourceGroupName(cr), key.MasterSecurityGroupName(cr), temporarySecurityRuleName, mastersRule)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if !masterFound || !workerFound {
		r.logger.LogCtx(ctx, "level", "debug", "message", "Security rules not in place yet")
		return currentState, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Security rules found")

	return DeploymentUninitialized, nil
}

func (r *Resource) ensureSecurityRule(ctx context.Context, resourceGroup string, securityGroupName string, ruleName string, rule network.SecurityRule) (bool, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Looking for security rule %s in security group %s", ruleName, securityGroupName))

	exists, err := r.securityRuleExists(ctx, resourceGroup, securityGroupName, ruleName)
	if err != nil {
		return false, microerror.Mask(err)
	}

	if !exists {
		// Create security rule
		r.logger.LogCtx(ctx, "level", "debug", "message", "Creating security rule")
		err = r.createSecurityRule(ctx, resourceGroup, securityGroupName, ruleName, rule)
		if err != nil {
			return false, microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "Security rule created")

		// Wait for security rule to be in place.
		return false, nil
	}

	// Security rule in place
	return true, nil
}

func (r *Resource) unblockAPICallsTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting security rule %s from security group %s", temporarySecurityRuleName, key.WorkerSecurityGroupName(cr)))

	// Delete security rule for workers
	err = r.deleteSecurityRule(ctx, key.ResourceGroupName(cr), key.WorkerSecurityGroupName(cr), temporarySecurityRuleName)
	if IsNotFound(err) {
		// Rule not exists, ok to continue.
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("security rule %s from security group %s was not found", temporarySecurityRuleName, key.WorkerSecurityGroupName(cr)))
	} else if err != nil {
		// In case of error just retry.
		return currentState, microerror.Mask(err)
	}

	// Delete security rule for masters
	err = r.deleteSecurityRule(ctx, key.ResourceGroupName(cr), key.MasterSecurityGroupName(cr), temporarySecurityRuleName)
	if IsNotFound(err) {
		// Rule not exists, ok to continue.
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("security rule %s from security group %s was not found", temporarySecurityRuleName, key.MasterSecurityGroupName(cr)))
	} else if err != nil {
		// In case of error just retry.
		return currentState, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "deleted temporary security rules")

	return RestartKubeletOnWorkers, nil
}

func (r *Resource) createSecurityRule(ctx context.Context, resourceGroup string, securityGroupname string, securityRuleName string, securityRule network.SecurityRule) error {
	c, err := r.getSecurityRulesClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.CreateOrUpdate(context.Background(), resourceGroup, securityGroupname, securityRuleName, securityRule)
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

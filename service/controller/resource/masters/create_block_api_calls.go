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

type namedRule struct {
	name string
	rule *network.SecurityRule
}

func (r *Resource) blockAPICallsTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	rules := getMasterRules()

	masterFound, err := r.ensureSecurityRules(ctx, key.ResourceGroupName(cr), key.MasterSecurityGroupName(cr), rules)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if !masterFound {
		r.logger.LogCtx(ctx, "level", "debug", "message", "Security rules not in place yet")
		return currentState, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Security rules found")

	return DeploymentUninitialized, nil
}

func (r *Resource) ensureSecurityRules(ctx context.Context, resourceGroup string, securityGroupName string, rules []namedRule) (bool, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Looking for existence of %d security rules in security group %s", len(rules), securityGroupName))

	found := true
	for _, rule := range rules {
		ruleName := rule.name
		exists, err := r.securityRuleExists(ctx, resourceGroup, securityGroupName, ruleName)
		if err != nil {
			return false, microerror.Mask(err)
		}

		if !exists {
			found = false

			// Create security rule
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Creating security rule %s", *rule.rule.Description))
			err = r.createSecurityRule(ctx, resourceGroup, securityGroupName, ruleName, *rule.rule)
			if err != nil {
				return false, microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Created security rule %s", *rule.rule.Description))
		}
	}

	// The 'found' bool is true if all rules are in place, false otherwise.
	return found, nil
}

func (r *Resource) unblockAPICallsTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting security rule %s from security group %s", temporarySecurityRuleName, key.WorkerSecurityGroupName(cr)))

	rules := getMasterRules()

	for _, namedRule := range rules {
		// Delete security rule for masters.
		err = r.deleteSecurityRule(ctx, key.ResourceGroupName(cr), key.MasterSecurityGroupName(cr), namedRule.name)
		if IsNotFound(err) {
			// Rule not exists, ok to continue.
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("security rule %s from security group %s was not found", namedRule.name, key.MasterSecurityGroupName(cr)))
		} else if err != nil {
			// In case of error just retry.
			return currentState, microerror.Mask(err)
		}
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

func getMasterRules() []namedRule {
	mastersRules := []namedRule{
		{
			name: fmt.Sprintf("%s-outbound", temporarySecurityRuleName),
			rule: &network.SecurityRule{
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
			},
		},
		{
			name: fmt.Sprintf("%s-inbound", temporarySecurityRuleName),
			rule: &network.SecurityRule{
				SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
					Access:                   network.SecurityRuleAccessDeny,
					Description:              to.StringPtr("Temporarily block incoming traffic to masters' 443 port during flatcar migration"),
					DestinationPortRange:     to.StringPtr("443"),
					DestinationAddressPrefix: to.StringPtr("*"),
					Direction:                network.SecurityRuleDirectionInbound,
					Protocol:                 network.SecurityRuleProtocolAsterisk,
					Priority:                 to.Int32Ptr(3001),
					SourceAddressPrefix:      to.StringPtr("*"),
					SourcePortRange:          to.StringPtr("*"),
				},
			},
		},
	}

	return mastersRules
}

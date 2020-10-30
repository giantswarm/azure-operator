package subnet

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/reconciliationcanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/client"
	"github.com/giantswarm/azure-operator/v5/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	subnet "github.com/giantswarm/azure-operator/v5/service/controller/resource/subnet/template"
)

const (
	mainDeploymentName = "subnet"
	// Name is the identifier of the resource.
	Name = "subnet"
)

type Config struct {
	AzureClientsFactory client.OrganizationFactory
	CtrlClient          ctrlclient.Client
	Debugger            *debugger.Debugger
	Logger              micrologger.Logger
}

// Resource creates a different subnet for every node pool using ARM deployments.
type Resource struct {
	azureClientsFactory client.OrganizationFactory
	ctrlClient          ctrlclient.Client
	debugger            *debugger.Debugger
	logger              micrologger.Logger
}

type StorageAccountIpRule struct {
	Value  string `json:"value"`
	Action string `json:"action"`
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		azureClientsFactory: config.AzureClientsFactory,
		ctrlClient:          config.CtrlClient,
		debugger:            config.Debugger,
		logger:              config.Logger,
	}

	return r, nil
}

// For every subnet declared in the `AzureCluster.Spec.NetworkSpec.Subnets` field, we submit a deployment to Azure to create the subnet.
// The ipam handler is the one updating AzureCluster with the required subnets.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureCluster, err := key.ToAzureCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsClient, err := r.azureClientsFactory.GetDeploymentsClient(ctx, azureCluster.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	natGatewaysClient, err := r.azureClientsFactory.GetNatGatewaysClient(ctx, azureCluster.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	storageAccountsClient, err := r.azureClientsFactory.GetStorageAccountsClient(ctx, azureCluster.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	subnetsClient, err := r.azureClientsFactory.GetSubnetsClient(ctx, azureCluster.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.garbageCollectSubnets(ctx, deploymentsClient, subnetsClient, azureCluster)
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "message", "resources not ready")
		r.logger.LogCtx(ctx, "message", "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureSubnets(ctx, deploymentsClient, storageAccountsClient, natGatewaysClient, azureCluster)
	if IsNatGatewayNotReadyError(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "Nat Gateway needs to be in state 'Succeeded' before subnets can be created")
		r.logger.LogCtx(ctx, "message", "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureSubnets(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, storageAccountsClient *storage.AccountsClient, natGatewaysClient *network.NatGatewaysClient, azureCluster capzv1alpha3.AzureCluster) error {
	armTemplate, err := subnet.GetARMTemplate()
	if err != nil {
		return microerror.Mask(err)
	}

	natGw, err := natGatewaysClient.Get(ctx, key.ClusterID(&azureCluster), "workers-nat-gw", "")
	if IsNotFound(err) {
		return microerror.Mask(natGatewayNotReadyError)
	} else if err != nil {
		return microerror.Mask(err)
	}

	if natGw.ProvisioningState != network.Succeeded {
		return microerror.Mask(natGatewayNotReadyError)
	}

	for i := 0; i < len(azureCluster.Spec.NetworkSpec.Subnets); i++ {
		deploymentName := key.SubnetDeploymentName(azureCluster.Spec.NetworkSpec.Subnets[i].Name)
		currentDeployment, err := deploymentsClient.Get(ctx, key.ClusterID(&azureCluster), deploymentName)
		if IsNotFound(err) {
			// fallthrough
		} else if err != nil {
			return microerror.Mask(err)
		}

		parameters, err := r.getDeploymentParameters(ctx, key.ClusterID(&azureCluster), strconv.FormatInt(azureCluster.ObjectMeta.Generation, 10), azureCluster.Spec.NetworkSpec.Vnet.Name, *natGw.ID, azureCluster.Spec.NetworkSpec.Subnets[i])
		if err != nil {
			return microerror.Mask(err)
		}

		desiredDeployment := azureresource.Deployment{
			Properties: &azureresource.DeploymentProperties{
				Mode:       azureresource.Incremental,
				Parameters: key.ToParameters(parameters),
				Template:   armTemplate,
			},
		}

		// We only submit the deployment if it doesn't exist or it exists but it's out of date.
		shouldSubmitDeployment := currentDeployment.IsHTTPStatus(404)
		if !shouldSubmitDeployment {
			shouldSubmitDeployment, err = r.isDeploymentOutOfDate(ctx, azureCluster, currentDeployment)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		if shouldSubmitDeployment {
			r.logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed", "subnet", azureCluster.Spec.NetworkSpec.Subnets[i].Name)
			err = r.createDeployment(ctx, deploymentsClient, key.ClusterID(&azureCluster), deploymentName, desiredDeployment)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)

			return nil
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged", "subnet", azureCluster.Spec.NetworkSpec.Subnets[i].Name)
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state %#q", *currentDeployment.Properties.ProvisioningState))

		if key.IsFailedProvisioningState(*currentDeployment.Properties.ProvisioningState) {
			r.debugger.LogFailedDeployment(ctx, currentDeployment, err)
			r.logger.LogCtx(ctx, "level", "debug", "message", "removing failed deployment")
			_, err = deploymentsClient.Delete(ctx, key.ClusterID(&azureCluster), deploymentName)
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		}

		if key.IsSucceededProvisioningState(*currentDeployment.Properties.ProvisioningState) {
			subnetID, err := getSubnetIDFromDeploymentOutput(ctx, currentDeployment)
			if err != nil {
				return microerror.Mask(err)
			}

			azureCluster.Spec.NetworkSpec.Subnets[i].ID = subnetID

			storageAccount, err := storageAccountsClient.GetProperties(ctx, key.ClusterID(&azureCluster), key.StorageAccountName(&azureCluster), "")
			if err != nil {
				return microerror.Mask(err)
			}

			if !isSubnetAllowedToStorageAccount(ctx, storageAccount, subnetID) {
				r.logger.LogCtx(ctx, "level", "debug", "message", "Ensuring subnet is allowed into storage account")

				err = addSubnetToStoreAccountAllowedSubnets(ctx, storageAccountsClient, storageAccount, key.ClusterID(&azureCluster), key.StorageAccountName(&azureCluster), subnetID)
				if err != nil {
					return microerror.Mask(err)
				}

				r.logger.LogCtx(ctx, "level", "debug", "message", "Ensured subnet is allowed into storage account")
			}
		}
	}

	// Update AzureCluster so that subnet.ID is saved.
	err = r.ctrlClient.Update(ctx, &azureCluster)
	if apierrors.IsConflict(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "conflict trying to save object in k8s API concurrently", "stack", microerror.JSON(microerror.Mask(err)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// garbageCollectSubnets removes subnets that have an ARM deployment in Azure but are not defined in `AzureCluster`.
// This is required because when removing a node pool, we remove the subnet from `AzureCluster`, so we can remove it here from Azure.
func (r *Resource) garbageCollectSubnets(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, subnetsClient *network.SubnetsClient, azureCluster capzv1alpha3.AzureCluster) error {
	subnetsIterator, err := subnetsClient.ListComplete(ctx, key.ClusterID(&azureCluster), azureCluster.Spec.NetworkSpec.Vnet.Name)
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Vnet %#q not found, cancelling resource", azureCluster.Spec.NetworkSpec.Vnet.Name))
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	for subnetsIterator.NotDone() {
		subnetInAzure := subnetsIterator.Value()

		if !isSubnetInAzureClusterSpec(ctx, azureCluster, *subnetInAzure.Name) && !isProtectedSubnet(*subnetInAzure.Name) {
			err = r.deleteSubnet(ctx, subnetsClient, key.ClusterID(&azureCluster), azureCluster.Spec.NetworkSpec.Vnet.Name, *subnetInAzure.Name)
			if IsSubnetInUse(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("subnet %q in Azure still in use by VMSS", *subnetInAzure.Name))

				err = subnetsIterator.NextWithContext(ctx)
				if err != nil {
					return microerror.Mask(err)
				}

				continue
			} else if err != nil {
				return microerror.Mask(err)
			}

			err = r.deleteARMDeployment(ctx, deploymentsClient, key.ClusterID(&azureCluster), key.SubnetDeploymentName(*subnetInAzure.Name))
			if err != nil {
				return microerror.Mask(err)
			}
		}

		err = subnetsIterator.NextWithContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) deleteARMDeployment(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, resourceGroupName, deploymentName string) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleting subnet %#q ARM deployment", deploymentName))

	_, err := deploymentsClient.Delete(ctx, resourceGroupName, deploymentName)
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "Subnet ARM deployment was already deleted")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleted subnet %#q ARM deployment", deploymentName))

	return nil
}

func (r *Resource) deleteSubnet(ctx context.Context, subnetsClient *network.SubnetsClient, resourceGroupName, virtualNetworkName, subnetName string) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting subnet %q", subnetName))

	_, err := subnetsClient.Delete(ctx, resourceGroupName, virtualNetworkName, subnetName)
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "Subnet was already deleted")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted subnet %q", subnetName))

	return nil
}

func isProtectedSubnet(subnetName string) bool {
	return strings.HasSuffix(subnetName, "-MasterSubnet") || strings.HasSuffix(subnetName, "-WorkerSubnet") || strings.HasSuffix(subnetName, key.VNetGatewaySubnetName())
}

func isSubnetInAzureClusterSpec(ctx context.Context, azureCluster capzv1alpha3.AzureCluster, subnetName string) bool {
	for _, subnetInSpec := range azureCluster.Spec.NetworkSpec.Subnets {
		if subnetInSpec.Name == subnetName {
			return true
		}
	}

	return false
}

func getSubnetIDFromDeploymentOutput(ctx context.Context, currentDeployment azureresource.DeploymentExtended) (string, error) {
	outputs, ok := currentDeployment.Properties.Outputs.(map[string]interface{})
	if !ok {
		return "", microerror.Maskf(wrongTypeError, "expected 'map[string]interface{}', got '%T'", currentDeployment.Properties.Outputs)
	}

	subnetID, ok := outputs["subnetID"].(map[string]interface{})
	if !ok {
		return "", microerror.Maskf(wrongTypeError, "expected 'map[string]interface{}', got '%T'", outputs["subnetID"])
	}

	return subnetID["value"].(string), nil
}

func addSubnetToStoreAccountAllowedSubnets(ctx context.Context, storageAccountsClient *storage.AccountsClient, storageAccount storage.Account, resourceGroupName, StorageAccountName, subnetID string) error {
	*storageAccount.AccountProperties.NetworkRuleSet.VirtualNetworkRules = append(*storageAccount.AccountProperties.NetworkRuleSet.VirtualNetworkRules, storage.VirtualNetworkRule{VirtualNetworkResourceID: to.StringPtr(subnetID)})
	_, err := storageAccountsClient.Update(ctx, resourceGroupName, StorageAccountName, storage.AccountUpdateParameters{
		AccountPropertiesUpdateParameters: &storage.AccountPropertiesUpdateParameters{
			CustomDomain:                          storageAccount.AccountProperties.CustomDomain,
			Encryption:                            storageAccount.AccountProperties.Encryption,
			AccessTier:                            storageAccount.AccountProperties.AccessTier,
			AzureFilesIdentityBasedAuthentication: storageAccount.AccountProperties.AzureFilesIdentityBasedAuthentication,
			EnableHTTPSTrafficOnly:                storageAccount.AccountProperties.EnableHTTPSTrafficOnly,
			NetworkRuleSet:                        storageAccount.AccountProperties.NetworkRuleSet,
			LargeFileSharesState:                  storageAccount.AccountProperties.LargeFileSharesState,
		},
	})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func isSubnetAllowedToStorageAccount(ctx context.Context, storageAccount storage.Account, subnetID string) bool {
	for _, networkRule := range *storageAccount.AccountProperties.NetworkRuleSet.VirtualNetworkRules {
		if *networkRule.VirtualNetworkResourceID == subnetID {
			return true
		}
	}

	return false
}

// This functions decides whether or not the ARM deployment is out of date.
// For that, we use the Generation field from the AzureCluster CR. This Generation field should change when there is a change in the CR.
func (r *Resource) isDeploymentOutOfDate(ctx context.Context, cr capzv1alpha3.AzureCluster, currentDeployment azureresource.DeploymentExtended) (bool, error) {
	crVersion := strconv.FormatInt(cr.ObjectMeta.Generation, 10)
	currentParams, ok := currentDeployment.Properties.Parameters.(map[string]interface{})
	if !ok {
		return false, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[string]interface{}{}, currentDeployment.Properties.Parameters)
	}

	deploymentVersion, ok := currentParams["azureClusterVersion"].(map[string]interface{})["value"].(string)
	if !ok {
		return false, microerror.Maskf(wrongTypeError, "expected 'string', got '%T'", currentDeployment.Properties.Parameters)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Checking if deployment is out of date", "azureClusterVersion", crVersion, "deploymentParameter", deploymentVersion)

	return crVersion != deploymentVersion, nil
}

func (r *Resource) getDeploymentParameters(ctx context.Context, clusterID, azureClusterVersion, virtualNetworkName, natGatewayId string, allocatedSubnet *capzv1alpha3.SubnetSpec) (map[string]interface{}, error) {
	// @TODO: nat gateway, route table and security group names should come from CR state instead of convention.
	return map[string]interface{}{
		"azureClusterVersion": azureClusterVersion,
		"natGatewayId":        natGatewayId,
		"nodepoolName":        allocatedSubnet.Name,
		"routeTableName":      fmt.Sprintf("%s-%s", clusterID, "RouteTable"),
		"securityGroupName":   fmt.Sprintf("%s-%s", clusterID, "WorkerSecurityGroup"),
		"subnetCidr":          allocatedSubnet.CidrBlock,
		"virtualNetworkName":  virtualNetworkName,
	}, nil
}

// EnsureDeleted is a noop since the deletion of deployments is redirected to
// the deletion of resource groups because they garbage collect them.
func (r *Resource) createDeployment(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, resourceGroup, deploymentName string, desiredDeployment azureresource.Deployment) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring subnets deployments")

	res, err := deploymentsClient.CreateOrUpdate(ctx, resourceGroup, deploymentName, desiredDeployment)
	if err != nil {
		maskedErr := microerror.Mask(err)
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment failed; deployment: %#v", desiredDeployment), "stack", microerror.JSON(maskedErr))

		return maskedErr
	}
	deploymentExtended, err := deploymentsClient.CreateOrUpdateResponder(res.Response())
	if err != nil {
		maskedErr := microerror.Mask(err)
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment failed; deployment: %#v", deploymentExtended), "stack", microerror.JSON(maskedErr))

		return maskedErr
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured deployment")

	return nil
}

// EnsureDeleted is a noop since the deletion of deployments is redirected to
// the deletion of resource groups because they garbage collect them.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

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
	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	corev1 "k8s.io/api/core/v1"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	subnet "github.com/giantswarm/azure-operator/v4/service/controller/resource/subnet/template"
)

const (
	credentialDefaultName = "credential-default"
	credentialNamespace   = "giantswarm"
	mainDeploymentName    = "subnet"
	// Name is the identifier of the resource.
	Name = "subnet"
)

type Config struct {
	AzureClientsFactory *client.Factory
	CtrlClient          ctrlclient.Client
	Debugger            *debugger.Debugger
	Logger              micrologger.Logger
}

type Resource struct {
	azureClientsFactory *client.Factory
	ctrlClient          ctrlclient.Client
	debugger            *debugger.Debugger
	logger              micrologger.Logger
}

type StorageAccountIpRule struct {
	Value  string `json:"value"`
	Action string `json:"action"`
}

func New(config Config) (*Resource, error) {
	if config.AzureClientsFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureClientsFactory must not be empty", config)
	}
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

	credentialSecret, err := r.getCredentialSecret(ctx, &azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsClient, err := r.azureClientsFactory.GetDeploymentsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	natGatewaysClient, err := r.azureClientsFactory.GetNatGatewaysClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	storageAccountsClient, err := r.azureClientsFactory.GetStorageAccountsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	subnetsClient, err := r.azureClientsFactory.GetSubnetsClient(credentialSecret.Namespace, credentialSecret.Name)
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

func getSubnetARMDeploymentName(subnetName string) string {
	return fmt.Sprintf("%s-%s", mainDeploymentName, subnetName)
}

func (r *Resource) ensureSubnets(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, storageAccountsClient *storage.AccountsClient, natGatewaysClient *network.NatGatewaysClient, azureCluster capzv1alpha3.AzureCluster) error {
	armTemplate, err := subnet.GetARMTemplate()
	if err != nil {
		return microerror.Mask(err)
	}

	natGw, err := natGatewaysClient.Get(ctx, key.ClusterID(&azureCluster), "workers-nat-gw", "")
	if err != nil {
		return microerror.Mask(err)
	}

	if natGw.ProvisioningState != network.Succeeded {
		return microerror.Mask(natGatewayNotReadyError)
	}

	for _, allocatedSubnet := range azureCluster.Spec.NetworkSpec.Subnets {
		deploymentName := getSubnetARMDeploymentName(allocatedSubnet.Name)
		currentDeployment, err := deploymentsClient.Get(ctx, key.ClusterID(&azureCluster), deploymentName)
		if IsNotFound(err) {
			// fallthrough
		} else if err != nil {
			return microerror.Mask(err)
		}

		parameters, err := r.getDeploymentParameters(ctx, key.ClusterID(&azureCluster), strconv.FormatInt(azureCluster.ObjectMeta.Generation, 10), azureCluster.Spec.NetworkSpec.Vnet.Name, *natGw.ID, allocatedSubnet)
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
			r.logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")
			err = r.createDeployment(ctx, deploymentsClient, key.ClusterID(&azureCluster), deploymentName, desiredDeployment)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)

			return nil
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")
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

	return nil
}

// garbageCollectSubnets removes subnets that have an ARM deployment in Azure but are not defined in `AzureCluster`.
// This is required because when removing a node pool, we remove the subnet from `AzureCluster`, so we can remove it here from Azure.
func (r *Resource) garbageCollectSubnets(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, subnetsClient *network.SubnetsClient, azureCluster capzv1alpha3.AzureCluster) error {
	subnetsIterator, err := subnetsClient.ListComplete(ctx, key.ClusterID(&azureCluster), azureCluster.Spec.NetworkSpec.Vnet.Name)
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Vnet not %#q found, cancelling resource", azureCluster.Spec.NetworkSpec.Vnet.Name))
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	for subnetsIterator.NotDone() {
		subnetInAzure := subnetsIterator.Value()

		if !isSubnetInAzureClusterSpec(ctx, azureCluster, *subnetInAzure.Name) && !isProtectedSubnet(*subnetInAzure.Name) {
			err = r.deleteSubnet(ctx, subnetsClient, key.ClusterID(&azureCluster), azureCluster.Spec.NetworkSpec.Vnet.Name, *subnetInAzure.Name)
			if IsSubnetInUse(err) {
				r.logger.LogCtx(ctx, "message", "Subnet still in use by VMSS, cancelling resource")
				return nil
			} else if err != nil {
				return microerror.Mask(err)
			}

			err = r.deleteARMDeployment(ctx, deploymentsClient, key.ClusterID(&azureCluster), getSubnetARMDeploymentName(*subnetInAzure.Name))
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
	r.logger.LogCtx(ctx, "message", "Deleting subnet ARM deployment")

	_, err := deploymentsClient.Delete(ctx, resourceGroupName, deploymentName)
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "message", "Subnet ARM deployment was already deleted")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", "Deleted subnet ARM deployment")

	return nil
}

func (r *Resource) deleteSubnet(ctx context.Context, subnetsClient *network.SubnetsClient, resourceGroupName, virtualNetworkName, subnetName string) error {
	r.logger.LogCtx(ctx, "message", "Deleting Subnet")

	_, err := subnetsClient.Delete(ctx, resourceGroupName, virtualNetworkName, subnetName)
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "message", "Subnet was already deleted")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", "Deleted Subnet")

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

	r.logger.LogCtx(ctx, "message", "Checking if deployment is out of date", "azureClusterVersion", crVersion, "deploymentParameter", currentParams["azureClusterVersion"])

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

func (r *Resource) getCredentialSecret(ctx context.Context, cluster key.LabelsGetter) (*v1alpha1.CredentialSecret, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding credential secret")

	organization, exists := cluster.GetLabels()[label.Organization]
	if !exists {
		return nil, microerror.Mask(missingOrganizationLabel)
	}

	secretList := &corev1.SecretList{}
	{
		err := r.ctrlClient.List(
			ctx,
			secretList,
			ctrlclient.InNamespace(credentialNamespace),
			ctrlclient.MatchingLabels{
				label.App:          "credentiald",
				label.Organization: organization,
			},
		)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// We currently only support one credential secret per organization.
	// If there are more than one, return an error.
	if len(secretList.Items) > 1 {
		return nil, microerror.Mask(tooManyCredentialsError)
	}

	// If one credential secret is found, we use that.
	if len(secretList.Items) == 1 {
		secret := secretList.Items[0]

		credentialSecret := &v1alpha1.CredentialSecret{
			Namespace: secret.Namespace,
			Name:      secret.Name,
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found credential secret %s/%s", credentialSecret.Namespace, credentialSecret.Name))

		return credentialSecret, nil
	}

	// If no credential secrets are found, we use the default.
	credentialSecret := &v1alpha1.CredentialSecret{
		Namespace: credentialNamespace,
		Name:      credentialDefaultName,
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "did not find credential secret, using default secret")

	return credentialSecret, nil
}

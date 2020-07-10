package subnet

import (
	"context"
	"fmt"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
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

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToAzureCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	credentialSecret, err := r.getCredentialSecret(ctx, &cr)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsClient, err := r.azureClientsFactory.GetDeploymentsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	armTemplate, err := subnet.GetARMTemplate()
	if err != nil {
		return microerror.Mask(err)
	}

	for _, allocatedSubnet := range cr.Spec.NetworkSpec.Subnets {
		deploymentName := fmt.Sprintf("%s-%s", mainDeploymentName, allocatedSubnet.Name)
		currentDeployment, err := deploymentsClient.Get(ctx, key.ClusterID(&cr), deploymentName)
		if IsNotFound(err) {
			// fallthrough
		} else if err != nil {
			return microerror.Mask(err)
		}

		parameters, err := r.getDeploymentParameters(ctx, key.ClusterID(&cr), cr.Spec.NetworkSpec.Vnet.Name, allocatedSubnet)
		if err != nil {
			return microerror.Mask(err)
		}

		desiredDeployment := azureresource.Deployment{
			Properties: &azureresource.DeploymentProperties{
				Mode:       azureresource.Incremental,
				Parameters: parameters,
				Template:   armTemplate,
			},
		}

		if currentDeployment.IsHTTPStatus(404) || isDeploymentOutOfDate(cr, currentDeployment) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")
			err = r.createDeployment(ctx, deploymentsClient, key.ClusterID(&cr), deploymentName, desiredDeployment)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)

			return nil
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", *currentDeployment.Properties.ProvisioningState))

		if key.IsFailedProvisioningState(*currentDeployment.Properties.ProvisioningState) {
			r.debugger.LogFailedDeployment(ctx, currentDeployment, err)
			r.logger.LogCtx(ctx, "level", "debug", "message", "removing failed deployment")
			_, err = deploymentsClient.Delete(ctx, key.ClusterID(&cr), deploymentName)
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		}
	}

	return nil
}

func isDeploymentOutOfDate(cr capzv1alpha3.AzureCluster, currentDeployment azureresource.DeploymentExtended) bool {
	currentParams := currentDeployment.Properties.Parameters.(map[string]interface{})
	return cr.ResourceVersion != currentParams["AzureClusterVersion"].(string)
}

func (r *Resource) getDeploymentParameters(ctx context.Context, clusterID, virtualNetworkName string, allocatedSubnet *capzv1alpha3.SubnetSpec) (map[string]interface{}, error) {
	return map[string]interface{}{
		"securityGroupName":  fmt.Sprintf("%s-%s", clusterID, "WorkerSecurityGroup"),
		"subnetCidr":         allocatedSubnet.CidrBlock,
		"nodepoolName":       allocatedSubnet.Name,
		"virtualNetworkName": virtualNetworkName,
	}, nil
}

// EnsureDeleted is a noop since the deletion of deployments is redirected to
// the deletion of resource groups because they garbage collect them.
func (r *Resource) createDeployment(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, resourceGroup, deploymentName string, desiredDeployment azureresource.Deployment) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring subnets deployments")

	res, err := deploymentsClient.CreateOrUpdate(ctx, resourceGroup, deploymentName, desiredDeployment)
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment failed; deployment: %#v", desiredDeployment), "stack", microerror.JSON(microerror.Mask(err)))
		return microerror.Mask(err)
	}
	deploymentExtended, err := deploymentsClient.CreateOrUpdateResponder(res.Response())
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment failed; deployment: %#v", deploymentExtended), "stack", microerror.JSON(microerror.Mask(err)))

		return microerror.Mask(err)
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

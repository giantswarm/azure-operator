package deployment

import (
	"context"
	"fmt"
	"time"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v5/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/reconciliationcanceledcontext"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/client"
	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v5/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/setting"
)

const (
	// Name is the identifier of the resource.
	Name = "deployment"
)

const (
	DeploymentTemplateChecksum   = "TemplateChecksum"
	DeploymentParametersChecksum = "ParametersChecksum"
	mainDeploymentName           = "cluster-main-template"
)

type Config struct {
	Debugger         *debugger.Debugger
	InstallationName string
	CtrlClient       ctrlClient.Client
	Logger           micrologger.Logger

	Azure                      setting.Azure
	AzureClientSet             *client.AzureClientSet
	ClientFactory              client.OrganizationFactory
	ControlPlaneSubscriptionID string
	Debug                      setting.Debug
}

type Resource struct {
	debugger         *debugger.Debugger
	installationName string
	ctrlClient       ctrlClient.Client
	logger           micrologger.Logger

	azure                      setting.Azure
	azureClientSet             *client.AzureClientSet
	clientFactory              client.OrganizationFactory
	controlPlaneSubscriptionID string
	debug                      setting.Debug
}

type StorageAccountIpRule struct {
	Value  string `json:"value"`
	Action string `json:"action"`
}

func New(config Config) (*Resource, error) {
	if config.AzureClientSet == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureClientSet must not be empty", config)
	}
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
	}
	if config.InstallationName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.InstallationName must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if config.ControlPlaneSubscriptionID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ControlPlaneSubscriptionID must not be empty", config)
	}

	r := &Resource{
		debugger:         config.Debugger,
		installationName: config.InstallationName,
		ctrlClient:       config.CtrlClient,
		logger:           config.Logger,

		azure:                      config.Azure,
		azureClientSet:             config.AzureClientSet,
		clientFactory:              config.ClientFactory,
		controlPlaneSubscriptionID: config.ControlPlaneSubscriptionID,
		debug:                      config.Debug,
	}

	return r, nil
}

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Pre-condition check: VNet CIDR must be set.
	if cr.Spec.Azure.VirtualNetwork.CIDR == "" {
		r.logger.Debugf(ctx, "vnet cidr not set")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	deploymentsClient, err := r.clientFactory.GetDeploymentsClient(ctx, cr.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensuring deployment")

	var deployment azureresource.Deployment

	failed := false

	d, err := deploymentsClient.Get(ctx, key.ClusterID(&cr), mainDeploymentName)
	if IsNotFound(err) {
		params := map[string]interface{}{
			"initialProvisioning": "Yes",
		}
		deployment, err = r.newDeployment(ctx, cr, params)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		s := *d.Properties.ProvisioningState

		r.logger.Debugf(ctx, "deployment is in state '%s'", s)

		if !key.IsSucceededProvisioningState(s) {
			r.debugger.LogFailedDeployment(ctx, d, err)
			failed = true
		}
		if !key.IsFinalProvisioningState(s) {
			reconciliationcanceledcontext.SetCanceled(ctx)
			r.logger.Debugf(ctx, "canceling reconciliation")
			return nil
		}

		params := map[string]interface{}{
			"initialProvisioning": "No",
		}
		deployment, err = r.newDeployment(ctx, cr, params)
		if err != nil {
			return microerror.Mask(err)
		}

		err = r.ensureServiceEndpoints(ctx, cr)
		if err != nil {
			return microerror.Mask(err)
		}

		if reconciliationcanceledcontext.IsCanceled(ctx) {
			return nil
		}

		err = r.enrichControllerContext(ctx, cr, deploymentsClient)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	desiredDeploymentTemplateChk, err := getDeploymentTemplateChecksum(deployment)
	if err != nil {
		return microerror.Mask(err)
	}

	desiredDeploymentParametersChk, err := getDeploymentParametersChecksum(deployment)
	if err != nil {
		return microerror.Mask(err)
	}

	if failed {
		r.logger.Debugf(ctx, "the main deployment is failed")
	} else {
		currentDeploymentTemplateChk, err := r.getResourceStatus(ctx, cr, DeploymentTemplateChecksum)
		if err != nil {
			return microerror.Mask(err)
		}

		currentDeploymentParametersChk, err := r.getResourceStatus(ctx, cr, DeploymentParametersChecksum)
		if err != nil {
			return microerror.Mask(err)
		}

		if currentDeploymentTemplateChk == desiredDeploymentTemplateChk && currentDeploymentParametersChk == desiredDeploymentParametersChk {
			r.logger.Debugf(ctx, "template and parameters unchanged")

			// Deployment is now stable, ensure the NAT gateway is enabled for the master subnet.
			err := r.ensureNatGatewayForMasterSubnet(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		}

		// As current and desired state differs, start process from the beginning.
		r.logger.Debugf(ctx, "template or parameters changed")
	}

	res, err := deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(&cr), mainDeploymentName, deployment)
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment failed; deployment: %#v", deployment), "stack", microerror.JSON(microerror.Mask(err)))

		return microerror.Mask(err)
	}

	deploymentExtended, err := deploymentsClient.CreateOrUpdateResponder(res.Response())
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment failed; deployment: %#v", deploymentExtended), "stack", microerror.JSON(microerror.Mask(err)))

		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensured deployment")

	if desiredDeploymentTemplateChk != "" {
		err = r.setResourceStatus(ctx, cr, DeploymentTemplateChecksum, desiredDeploymentTemplateChk)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "set %s to '%s'", DeploymentTemplateChecksum, desiredDeploymentTemplateChk)
	} else {
		r.logger.Debugf(ctx, "Unable to get a valid Checksum for %s", DeploymentTemplateChecksum)
	}

	if desiredDeploymentParametersChk != "" {
		err = r.setResourceStatus(ctx, cr, DeploymentParametersChecksum, desiredDeploymentParametersChk)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "set %s to '%s'", DeploymentParametersChecksum, desiredDeploymentParametersChk)
	} else {
		r.logger.Debugf(ctx, "Unable to get a valid Checksum for %s", DeploymentParametersChecksum)
	}

	r.logger.Debugf(ctx, "canceling reconciliation")
	reconciliationcanceledcontext.SetCanceled(ctx)

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

func (r *Resource) enrichControllerContext(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentsClient *azureresource.DeploymentsClient) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	resourceGroupName := key.ClusterID(&customObject)
	{
		v, err := r.getDeploymentOutputValue(ctx, deploymentsClient, resourceGroupName, "master_load_balancer_setup", "backendPoolId")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.MasterLBBackendPoolID = v
		}
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, deploymentsClient, resourceGroupName, "virtual_network_setup", "masterSubnetID")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.MasterSubnetID = v
		}
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, deploymentsClient, resourceGroupName, "virtual_network_setup", "workerSubnetID")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.WorkerSubnetID = v
		}
	}

	return nil
}

func (r *Resource) getDeploymentOutputValue(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, resourceGroupName, deploymentName, outputName string) (string, error) {
	d, err := deploymentsClient.Get(ctx, resourceGroupName, deploymentName)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if d.Properties.Outputs == nil {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("cannot get output value '%s' of deployment '%s'", outputName, deploymentName))
		r.logger.LogCtx(ctx, "level", "warning", "message", "assuming deployment is in failed state")
		r.logger.LogCtx(ctx, "level", "warning", "message", "canceling controller context enrichment")
		return "", nil
	}

	m, err := key.ToMap(d.Properties.Outputs)
	if err != nil {
		return "", microerror.Mask(err)
	}
	v, ok := m[outputName]
	if !ok {
		return "", microerror.Maskf(missingOutputValueError, outputName)
	}
	m, err = key.ToMap(v)
	if err != nil {
		return "", microerror.Mask(err)
	}
	v, err = key.ToKeyValue(m)
	if err != nil {
		return "", microerror.Mask(err)
	}
	s, err := key.ToString(v)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return s, nil
}

func (r *Resource) getResourceStatus(ctx context.Context, customObject providerv1alpha1.AzureConfig, t string) (string, error) {
	{
		objectKey := ctrlClient.ObjectKey{
			Namespace: customObject.Namespace,
			Name:      customObject.Name,
		}
		c := &providerv1alpha1.AzureConfig{}
		err := r.ctrlClient.Get(ctx, objectKey, c)
		if err != nil {
			return "", microerror.Mask(err)
		}

		customObject = *c
	}

	for _, r := range customObject.Status.Cluster.Resources {
		if r.Name != Name {
			continue
		}

		for _, c := range r.Conditions {
			if c.Type == t {
				return c.Status, nil
			}
		}
	}

	return "", nil
}

func (r *Resource) setResourceStatus(ctx context.Context, customObject providerv1alpha1.AzureConfig, t string, s string) error {
	// Get the newest CR version. Otherwise status update may fail because of:
	//
	//	 the object has been modified; please apply your changes to the
	//	 latest version and try again
	//
	{
		objectKey := ctrlClient.ObjectKey{
			Namespace: customObject.Namespace,
			Name:      customObject.Name,
		}
		c := &providerv1alpha1.AzureConfig{}
		err := r.ctrlClient.Get(ctx, objectKey, c)
		if err != nil {
			return microerror.Mask(err)
		}

		customObject = *c
	}

	resourceStatus := providerv1alpha1.StatusClusterResource{
		Conditions: []providerv1alpha1.StatusClusterResourceCondition{
			{
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Status:             s,
				Type:               t,
			},
		},
		Name: Name,
	}

	var set bool
	for i, r := range customObject.Status.Cluster.Resources {
		if r.Name != Name {
			continue
		}

		for _, c := range r.Conditions {
			if c.Type == t {
				continue
			}
			resourceStatus.Conditions = append(resourceStatus.Conditions, c)
		}

		customObject.Status.Cluster.Resources[i] = resourceStatus
		set = true
	}

	if !set {
		customObject.Status.Cluster.Resources = append(customObject.Status.Cluster.Resources, resourceStatus)
	}

	{
		err := r.ctrlClient.Update(ctx, &customObject)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

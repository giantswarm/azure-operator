package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	gocache "github.com/patrickmn/go-cache"

	"github.com/giantswarm/azure-operator/v4/pkg/credential"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	cacheHitLogKey       = "cacheHit"
	clientTypeLogKey     = "clientType"
	clusterIDLogKey      = "clusterID"
	credentialNameLogKey = "credentialName"
	logLevelLogKey       = "level"
	logLevelDebug        = "debug"
	messageLogKey        = "message"
)

type FactoryConfig struct {
	CacheDuration time.Duration
	GSTenantID    string
	K8sClient     k8sclient.Interface
	Logger        micrologger.Logger
}

// Factory is creating Azure clients for specified AzureConfig CRs, so basically for specified
// tenant clusters. All created clients are cached.
type Factory struct {
	gsTenantID string
	k8sClient  k8sclient.Interface
	logger     micrologger.Logger
	mutex      sync.Mutex

	// map [credentialName + client type] -> client
	cachedClients *gocache.Cache
}

type clientCreatorFunc func(autorest.Authorizer, string, string) (interface{}, error)

// NewFactory returns a new Azure client factory that is used throughout entire azure-operator
// lifetime.
func NewFactory(config FactoryConfig) (*Factory, error) {
	if config.CacheDuration < 5*time.Minute { // cache at least for one reconciliation loop duration
		return nil, microerror.Maskf(invalidConfigError, "%T.CacheDuration must be at least 5 minutes", config)
	}
	if len(config.GSTenantID) == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.GSTenantID must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	factory := &Factory{
		k8sClient:     config.K8sClient,
		logger:        config.Logger,
		gsTenantID:    config.GSTenantID,
		cachedClients: gocache.New(config.CacheDuration, 2*config.CacheDuration),
	}

	factory.cachedClients.OnEvicted(func(clientKey string, i interface{}) {
		factory.onEvicted(clientKey)
	})

	return factory, nil
}

// GetDeploymentsClient returns DeploymentsClient that is used for management of deployments and
// ARM templates. The client (for specified cluster) is cached after creation, so the same client
// is returned every time.
func (f *Factory) GetDeploymentsClient(cr v1alpha1.AzureConfig) (*resources.DeploymentsClient, error) {
	client, err := f.getClient(cr, "DeploymentsClient", newDeploymentsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toDeploymentsClient(client), nil
}

// GetGroupsClient returns GroupsClient that is used for management of resource groups for the
// specified cluster. The created client is cached for the time period specified in the factory
// config.
func (f *Factory) GetGroupsClient(cr v1alpha1.AzureConfig) (*resources.GroupsClient, error) {
	client, err := f.getClient(cr, "GroupsClient", newGroupsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toGroupsClient(client), nil
}

// GetVirtualMachineScaleSetsClient returns VirtualMachineScaleSetsClient that is used for
// management of virtual machine scale sets for the specified cluster. The created client is cached
// for the time period specified in the factory config.
func (f *Factory) GetVirtualMachineScaleSetsClient(cr v1alpha1.AzureConfig) (*compute.VirtualMachineScaleSetsClient, error) {
	client, err := f.getClient(cr, "VirtualMachineScaleSetsClient", newVirtualMachineScaleSetsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toVirtualMachineScaleSetsClient(client), nil
}

// GetVirtualMachineScaleSetVMsClient returns GetVirtualMachineScaleSetVMsClient that is used for
// management of virtual machine scale set instances for the specified cluster. The created client
// is cached for the time period specified in the factory config.
func (f *Factory) GetVirtualMachineScaleSetVMsClient(cr v1alpha1.AzureConfig) (*compute.VirtualMachineScaleSetVMsClient, error) {
	client, err := f.getClient(cr, "VirtualMachineScaleSetVMsClient", newVirtualMachineScaleSetVMsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toVirtualMachineScaleSetVMsClient(client), nil
}

// GetStorageAccountsClient returns *storage.AccountsClient that is used for management of Azure
// storage accounts for the specified cluster. The created client is cached for the time period
// specified in the factory config.
func (f *Factory) GetStorageAccountsClient(cr v1alpha1.AzureConfig) (*storage.AccountsClient, error) {
	client, err := f.getClient(cr, "StorageAccountsClient", newStorageAccountsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toStorageAccountsClient(client), nil
}

func (f *Factory) getClient(cr v1alpha1.AzureConfig, clientType string, createClient clientCreatorFunc) (interface{}, error) {
	l := f.logger.With(
		logLevelLogKey, logLevelDebug,
		messageLogKey, "get client",
		credentialNameLogKey, key.CredentialName(cr),
		clusterIDLogKey, key.ClusterID(&cr),
		clientTypeLogKey, clientType)

	clientKey := getClientKey(cr, clientType)
	var client interface{}
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if cachedClient, ok := f.cachedClients.Get(clientKey); ok {
		// client found, it will be refreshed in cache
		l.Log(cacheHitLogKey, true)
		client = cachedClient
	} else {
		// client not found, create it, it will be saved in cache
		l.Log(cacheHitLogKey, false)
		newClient, err := f.createClient(cr, createClient)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		client = newClient
	}

	// refresh existing client or set new one
	f.cachedClients.SetDefault(clientKey, client)
	return client, nil
}

func (f *Factory) createClient(cr v1alpha1.AzureConfig, createClient clientCreatorFunc) (interface{}, error) {
	organizationCredentialsConfig, subscriptionID, partnerID, err := credential.GetOrganizationAzureCredentials(context.Background(), f.k8sClient, cr, f.gsTenantID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	authorizer, err := organizationCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	client, err := createClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func getClientKey(cr v1alpha1.AzureConfig, clientType string) string {
	return fmt.Sprintf("%s.%s", key.CredentialName(cr), clientType)
}

func getClientKeyParts(clientKey string) (credentialName, clientType string) {
	parts := strings.Split(clientKey, ".")
	partsCount := len(parts)

	if partsCount > 2 {
		credentialName = strings.Join(parts[0:partsCount-1], ".")
		clientType = parts[partsCount-1]
	} else if partsCount == 2 {
		credentialName, clientType = parts[0], parts[1]
	} else {
		// this should never happen, don't return error, this is for logging only
		credentialName, clientType = "unknown", "unknown"
	}

	return
}

func (f *Factory) onEvicted(clientKey string) {
	credentialName, clientType := getClientKeyParts(clientKey)
	f.logger.Log(
		logLevelLogKey, logLevelDebug,
		messageLogKey, "client evicted",
		credentialNameLogKey, credentialName,
		clientTypeLogKey, clientType)
}

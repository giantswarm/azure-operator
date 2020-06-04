package client

import (
	"sync"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v4/pkg/credential"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	cacheHitLogKey          = "cacheHit"
	clientFactoryFuncLogKey = "clientFactoryFunc"
	clientSetKeyLogKey      = "clientSetKey"
)

type FactoryConfig struct {
	GSTenantID string
	K8sClient  k8sclient.Interface
	Logger     micrologger.Logger
}

// Factory is creating Azure clients for specified AzureConfig CRs, so basically for specified
// tenant clusters. All created clients are cached.
type Factory struct {
	gsTenantID string
	k8sClient  k8sclient.Interface
	logger     micrologger.Logger
	mutex      sync.Mutex

	clients map[credentialID]*AzureClientSet
}

type credentialID string
type clientCreatorFunc func(autorest.Authorizer, string, string) (interface{}, error)

// NewFactory returns a new Azure client factory that is used throughout entire azure-operator
// lifetime.
func NewFactory(config FactoryConfig) (*Factory, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if len(config.GSTenantID) == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.GSTenantID must not be empty", config)
	}

	factory := &Factory{
		k8sClient:  config.K8sClient,
		logger:     config.Logger,
		gsTenantID: config.GSTenantID,
		clients:    make(map[credentialID]*AzureClientSet),
	}

	return factory, nil
}

// GetDeploymentsClient returns DeploymentsClient that is used for management of deployments and
// ARM templates. The client (for specified cluster) is cached after creation, so the same client
// is returned every time.
func (f *Factory) GetDeploymentsClient(cr v1alpha1.AzureConfig) (*resources.DeploymentsClient, error) {
	clientSetKey := getClientSetKey(cr)
	logger := f.logger.With(clientFactoryFuncLogKey, "GetDeploymentsClient", clientSetKeyLogKey, clientSetKey)

	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.ensureClientSetExists(clientSetKey, logger)

	if f.clients[clientSetKey].DeploymentsClient == nil {
		deploymentClient, err := f.createClient(cr, func(authorizer autorest.Authorizer, subscriptionID string, partnerID string) (interface{}, error) {
			return newDeploymentsClient(authorizer, subscriptionID, partnerID)
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
		f.clients[clientSetKey].DeploymentsClient = deploymentClient.(*resources.DeploymentsClient)
		logger.Log(cacheHitLogKey, false)
	} else {
		logger.Log(cacheHitLogKey, true)
	}

	return f.clients[clientSetKey].DeploymentsClient, nil
}

// GetGroupsClient returns GroupsClient that is used for management of resource groups. The client
// (for specified cluster) is cached after creation, so the same client is returned every time.
func (f *Factory) GetGroupsClient(cr v1alpha1.AzureConfig) (*resources.GroupsClient, error) {
	clientSetKey := getClientSetKey(cr)
	logger := f.logger.With(clientFactoryFuncLogKey, "GetGroupsClient", clientSetKeyLogKey, clientSetKey)

	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.ensureClientSetExists(clientSetKey, logger)

	if f.clients[clientSetKey].GroupsClient == nil {
		groupClient, err := f.createClient(cr, func(authorizer autorest.Authorizer, subscriptionID string, partnerID string) (interface{}, error) {
			return newGroupsClient(authorizer, subscriptionID, partnerID)
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
		f.clients[clientSetKey].GroupsClient = groupClient.(*resources.GroupsClient)
		logger.Log(cacheHitLogKey, false)
	} else {
		logger.Log(cacheHitLogKey, true)
	}

	return f.clients[clientSetKey].GroupsClient, nil
}

// GetVirtualMachineScaleSetsClient returns VirtualMachineScaleSetsClient that is used for
// management of virtual machine scale sets. The client (for specified cluster) is cached after
// creation, so the same client is returned every time.
func (f *Factory) GetVirtualMachineScaleSetsClient(cr v1alpha1.AzureConfig) (*compute.VirtualMachineScaleSetsClient, error) {
	clientSetKey := getClientSetKey(cr)
	logger := f.logger.With(clientFactoryFuncLogKey, "GetVirtualMachineScaleSetsClient", clientSetKeyLogKey, clientSetKey)

	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.ensureClientSetExists(clientSetKey, logger)

	if f.clients[clientSetKey].VirtualMachineScaleSetsClient == nil {
		vmssClient, err := f.createClient(cr, func(authorizer autorest.Authorizer, subscriptionID string, partnerID string) (interface{}, error) {
			return newVirtualMachineScaleSetsClient(authorizer, subscriptionID, partnerID)
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
		f.clients[clientSetKey].VirtualMachineScaleSetsClient = vmssClient.(*compute.VirtualMachineScaleSetsClient)
		logger.Log(cacheHitLogKey, false)
	} else {
		logger.Log(cacheHitLogKey, true)
	}

	return f.clients[clientSetKey].VirtualMachineScaleSetsClient, nil
}

// GetVirtualMachineScaleSetVMsClient returns GetVirtualMachineScaleSetVMsClient that is used for
// management of virtual machine scale set instances. The client (for specified cluster) is cached
// after creation, so the same client is returned every time.
func (f *Factory) GetVirtualMachineScaleSetVMsClient(cr v1alpha1.AzureConfig) (*compute.VirtualMachineScaleSetVMsClient, error) {
	clientSetKey := getClientSetKey(cr)
	logger := f.logger.With(clientFactoryFuncLogKey, "GetVirtualMachineScaleSetVMsClient", clientSetKeyLogKey, clientSetKey)

	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.ensureClientSetExists(clientSetKey, logger)

	if f.clients[clientSetKey].VirtualMachineScaleSetVMsClient == nil {
		vmssVMsClient, err := f.createClient(cr, func(authorizer autorest.Authorizer, subscriptionID string, partnerID string) (interface{}, error) {
			return newVirtualMachineScaleSetVMsClient(authorizer, subscriptionID, partnerID)
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
		f.clients[clientSetKey].VirtualMachineScaleSetVMsClient = vmssVMsClient.(*compute.VirtualMachineScaleSetVMsClient)
		logger.Log(cacheHitLogKey, false)
	} else {
		logger.Log(cacheHitLogKey, true)
	}

	return f.clients[clientSetKey].VirtualMachineScaleSetVMsClient, nil
}

// GetStorageAccountsClient returns *storage.AccountsClient that is used for management of Azure
// storage accounts. The client (for specified cluster) is cached after creation, so the same
// client is returned every time.
func (f *Factory) GetStorageAccountsClient(cr v1alpha1.AzureConfig) (*storage.AccountsClient, error) {
	clientSetKey := getClientSetKey(cr)
	logger := f.logger.With(clientFactoryFuncLogKey, "GetStorageAccountsClient", clientSetKeyLogKey, clientSetKey)

	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.ensureClientSetExists(clientSetKey, logger)

	if f.clients[clientSetKey].StorageAccountsClient == nil {
		storageAccountsClient, err := f.createClient(cr, func(authorizer autorest.Authorizer, subscriptionID string, partnerID string) (interface{}, error) {
			return newStorageAccountsClient(authorizer, subscriptionID, partnerID)
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
		f.clients[clientSetKey].StorageAccountsClient = storageAccountsClient.(*storage.AccountsClient)
		logger.Log(cacheHitLogKey, false)
	} else {
		logger.Log(cacheHitLogKey, true)
	}

	return f.clients[clientSetKey].StorageAccountsClient, nil
}

// RemoveAllClients removes all cached clients for the specified tenant cluster.
func (f *Factory) RemoveAllClients(cr v1alpha1.AzureConfig) {
	clientSetKey := getClientSetKey(cr)
	logger := f.logger.With(clientFactoryFuncLogKey, "RemoveAllClients", clientSetKeyLogKey, clientSetKey)

	f.mutex.Lock()
	defer f.mutex.Unlock()
	if _, ok := f.clients[clientSetKey]; !ok {
		logger.Log(cacheHitLogKey, false)
		return
	}

	logger.Log(cacheHitLogKey, true)
	delete(f.clients, clientSetKey)
}

func (f *Factory) createClient(cr v1alpha1.AzureConfig, createClient clientCreatorFunc) (interface{}, error) {
	organizationCredentialsConfig, subscriptionID, partnerID, err := credential.GetOrganizationAzureCredentials(f.k8sClient, cr, f.gsTenantID)
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

func (f *Factory) ensureClientSetExists(clientSetKey credentialID, logger micrologger.Logger) {
	if _, ok := f.clients[clientSetKey]; ok {
		logger.Log("azureClientSetCacheHit", true)
	} else {
		logger.Log("azureClientSetCacheHit", false)
		f.clients[clientSetKey] = &AzureClientSet{}
	}
}

func getClientSetKey(cr v1alpha1.AzureConfig) credentialID {
	return credentialID(key.CredentialName(cr))
}

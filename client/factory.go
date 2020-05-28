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

	"github.com/giantswarm/azure-operator/v4/pkg/credential"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

type FactoryConfig struct {
	K8sClient  k8sclient.Interface
	GSTenantID string
}

type Factory struct {
	mutex      sync.Mutex
	k8sClient  k8sclient.Interface
	gsTenantID string

	clients map[string]*AzureClientSet
}

type clientCreatorFunc func(autorest.Authorizer, string, string) (interface{}, error)

// call NewFactory in EnsureCreated, so it's used in only a single reconciliation loop
func NewFactory(config FactoryConfig) (*Factory, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if len(config.GSTenantID) == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.GSTenantID must not be empty", config)
	}

	factory := &Factory{
		k8sClient:  config.K8sClient,
		gsTenantID: config.GSTenantID,
		clients:    make(map[string]*AzureClientSet),
	}

	return factory, nil
}

// GetDeploymentsClient returns DeploymentsClient that is used for management of deployments and
// ARM templates.
func (f *Factory) GetDeploymentsClient(cr v1alpha1.AzureConfig) (*resources.DeploymentsClient, error) {
	clientSetKey := key.CredentialName(cr)

	f.mutex.Lock()
	if _, ok := f.clients[clientSetKey]; !ok {
		f.clients[clientSetKey] = &AzureClientSet{}
	}
	if f.clients[clientSetKey].DeploymentsClient == nil {
		deploymentClient, err := f.createClient(cr, func(authorizer autorest.Authorizer, subscriptionID string, partnerID string) (interface{}, error) {
			return newDeploymentsClient(authorizer, subscriptionID, partnerID)
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
		f.clients[clientSetKey].DeploymentsClient = deploymentClient.(*resources.DeploymentsClient)
	}
	f.mutex.Unlock()

	return f.clients[clientSetKey].DeploymentsClient, nil
}

// GetGroupsClient returns GroupsClient that is used for management of resource groups.
func (f *Factory) GetGroupsClient(cr v1alpha1.AzureConfig) (*resources.GroupsClient, error) {
	clientSetKey := key.CredentialName(cr)

	f.mutex.Lock()
	if _, ok := f.clients[clientSetKey]; !ok {
		f.clients[clientSetKey] = &AzureClientSet{}
	}
	if f.clients[clientSetKey].GroupsClient == nil {
		groupClient, err := f.createClient(cr, func(authorizer autorest.Authorizer, subscriptionID string, partnerID string) (interface{}, error) {
			return newGroupsClient(authorizer, subscriptionID, partnerID)
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
		f.clients[clientSetKey].GroupsClient = groupClient.(*resources.GroupsClient)
	}
	f.mutex.Unlock()

	return f.clients[clientSetKey].GroupsClient, nil
}

// GetVirtualMachineScaleSetsClient returns VirtualMachineScaleSetsClient that is used for
// management of virtual machine scale sets.
func (f *Factory) GetVirtualMachineScaleSetsClient(cr v1alpha1.AzureConfig) (*compute.VirtualMachineScaleSetsClient, error) {
	clientSetKey := key.CredentialName(cr)

	f.mutex.Lock()
	if _, ok := f.clients[clientSetKey]; !ok {
		f.clients[clientSetKey] = &AzureClientSet{}
	}
	if f.clients[clientSetKey].VirtualMachineScaleSetsClient == nil {
		vmssClient, err := f.createClient(cr, func(authorizer autorest.Authorizer, subscriptionID string, partnerID string) (interface{}, error) {
			return newVirtualMachineScaleSetsClient(authorizer, subscriptionID, partnerID)
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
		f.clients[clientSetKey].VirtualMachineScaleSetsClient = vmssClient.(*compute.VirtualMachineScaleSetsClient)
	}
	f.mutex.Unlock()

	return f.clients[clientSetKey].VirtualMachineScaleSetsClient, nil
}

// GetVirtualMachineScaleSetVMsClient returns GetVirtualMachineScaleSetVMsClient that is used for
// management of virtual machine scale set instances.
func (f *Factory) GetVirtualMachineScaleSetVMsClient(cr v1alpha1.AzureConfig) (*compute.VirtualMachineScaleSetVMsClient, error) {
	clientSetKey := key.CredentialName(cr)

	f.mutex.Lock()
	if _, ok := f.clients[clientSetKey]; !ok {
		f.clients[clientSetKey] = &AzureClientSet{}
	}
	if f.clients[clientSetKey].VirtualMachineScaleSetVMsClient == nil {
		vmssClient, err := f.createClient(cr, func(authorizer autorest.Authorizer, subscriptionID string, partnerID string) (interface{}, error) {
			return newVirtualMachineScaleSetVMsClient(authorizer, subscriptionID, partnerID)
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
		f.clients[clientSetKey].VirtualMachineScaleSetVMsClient = vmssClient.(*compute.VirtualMachineScaleSetVMsClient)
	}
	f.mutex.Unlock()

	return f.clients[clientSetKey].VirtualMachineScaleSetVMsClient, nil
}

// GetStorageAccountsClient returns *storage.AccountsClient that is used for management of Azure
// storage accounts. The client (for specified cluster) is cached after creation, so the same
// client is returned every time.
func (f *Factory) GetStorageAccountsClient(cr v1alpha1.AzureConfig) (*storage.AccountsClient, error) {
	clientSetKey := key.CredentialName(cr)

	f.mutex.Lock()
	if _, ok := f.clients[clientSetKey]; !ok {
		f.clients[clientSetKey] = &AzureClientSet{}
	}
	if f.clients[clientSetKey].StorageAccountsClient == nil {
		storageAccountsClient, err := f.createClient(cr, func(authorizer autorest.Authorizer, subscriptionID string, partnerID string) (interface{}, error) {
			return newStorageAccountsClient(authorizer, subscriptionID, partnerID)
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
		f.clients[clientSetKey].StorageAccountsClient = storageAccountsClient.(*storage.AccountsClient)
	}
	f.mutex.Unlock()

	return f.clients[clientSetKey].StorageAccountsClient, nil
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

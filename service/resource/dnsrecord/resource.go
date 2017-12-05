package dnsrecord

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/dns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/key"
)

const (
	// Name is the identifier of the resource.
	Name = "resourcegroup"

	clusterIDTag  = "ClusterID"
	customerIDTag = "CustomerID"
	deleteTimeout = 5 * time.Minute
	managedBy     = "azure-operator"
)

// Config is the resource group Resource configuration.
type Config struct {
	// Dependencies.

	AzureConfig client.AzureConfig
	Logger      micrologger.Logger
}

// DefaultConfig provides a default configuration to create a new resource by
// best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		AzureConfig: client.DefaultAzureConfig(),
		Logger:      nil,
	}
}

// Resource manages Azure resource groups.
type Resource struct {
	// Dependencies.

	azureConfig client.AzureConfig
	logger      micrologger.Logger
}

// New creates a new configured resource group resource.
func New(config Config) (*Resource, error) {
	// Dependencies.
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig.%s", err)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	newService := &Resource{
		azureConfig: config.AzureConfig,
		logger: config.Logger.With(
			"resource", Name,
		),
	}

	return newService, nil
}
func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	o, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r.getCurrentState(ctx, o)
}

func (r *Resource) getCurrentState(ctx context.Context, obj azuretpr.CustomObject) (dnsRecords, error) {
	recordSetsClient, err := r.getDNSRecordSetsClient()
	if err != nil {
		return nil, microerror.Maskf(err, "retrieving current state")
	}

	current := newPartialDNSRecords(obj)

	for i, record := range current {
		resp, err := recordSetsClient.Get(record.ZoneRG, record.Zone, record.RelativeName, dns.NS)
		if client.ResponseWasNotFound(resp.Response) {
			continue
		} else if err != nil {
			return nil, microerror.Maskf(err, "retrieving current state: getting record=%#v", record)
		}

		var nameServers []string
		for _, ns := range *resp.NsRecords {
			nameServers = append(nameServers, *ns.Nsdname)
		}

		current[i].NameServers = nameServers
	}

	return current, nil
}

// GetDesiredState returns the desired resource group for this cluster.
func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	o, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Maskf(err, "computing desired state")
	}

	return r.getDesiredState(ctx, o)
}

func (r *Resource) getDesiredState(ctx context.Context, obj azuretpr.CustomObject) (dnsRecords, error) {
	zonesClient, err := r.getDNSZonesClient()
	if err != nil {
		return nil, microerror.Maskf(err, "GetDesiredState")
	}

	desired := newPartialDNSRecords(obj)

	for i, record := range desired {
		zone := record.RelativeName + "." + record.Zone
		resp, err := zonesClient.Get(key.ResourceGroupName(obj), zone)
		if client.ResponseWasNotFound(resp.Response) {
			return dnsRecords{}, nil
		} else if err != nil {
			return nil, microerror.Maskf(err, "GetDesiredState: getting zone=%q", zone)
		}

		var nameServers []string
		for _, ns := range *resp.NameServers {
			nameServers = append(nameServers, ns)
		}

		desired[i].NameServers = nameServers
	}

	return desired, nil
}

// NewUpdatePatch returns the patch creating resource group for this cluster if
// it is needed.
func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*framework.Patch, error) {
	o, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Maskf(err, "NewUpdatePatch")
	}
	c, err := toDNSRecords(currentState)
	if err != nil {
		return nil, microerror.Maskf(err, "NewUpdatePatch")
	}
	d, err := toDNSRecords(desiredState)
	if err != nil {
		return nil, microerror.Maskf(err, "NewUpdatePatch")
	}

	patch := framework.NewPatch()

	updateChange, err := r.newUpdateChange(ctx, o, c, d)
	if err != nil {
		return nil, microerror.Maskf(err, "NewUpdatePatch")
	}

	patch.SetUpdateChange(updateChange)
	return patch, nil
}

// NewDeletePatch returns the patch deleting resource group for this cluster if
// it is needed.
func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*framework.Patch, error) {
	o, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}
	c, err := toDNSRecords(currentState)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}
	d, err := toDNSRecords(desiredState)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}

	patch := framework.NewPatch()

	deleteChange, err := r.newDeleteChange(ctx, o, c, d)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}

	patch.SetDeleteChange(deleteChange)
	return patch, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

// ApplyCreateChange is never called. We do not like it. It is not idempotent.
func (r *Resource) ApplyCreateChange(ctx context.Context, obj, change interface{}) error {
	o, err := toCustomObject(obj)
	if err != nil {
		return microerror.Maskf(err, "creating DNS records in host cluster")
	}

	c, err := toDNSRecords(change)
	if err != nil {
		return microerror.Maskf(err, "creating DNS records in host cluster")
	}

	return r.applyCreateChange(ctx, o, c)
}

func (r *Resource) applyCreateChange(ctx context.Context, obj azuretpr.CustomObject, change dnsRecords) error {
	return nil
}

// ApplyDeleteChange deletes the resource group via the Azure API.
func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, change interface{}) error {
	o, err := toCustomObject(obj)
	if err != nil {
		return microerror.Maskf(err, "creating DNS records in host cluster")
	}
	c, err := toDNSRecords(change)
	if err != nil {
		return microerror.Maskf(err, "creating DNS records in host cluster")
	}

	return r.applyDeleteChange(ctx, o, c)
}

func (r *Resource) applyDeleteChange(ctx context.Context, obj azuretpr.CustomObject, change dnsRecords) error {
	r.logger.LogCtx(ctx, "debug", "deleting host cluster DNS records")

	if len(change) == 0 {
		r.logger.LogCtx(ctx, "debug", "deleting host cluster DNS records: already deleted")
		return nil
	}

	recordSetsClient, err := r.getDNSRecordSetsClient()
	if err != nil {
		return microerror.Maskf(err, "deleting host cluster DNS records")
	}

	for _, record := range change {
		_, err := recordSetsClient.Delete(key.HostClusterResourceGroupName(obj), record.Zone, record.RelativeName, dns.NS, "")
		if err != nil {
			return microerror.Maskf(err, fmt.Sprintf("deleting host cluster DNS record=%#v", record))
		}

		r.logger.LogCtx(ctx, "debug", fmt.Sprintf("deleting host cluster DNS record=%#v", record))
	}

	return nil
}

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, change interface{}) error {
	o, err := toCustomObject(obj)
	if err != nil {
		return microerror.Maskf(err, "ensuring DNS NS records for zones")
	}

	c, err := toDNSRecords(change)
	if err != nil {
		return microerror.Maskf(err, "ensuring DNS NS records for zones")
	}

	return r.applyUpdateChange(ctx, o, c)
}

func (r *Resource) applyUpdateChange(ctx context.Context, obj azuretpr.CustomObject, change dnsRecords) error {
	r.logger.LogCtx(ctx, "debug", "ensuring host cluster DNS records")

	recordSetsClient, err := r.getDNSRecordSetsClient()
	if err != nil {
		return microerror.Maskf(err, "ensuring host cluster DNS records")
	}

	if len(change) == 0 {
		r.logger.LogCtx(ctx, "debug", "ensuring host cluster DNS records: already ensured")
	}

	for _, record := range change {
		r.logger.LogCtx(ctx, "debug", fmt.Sprintf("ensuring host cluster DNS record=%#v", record))

		var params dns.RecordSet
		{
			var nameServers []dns.NsRecord
			for _, ns := range record.NameServers {
				nameServers = append(nameServers, dns.NsRecord{Nsdname: to.StringPtr(ns)})
			}
			params.RecordSetProperties = &dns.RecordSetProperties{
				TTL:       to.Int64Ptr(300),
				NsRecords: &nameServers,
			}
		}

		_, err := recordSetsClient.CreateOrUpdate(key.HostClusterResourceGroupName(obj), record.Zone, record.RelativeName, dns.NS, params, "", "")
		if err != nil {
			return microerror.Maskf(err, fmt.Sprintf("ensuring host cluster DNS record=%#v", record))
		}

		r.logger.LogCtx(ctx, "debug", fmt.Sprintf("ensuring host cluster DNS record=%#v: ensured", record))
	}

	r.logger.LogCtx(ctx, "debug", "ensuring host cluster DNS records: ensured")
	return nil
}

// Underlying returns the underlying resource.
func (r *Resource) Underlying() framework.Resource {
	return r
}

func (r *Resource) getDNSRecordSetsClient() (*dns.RecordSetsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "creating Azure DNS record sets client")
	}

	return azureClients.DNSRecordSetsClient, nil
}

func (r *Resource) getDNSZonesClient() (*dns.ZonesClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "creating Azure DNS zones client")
	}

	return azureClients.DNSZonesClient, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, obj azuretpr.CustomObject, currentState, desiredState dnsRecords) (dnsRecords, error) {
	var change dnsRecords
	for _, d := range desiredState {
		if !currentState.Contains(d) {
			change = append(change, d)
		}
	}

	return change, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, obj azuretpr.CustomObject, currentState, desiredState dnsRecords) (dnsRecords, error) {
	return currentState, nil
}

package collector

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

var (
	vpnConnectionDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("azure_operator", "vpn_connection", "info"),
		"VPN connection informations.",
		[]string{
			"id",
			"name",
			"location",
			"connection_type",
			"connection_status",
			"provisioning_state",
		},
		nil,
	)
)

type VPNConnectionConfig struct {
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger

	AzureSetting             setting.Azure
	HostAzureClientSetConfig client.AzureClientSetConfig
}

type VPNConnection struct {
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	azureSetting             setting.Azure
	hostAzureClientSetConfig client.AzureClientSetConfig
}

func NewVPNConnection(config VPNConnectionConfig) (*VPNConnection, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.AzureSetting.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureSetting.%s", config, err)
	}
	if err := config.HostAzureClientSetConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostAzureClientSetConfig.%s", config, err)
	}

	v := &VPNConnection{
		k8sClient: config.K8sClient,
		logger:    config.Logger,

		azureSetting:             config.AzureSetting,
		hostAzureClientSetConfig: config.HostAzureClientSetConfig,
	}

	return v, nil
}

func (v *VPNConnection) Collect(ch chan<- prometheus.Metric) error {
	vpnConnectionClient, err := v.getVPNConnectionsClient()
	if err != nil {
		return microerror.Mask(err)
	}

	ctx := context.Background()
	resourceGroup := v.azureSetting.HostCluster.ResourceGroup
	connections, err := vpnConnectionClient.ListComplete(ctx, resourceGroup)
	if err != nil {
		return microerror.Mask(err)
	}

	var g errgroup.Group

	for connections.NotDone() {
		c := connections.Value()
		connectionName := to.String(c.Name)

		// ConnectionStatus returned by the API when listing connections is always empty.
		// Details for each connection must be requested in order to get a value for ConnectionStatus.
		g.Go(func() error {
			connection, err := vpnConnectionClient.Get(ctx, resourceGroup, connectionName)
			if err != nil {
				return microerror.Mask(err)
			}

			ch <- prometheus.MustNewConstMetric(
				vpnConnectionDesc,
				prometheus.GaugeValue,
				1,
				to.String(connection.ID),
				connectionName,
				to.String(connection.Location),
				string(connection.ConnectionType),
				string(connection.ConnectionStatus),
				to.String(connection.ProvisioningState),
			)

			return nil
		})

		if err := connections.Next(); err != nil {
			return microerror.Mask(err)
		}
	}

	if err := g.Wait(); err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (v *VPNConnection) Describe(ch chan<- *prometheus.Desc) error {
	ch <- vpnConnectionDesc
	return nil
}

func (v *VPNConnection) getVPNConnectionsClient() (*network.VirtualNetworkGatewayConnectionsClient, error) {
	azureClients, err := client.NewAzureClientSet(v.hostAzureClientSetConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.VirtualNetworkGatewayConnectionsClient, nil
}

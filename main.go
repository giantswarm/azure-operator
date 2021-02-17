package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/microkit/command"
	microserver "github.com/giantswarm/microkit/server"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/versionbundle"
	"github.com/spf13/viper"

	"github.com/giantswarm/azure-operator/v5/flag"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/server"
	"github.com/giantswarm/azure-operator/v5/service"
)

var (
	f = flag.New()
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	err := mainError()
	if err != nil {
		panic(fmt.Sprintf("%#v\n", err))
	}
}

func mainError() error {
	var err error

	ctx := context.Background()
	logger, err := micrologger.New(micrologger.Config{})
	if err != nil {
		return microerror.Mask(err)
	}

	// We define a server factory to create the custom server once all command
	// line flags are parsed and all microservice configuration is sorted out.
	serverFactory := func(v *viper.Viper) microserver.Server {
		// Create a new custom service which implements business logic.
		var newService *service.Service
		{
			c := service.Config{
				Flag:   f,
				Logger: logger,
				Viper:  v,

				Description: project.Description(),
				GitCommit:   project.GitSHA(),
				ProjectName: project.Name(),
				Source:      project.Source(),
				Version:     project.Version(),
			}

			newService, err = service.New(c)
			if err != nil {
				panic(fmt.Sprintf("%#v", microerror.Mask(err)))
			}

			go newService.Boot(ctx)
		}

		// Create a new custom server which bundles our endpoints.
		var newServer microserver.Server
		{
			c := server.Config{
				Logger:  logger,
				Service: newService,
				Viper:   v,

				ProjectName: project.Name(),
			}

			newServer, err = server.New(c)
			if err != nil {
				panic(fmt.Sprintf("%#v", microerror.Mask(err)))
			}
		}

		return newServer
	}

	// Create a new microkit command which manages our custom microservice.
	var newCommand command.Command
	{
		c := command.Config{
			Logger:        logger,
			ServerFactory: serverFactory,

			Description:    project.Description(),
			GitCommit:      project.GitSHA(),
			Name:           project.Name(),
			Source:         project.Source(),
			Version:        project.Version(),
			VersionBundles: []versionbundle.Bundle{project.NewVersionBundle()},
		}

		newCommand, err = command.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	daemonCommand := newCommand.DaemonCommand().CobraCommand()

	daemonCommand.PersistentFlags().String(f.Service.Azure.ClientID, "", "ID of the Active Directory Service Principal that has access to the Management Cluster subscription.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.ClientSecret, "", "Secret of the Active Directory Service Principal that has access to the Management Cluster subscription.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.EnvironmentName, "AZUREPUBLICCLOUD", "Azure Cloud Environment identifier.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.Location, "westeurope", "Location of the host and guset clusters.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.PartnerID, "", "Partner id used in Azure for the attribution partner program.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.SubscriptionID, "", "ID of the Azure Subscription where the Management Cluster is.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.TenantID, "", "ID of the Active Directory Tenant the Management Cluster subscription belongs to.")
	daemonCommand.PersistentFlags().Bool(f.Service.Azure.MSI.Enabled, true, "Whether to enabled Managed Service Identity (MSI).")
	daemonCommand.PersistentFlags().String(f.Service.Azure.HostCluster.CIDR, "10.0.0.0/16", "CIDR of the host cluster virtual network used to create a peering.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.HostCluster.ResourceGroup, "", "Host cluster resource group name.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.HostCluster.Tenant.TenantID, "", "Tenant ID used for the Control Plane cluster.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.HostCluster.Tenant.SubscriptionID, "", "Subscription ID used for the Control Plane cluster.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.HostCluster.Tenant.PartnerID, "", "Partner ID used for the Control Plane cluster.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.HostCluster.VirtualNetwork, "", "Host cluster virtual network name.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.HostCluster.VirtualNetworkGateway, "", "Host cluster virtual network gateway name.")

	daemonCommand.PersistentFlags().String(f.Service.Cluster.BaseDomain, "ghost.westeurope.azure.gigantic.io", "Cluster base domain without k8s/g8s prefixes.")

	daemonCommand.PersistentFlags().Int(f.Service.Cluster.Calico.CIDR, 16, "Calico cidr of guest clusters.")
	daemonCommand.PersistentFlags().Int(f.Service.Cluster.Calico.MTU, 1500, "Calico MTU of guest clusters.")
	daemonCommand.PersistentFlags().String(f.Service.Cluster.Calico.Subnet, "", "Calico subnet of guest clusters.")

	daemonCommand.PersistentFlags().String(f.Service.Cluster.Docker.Daemon.CIDR, "", "CIDR of the Docker daemon bridge configured in guest clusters.")
	daemonCommand.PersistentFlags().String(f.Service.Cluster.Docker.Daemon.ExtraArgs, "", "Extra args of the Docker daemon configured in guest clusters.")

	daemonCommand.PersistentFlags().String(f.Service.Cluster.Etcd.AltNames, "", "Alternative names for guest cluster Calico certificates.")
	daemonCommand.PersistentFlags().Int(f.Service.Cluster.Etcd.Port, 0, "Port of guest cluster etcd.")
	daemonCommand.PersistentFlags().String(f.Service.Cluster.Etcd.Prefix, "", "Prefix of guest cluster etcd.")

	daemonCommand.PersistentFlags().String(f.Service.Cluster.Kubernetes.API.AltNames, "", "Alternative names for guest cluster API certificates.")
	daemonCommand.PersistentFlags().String(f.Service.Cluster.Kubernetes.API.ClusterIPRange, "", "Service IP range within guest clusters.")
	daemonCommand.PersistentFlags().Int(f.Service.Cluster.Kubernetes.API.SecurePort, 0, "Secure port of guest cluster API.")

	daemonCommand.PersistentFlags().String(f.Service.Cluster.Kubernetes.Domain, "", "Base domain for guest clusters.")
	daemonCommand.PersistentFlags().String(f.Service.Cluster.Kubernetes.IngressController.BaseDomain, "", "Base domain for guest cluster Ingress Controller.")
	daemonCommand.PersistentFlags().Int(f.Service.Cluster.Kubernetes.IngressController.InsecurePort, 0, "Insecure port of guest cluster Ingress Controller.")
	daemonCommand.PersistentFlags().Int(f.Service.Cluster.Kubernetes.IngressController.SecurePort, 0, "Secure port of guest cluster Ingress Controller.")
	daemonCommand.PersistentFlags().String(f.Service.Cluster.Kubernetes.Kubelet.AltNames, "", "Alternative names for guest cluster kubelet certificates.")
	daemonCommand.PersistentFlags().Int(f.Service.Cluster.Kubernetes.Kubelet.Port, 0, "Port to bind guest cluster kubelets on.")
	daemonCommand.PersistentFlags().String(f.Service.Cluster.Kubernetes.SSH.UserList, "", "Comma separated list of ssh users and their public key in format `username:publickey`, being installed in the guest cluster nodes.")
	daemonCommand.PersistentFlags().String(f.Service.Installation.Guest.IPAM.Network.CIDR, "10.1.0.0/8", "Guest cluster network segment from which IPAM allocates subnets.")
	daemonCommand.PersistentFlags().Int(f.Service.Installation.Guest.IPAM.Network.SubnetMaskBits, 16, "Number of bits in guest cluster subnet network mask.")
	daemonCommand.PersistentFlags().String(f.Service.Installation.Name, "", "Installation name for tagging Azure resources.")
	daemonCommand.PersistentFlags().String(f.Service.Installation.Tenant.Kubernetes.API.Auth.Provider.OIDC.ClientID, "", "OIDC authorization provider ClientID.")
	daemonCommand.PersistentFlags().String(f.Service.Installation.Tenant.Kubernetes.API.Auth.Provider.OIDC.IssuerURL, "", "OIDC authorization provider IssuerURL.")
	daemonCommand.PersistentFlags().String(f.Service.Installation.Tenant.Kubernetes.API.Auth.Provider.OIDC.UsernameClaim, "", "OIDC authorization provider UsernameClaim.")
	daemonCommand.PersistentFlags().String(f.Service.Installation.Tenant.Kubernetes.API.Auth.Provider.OIDC.GroupsClaim, "", "OIDC authorization provider GroupsClaim.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.Address, "", "Address used to connect to Kubernetes. When empty in-cluster config is created.")
	daemonCommand.PersistentFlags().Bool(f.Service.Kubernetes.InCluster, true, "Whether to use the in-cluster config to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.KubeConfig, "", "KubeConfig used to connect to Kubernetes. When empty other settings are used.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.KubeConfigPath, "", "Optional path to KubeConfig file to connect to Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.CAFile, "", "Certificate authority file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.CrtFile, "", "Certificate file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.KeyFile, "", "Key file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().Bool(f.Service.Tenant.Ignition.Debug.Enabled, false, "Enable services which help debugging ignition.")
	daemonCommand.PersistentFlags().String(f.Service.Tenant.Ignition.Debug.LogsPrefix, "", "Enable services which help debugging ignition.")
	daemonCommand.PersistentFlags().String(f.Service.Tenant.Ignition.Debug.LogsToken, "", "Enable services which help debugging ignition.")
	daemonCommand.PersistentFlags().String(f.Service.Tenant.Ignition.Path, "/opt/ignition", "Default path for the ignition base directory.")
	daemonCommand.PersistentFlags().String(f.Service.Tenant.SSH.SSOPublicKey, "", "Public key for trusted SSO CA.")
	daemonCommand.PersistentFlags().String(f.Service.Sentry.DSN, "", "DSN of the Sentry instance to forward errors to.")
	daemonCommand.PersistentFlags().String(f.Service.Registry.DockerhubToken, "", "Token used to authenticate/authorize to DockerHub.")
	daemonCommand.PersistentFlags().String(f.Service.Registry.Domain, "docker.io", "Image registry domain.")
	daemonCommand.PersistentFlags().StringSlice(f.Service.Registry.Mirrors, []string{}, `Image registry mirror domains. Can be set only if registry domain is "docker.io".`)

	daemonCommand.PersistentFlags().Bool(f.Service.Debug.InsecureStorageAccount, false, "Whether to disable the storage account firewall for tenant clusters.")

	return newCommand.CobraCommand().Execute()
}

package main

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/microkit/command"
	microserver "github.com/giantswarm/microkit/server"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/viper"

	"github.com/giantswarm/azure-operator/flag"
	"github.com/giantswarm/azure-operator/server"
	"github.com/giantswarm/azure-operator/service"
)

const (
	notAvailable = "n/a"
)

var (
	description string     = "The azure-operator manages Kubernetes clusters on Azure."
	f           *flag.Flag = flag.New()
	gitCommit   string     = notAvailable
	name        string     = "azure-operator"
	source      string     = "https://github.com/giantswarm/azure-operator"
)

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
	// line flags are parsed and all microservice configuration is storted out.
	serverFactory := func(v *viper.Viper) microserver.Server {
		// Create a new custom service which implements business logic.
		var newService *service.Service
		{
			c := service.Config{
				Flag:   f,
				Logger: logger,
				Viper:  v,

				Description: description,
				GitCommit:   gitCommit,
				ProjectName: name,
				Source:      source,
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

				ProjectName: name,
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

			Description:    description,
			GitCommit:      gitCommit,
			Name:           name,
			Source:         source,
			VersionBundles: service.NewVersionBundles(),
		}

		newCommand, err = command.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var defaultTemplateVersion string
	{
		if gitCommit != notAvailable {
			defaultTemplateVersion = gitCommit
		}
	}

	daemonCommand := newCommand.DaemonCommand().CobraCommand()

	daemonCommand.PersistentFlags().String(f.Service.Azure.ClientID, "", "ID of the Active Directory Service Principal.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.ClientSecret, "", "Secret of the Active Directory Service Principal.")
	// The cloud environment identifier. Takes values from https://github.com/Azure/go-autorest/blob/ec5f4903f77ed9927ac95b19ab8e44ada64c1356/autorest/azure/environments.go#L13
	daemonCommand.PersistentFlags().String(f.Service.Azure.Cloud, "AZUREPUBLICCLOUD", "Azure Cloud Environment identifier.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.HostCluster.CIDR, "", "CIDR of the host cluster virtual network used to create a peering.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.HostCluster.ResourceGroup, "", "Host cluster resource group name.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.HostCluster.VirtualNetwork, "", "Host cluster virtual network name.")
	daemonCommand.PersistentFlags().Bool(f.Service.Azure.MSI.Enabled, true, "Whether to enabled Managed Service Identity (MSI).")
	daemonCommand.PersistentFlags().String(f.Service.Azure.Location, "", "Location of the host and guset clusters.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.SubscriptionID, "", "ID of the Azure Subscription.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.TenantID, "", "ID of the Active Directory Tenant.")
	daemonCommand.PersistentFlags().String(f.Service.Azure.Template.URI.Version, defaultTemplateVersion, "URI version for ARM template links.")
	daemonCommand.PersistentFlags().String(f.Service.Installation.Name, "", "Installation name for tagging Azure resources.")
	daemonCommand.PersistentFlags().String(f.Service.Installation.Guest.Kubernetes.API.Auth.Provider.OIDC.ClientID, "", "OIDC authorization provider ClientID.")
	daemonCommand.PersistentFlags().String(f.Service.Installation.Guest.Kubernetes.API.Auth.Provider.OIDC.IssuerURL, "", "OIDC authorization provider IssuerURL.")
	daemonCommand.PersistentFlags().String(f.Service.Installation.Guest.Kubernetes.API.Auth.Provider.OIDC.UsernameClaim, "", "OIDC authorization provider UsernameClaim.")
	daemonCommand.PersistentFlags().String(f.Service.Installation.Guest.Kubernetes.API.Auth.Provider.OIDC.GroupsClaim, "", "OIDC authorization provider GroupsClaim.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.Address, "", "Address used to connect to Kubernetes. When empty in-cluster config is created.")
	daemonCommand.PersistentFlags().Bool(f.Service.Kubernetes.InCluster, true, "Whether to use the in-cluster config to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.CAFile, "", "Certificate authority file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.CrtFile, "", "Certificate file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.KeyFile, "", "Key file path to use to authenticate with Kubernetes.")

	newCommand.CobraCommand().Execute()

	return nil
}

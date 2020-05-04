package setup

import (
	"context"
	"fmt"
	"net"
	"time"

	appv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/key"
	"github.com/giantswarm/azure-operator/pkg/project"
)

const (
	azureOperatorChartValues = `
---
Installation:
  V1:
    Guest:
      Kubernetes:
        API:
          Auth:
            Provider:
              OIDC:
                ClientID: "%s"
                IssuerURL: "https://login.microsoftonline.com/%s/v2.0"
                UsernameClaim: "email"
                GroupsClaim: "groups"
      SSH:
        SSOPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDPr6Mxx3cdPNm3v4Ufvo5sRfT7jCgDi7z3wwaCufVrw8am+PBW7toRWBQtGddtp7zsdicHy1+FeWHw09txsbzjupO0yynVAtXSxS8HjsWZOcn0ZRQXMtbbikSxWRs9C255yBswPlD7y9OOiUr8OidIHRYq/vMKIPE+066PqVBYIgO4wR9BRhWPz385+Ob+36K+jkSbniiQr4c8Q545Fm+ilCccLCN1KVVj2pYkCyLHSaEJEp57eyU2ZiBqN0ntgqCVo3xery3HQQalin6Uhqaecwla9bpj48Oo22PLYN2yNhxFU66sSN9TkBquP2VGWlHmWRRg3RPnTY1IPBBC4ea3JOurYOEHydHtoMOGQ6irnd8avqFonXKT2cc/UWUsktv5RwI7S+hUbBdy0o/uX6SbecLIyL+iIIcWL5A0loWyjMEPdDdLjdz72EdnuLVeQohFuSeTpVqpiHugzCCZYwItT7N8QRSgx6wF7j8XkTDZYhWTv9nxtjsRwSDfZJbhsPsgjeQh0z1YJEKZ6RMyrHAzsui/6seFzlgvogRH2iJBzzrKui0uNyE7lQVAeRGHfqUN9YX0DgQ/AvT0BBnCyhMQCD7cJsFJ7A4nRTNsvpPR2uJ2n8fSf2kxXCHH2Tz+CbobVLeZqslKSiz5aO5iKCrHPK7fGnDCKKW8CyYG6V974Q=="
    Name: ci-azure-operator
    Provider:
      Azure:
        Cloud: AZUREPUBLICCLOUD
        HostCluster:
          ResourceGroup: godsmack
          VirtualNetwork: "godsmack"
          VirtualNetworkGateway: "godsmack-vpn-gateway"
          CIDR: "0.0.0.0/0"
        MSI:
          Enabled: true
        Location: %s
    Registry:
      Domain: quay.io
    Secret:
      AzureOperator:
        SecretYaml: |
          service:
            azure:
              clientid: "%s"
              clientsecret: "%s"
              subscriptionid: "%s"
              tenantid: "%s"
              template:
                uri:
                  version: %s
`
)

// provider installs the operator and tenant cluster CR.
func provider(ctx context.Context, config Config) error {
	renderedAzureOperatorChartValues := fmt.Sprintf(azureOperatorChartValues, env.AzureClientID(), env.AzureTenantID(), env.AzureLocation(), env.AzureClientID(), env.AzureClientSecret(), env.AzureSubscriptionID(), env.AzureTenantID(), env.CircleSHA())
	var version string
	{
		// version is the link between an operator and a CustomResource.
		// Operator with version `version` will only reconcile `CustomResources` labeled with `version`.
		version = project.Version()
		if env.TestDir() == "integration/test/update" {
			// When testing the update process, we want the latest release of the operator to reconcile the `CustomResource` and create a cluster.
			// We can then update the label in the `CustomResource`, making the operator under test to reconcile it and update the cluster.
			version = GetLatestOperatorRelease()
		}
	}

	{
		err := installChartPackageBeingTested(ctx, config, renderedAzureOperatorChartValues)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err := installLatestReleaseChartPackage(ctx, config, project.Name(), renderedAzureOperatorChartValues)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring chart CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, appv1alpha1.NewChartCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured chart CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring AzureConfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, providerv1alpha1.NewAzureConfigCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureConfig CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring AzureClusterConfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, v1alpha1.NewAzureClusterConfigCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureClusterConfig CRD exists")
	}

	{
		encryptionSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", key.TestAppReleaseName(), "encryption"),
				Namespace: "default",
				Labels: map[string]string{
					"giantswarm.io/cluster":   env.ClusterID(),
					"giantswarm.io/randomkey": "encryption",
				},
			},
			Data: map[string][]byte{
				"encryption": []byte("B+QdiVynV8Z6bgo1TpAD6Qj1DvRTAx2j/j9EoNlOP38="),
			},
			Type: "Opaque",
		}

		_, err := config.K8sClients.K8sClient().CoreV1().Secrets("default").Create(encryptionSecret)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		_, err := config.K8sClients.K8sClient().CoreV1().Secrets("giantswarm").Create(credentialDefault())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var nodeSSHConfiguration providerv1alpha1.ClusterKubernetesSSH
	{
		if env.SSHPublicKey() != "" {
			nodeSSHConfiguration = providerv1alpha1.ClusterKubernetesSSH{
				UserList: []providerv1alpha1.ClusterKubernetesSSHUser{
					{
						Name:      "e2e",
						PublicKey: env.SSHPublicKey(),
					},
				},
			}
		}
	}

	{
		azureConfig := &providerv1alpha1.AzureConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      env.ClusterID(),
				Namespace: "default",
				Labels: map[string]string{
					"giantswarm.io/cluster":                env.ClusterID(),
					"azure-operator.giantswarm.io/version": version,
					"release.giantswarm.io/version":        ReleaseName,
				},
			},
			Spec: providerv1alpha1.AzureConfigSpec{
				Azure: providerv1alpha1.AzureConfigSpecAzure{
					AvailabilityZones: env.AzureAvailabilityZones(),
					CredentialSecret: providerv1alpha1.CredentialSecret{
						Name:      "credential-default",
						Namespace: "giantswarm",
					},
					DNSZones: providerv1alpha1.AzureConfigSpecAzureDNSZones{
						API: providerv1alpha1.AzureConfigSpecAzureDNSZonesDNSZone{
							ResourceGroup: env.CommonDomainResourceGroup(),
							Name:          env.CommonDomain(),
						},
						Etcd: providerv1alpha1.AzureConfigSpecAzureDNSZonesDNSZone{
							ResourceGroup: env.CommonDomainResourceGroup(),
							Name:          env.CommonDomain(),
						},
						Ingress: providerv1alpha1.AzureConfigSpecAzureDNSZonesDNSZone{
							ResourceGroup: env.CommonDomainResourceGroup(),
							Name:          env.CommonDomain(),
						},
					},
					Masters: []providerv1alpha1.AzureConfigSpecAzureNode{
						{
							VMSize: "Standard_D4_v2",
						},
					},
					VirtualNetwork: providerv1alpha1.AzureConfigSpecAzureVirtualNetwork{
						CIDR:             env.AzureCIDR(),
						MasterSubnetCIDR: env.AzureMasterSubnetCIDR(),
						WorkerSubnetCIDR: env.AzureWorkerSubnetCIDR(),
						CalicoSubnetCIDR: env.AzureCalicoSubnetCIDR(),
					},
					Workers: []providerv1alpha1.AzureConfigSpecAzureNode{
						{
							VMSize: "Standard_D4_v2",
						},
						{
							VMSize: "Standard_D4_v2",
						},
					},
				},
				Cluster: providerv1alpha1.Cluster{
					Calico: providerv1alpha1.ClusterCalico{
						CIDR: 16,
						MTU:  1500,
					},
					Customer: providerv1alpha1.ClusterCustomer{ID: "example-customer"},
					Docker:   providerv1alpha1.ClusterDocker{Daemon: providerv1alpha1.ClusterDockerDaemon{CIDR: "172.17.0.1/16"}},
					Etcd: providerv1alpha1.ClusterEtcd{
						Domain: "etcd." + env.ClusterID() + ".k8s." + env.CommonDomain(),
						Port:   2379,
						Prefix: "giantswarm.io",
					},
					ID: env.ClusterID(),
					Kubernetes: providerv1alpha1.ClusterKubernetes{
						API: providerv1alpha1.ClusterKubernetesAPI{
							ClusterIPRange: "172.31.0.0/16",
							Domain:         "api." + env.ClusterID() + ".k8s." + env.CommonDomain(),
							SecurePort:     443,
						},
						CloudProvider: "azure",
						DNS:           providerv1alpha1.ClusterKubernetesDNS{IP: net.IPv4(172, 31, 0, 10)},
						Domain:        "cluster.local",
						IngressController: providerv1alpha1.ClusterKubernetesIngressController{
							Domain:         "ingress." + env.ClusterID() + ".k8s." + env.CommonDomain(),
							WildcardDomain: "*." + env.ClusterID() + ".k8s." + env.CommonDomain(),
							InsecurePort:   30010,
							SecurePort:     30011,
						},
						Kubelet: providerv1alpha1.ClusterKubernetesKubelet{
							AltNames: "kubernetes,kubernetes.default,kubernetes.default.svc,kubernetes.default.svc.cluster.local",
							Domain:   "worker." + env.ClusterID() + ".k8s." + env.CommonDomain(),
							Labels:   "giantswarm.io/provider=azure,azure-operator.giantswarm.io/version=" + version,
							Port:     10250,
						},
						NetworkSetup: providerv1alpha1.ClusterKubernetesNetworkSetup{Docker: providerv1alpha1.ClusterKubernetesNetworkSetupDocker{Image: "quay.io/giantswarm/k8s-setup-network-environment:1f4ffc52095ac368847ce3428ea99b257003d9b9"}},
						SSH:          nodeSSHConfiguration,
					},
				},
				VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{Version: version},
			},
		}
		_, err := config.K8sClients.G8sClient().ProviderV1alpha1().AzureConfigs("default").Create(azureConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		azureClusterConfig := &v1alpha1.AzureClusterConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      env.ClusterID(),
				Namespace: "default",
				Labels: map[string]string{
					"giantswarm.io/cluster":                env.ClusterID(),
					"azure-operator.giantswarm.io/version": version,
					"release.giantswarm.io/version":        ReleaseName,
				},
			},
			Spec: v1alpha1.AzureClusterConfigSpec{
				Guest: v1alpha1.AzureClusterConfigSpecGuest{
					ClusterGuestConfig: v1alpha1.ClusterGuestConfig{
						AvailabilityZones: len(env.AzureAvailabilityZones()),
						DNSZone:           ".k8s." + env.CommonDomain(),
						ID:                env.ClusterID(),
						Name:              env.ClusterID(),
						Owner:             "giantswarm",
						ReleaseVersion:    ReleaseName,
						VersionBundles: []v1alpha1.ClusterGuestConfigVersionBundle{
							{
								Name:    ReleaseName,
								Version: ReleaseName,
							},
						},
					},
					CredentialSecret: v1alpha1.AzureClusterConfigSpecGuestCredentialSecret{
						Name:      "credential-default",
						Namespace: "giantswarm",
					},
					Masters: []v1alpha1.AzureClusterConfigSpecGuestMaster{
						{
							AzureClusterConfigSpecGuestNode: v1alpha1.AzureClusterConfigSpecGuestNode{
								ID:     "",
								VMSize: "",
							},
						},
					},
					Workers: []v1alpha1.AzureClusterConfigSpecGuestWorker{
						{
							AzureClusterConfigSpecGuestNode: v1alpha1.AzureClusterConfigSpecGuestNode{
								ID:     "",
								VMSize: "",
							},
							Labels: map[string]string{
								"some": "label",
							},
						},
					},
				},
				VersionBundle: v1alpha1.AzureClusterConfigSpecVersionBundle{Version: version},
			},
		}
		_, err := config.K8sClients.G8sClient().CoreV1alpha1().AzureClusterConfigs("default").Create(azureClusterConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

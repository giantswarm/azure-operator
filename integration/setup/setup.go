package setup

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	appv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/helm"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/key"
	"github.com/giantswarm/azure-operator/pkg/project"
)

const values = `
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

var (
	latestOperatorRelease string
)

func init() {
	fmt.Printf("calculating latest %#q release\n", project.Name())

	var err error
	latestOperatorRelease, err = appcatalog.GetLatestVersion(context.Background(), key.DefaultCatalogStorageURL(), project.Name())
	if err != nil {
		panic(fmt.Sprintln("cannot calculate latest operator release from app catalog"))
	}

	fmt.Printf("latest %#q release is %#q\n", project.Name(), latestOperatorRelease)
}

// WrapTestMain setup and teardown e2e testing environment.
func WrapTestMain(m *testing.M, c Config) {
	var r int

	ctx := context.Background()

	err := Setup(ctx, c)
	if err != nil {
		log.Printf("%#v\n", err)
		r = 1
	} else {
		r = m.Run()
	}

	if env.KeepResources() != "true" {
		err := Teardown(c)
		if err != nil {
			log.Printf("%#v\n", err)
			r = 1
		}
	}

	os.Exit(r)
}

// Setup e2e testing environment.
func Setup(ctx context.Context, c Config) error {
	var err error

	err = common(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = provider(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = installResources(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = bastion(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = c.Guest.Setup(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func GetLatestOperatorRelease() string {
	return latestOperatorRelease
}

func installResources(ctx context.Context, config Config) error {
	var err error

	operatorTarballPath := env.OperatorHelmTarballPath()
	if operatorTarballPath == "" {
		config.Logger.LogCtx(ctx, "level", "debug", "message", "getting tarball URL for tested version")

		operatorVersion := fmt.Sprintf("%s-%s", latestOperatorRelease, env.CircleSHA())
		operatorTarballURL, err := appcatalog.NewTarballURL(key.DefaultTestCatalogStorageURL(), project.Name(), operatorVersion)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball URL for tested version is %#q", operatorTarballURL))

		config.Logger.LogCtx(ctx, "level", "debug", "message", "pulling tarball for tested version")

		operatorTarballPath, err = config.HelmClient.PullChartTarball(ctx, operatorTarballURL)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path for tested version is %#q", operatorTarballPath))

	var latestReleasedOperatorTarballPath string
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "getting tarball URL for latest release")

		latestReleaseOperatorTarballURL, err := appcatalog.NewTarballURL(key.DefaultTestCatalogStorageURL(), project.Name(), latestOperatorRelease)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball URL for latest release is %#q", latestReleaseOperatorTarballURL))

		config.Logger.LogCtx(ctx, "level", "debug", "message", "pulling tarball for latest release")

		latestReleasedOperatorTarballPath, err = config.HelmClient.PullChartTarball(ctx, latestReleaseOperatorTarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path for latest release is %#q", latestReleasedOperatorTarballPath))
	}

	{
		defer func() {
			fs := afero.NewOsFs()
			err := fs.Remove(operatorTarballPath)
			if err != nil {
				config.Logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", operatorTarballPath), "stack", fmt.Sprintf("%#v", err))
			}
		}()

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing %#q", project.Name()))

		err = config.HelmClient.InstallReleaseFromTarball(ctx,
			operatorTarballPath,
			key.Namespace(),
			helm.ReleaseName(key.ReleaseName()),
			helm.ValueOverrides([]byte(fmt.Sprintf(values, env.AzureClientID(), env.AzureTenantID(), env.AzureLocation(), env.AzureClientID(), env.AzureClientSecret(), env.AzureSubscriptionID(), env.AzureTenantID(), env.CircleSHA()))),
			helm.InstallWait(true))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed %#q", project.Name()))
	}

	{
		defer func() {
			fs := afero.NewOsFs()
			err := fs.Remove(latestReleasedOperatorTarballPath)
			if err != nil {
				config.Logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", latestReleasedOperatorTarballPath), "stack", fmt.Sprintf("%#v", err))
			}
		}()

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing %#q-%s", project.Name(), "latest-release"))

		err = config.HelmClient.InstallReleaseFromTarball(ctx,
			latestReleasedOperatorTarballPath,
			key.Namespace(),
			helm.ReleaseName(fmt.Sprintf("%s-%s", key.ReleaseName(), "latest-release")),
			helm.ValueOverrides([]byte(fmt.Sprintf(values, env.AzureClientID(), env.AzureTenantID(), env.AzureLocation(), env.AzureClientID(), env.AzureClientSecret(), env.AzureSubscriptionID(), env.AzureTenantID(), env.CircleSHA()))),
			helm.InstallWait(true))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed %#q-%s", project.Name(), "latest-release"))
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring chart CRD exists")

		// The operator will install the CRD on boot but we create chart CRs
		// in the tests so this ensures the CRD is present.
		err = config.K8sClients.CRDClient().EnsureCreated(ctx, appv1alpha1.NewChartCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured chart CRD exists")
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
		version := env.VersionBundleVersion()
		if env.TestDir() == "integration/test/update" {
			version = GetLatestOperatorRelease()
		}
		azureConfig := &providerv1alpha1.AzureConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      env.ClusterID(),
				Namespace: "default",
				Labels: map[string]string{
					"giantswarm.io/cluster":                env.ClusterID(),
					"azure-operator.giantswarm.io/version": version,
					"release.giantswarm.io/version":        "1.0.0",
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
						SSH: providerv1alpha1.ClusterKubernetesSSH{UserList: []providerv1alpha1.ClusterKubernetesSSHUser{
							{
								Name:      "test-user",
								PublicKey: env.SSHPublicKey(),
							},
						}},
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

	return nil
}

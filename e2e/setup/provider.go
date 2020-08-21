package setup

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/apiextensions/pkg/annotation"
	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/crd"
	"github.com/giantswarm/apiextensions/pkg/label"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/e2e/env"
	"github.com/giantswarm/azure-operator/v4/e2e/key"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	key2 "github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	azureOperatorChartValues = `
---
Installation:
  V1:
    Debug:
      InsecureStorageAccount: "true"
    Guest:
      Calico:
        CIDR: ""
        Subnet: ""
      Docker:
        CIDR: ""
      Ingress:
        Version: ""
      Kubectl:
        Version: ""
      IPAM:
        NetworkCIDR: "10.1.0.0/8"
        CIDRMask: 16
      Kubernetes:
        API:
          Auth:
            Provider:
              OIDC:
                ClientID: "%s"
                IssuerURL: "https://login.microsoftonline.com/%s/v2.0"
                UsernameClaim: "email"
                GroupsClaim: "groups"
          ClusterIPRange: "172.31.0.0/16"
          Domain: ""
        ClusterDomain: ""
        IngressController:
          BaseDomain: ""
      SSH:
        SSOPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDPr6Mxx3cdPNm3v4Ufvo5sRfT7jCgDi7z3wwaCufVrw8am+PBW7toRWBQtGddtp7zsdicHy1+FeWHw09txsbzjupO0yynVAtXSxS8HjsWZOcn0ZRQXMtbbikSxWRs9C255yBswPlD7y9OOiUr8OidIHRYq/vMKIPE+066PqVBYIgO4wR9BRhWPz385+Ob+36K+jkSbniiQr4c8Q545Fm+ilCccLCN1KVVj2pYkCyLHSaEJEp57eyU2ZiBqN0ntgqCVo3xery3HQQalin6Uhqaecwla9bpj48Oo22PLYN2yNhxFU66sSN9TkBquP2VGWlHmWRRg3RPnTY1IPBBC4ea3JOurYOEHydHtoMOGQ6irnd8avqFonXKT2cc/UWUsktv5RwI7S+hUbBdy0o/uX6SbecLIyL+iIIcWL5A0loWyjMEPdDdLjdz72EdnuLVeQohFuSeTpVqpiHugzCCZYwItT7N8QRSgx6wF7j8XkTDZYhWTv9nxtjsRwSDfZJbhsPsgjeQh0z1YJEKZ6RMyrHAzsui/6seFzlgvogRH2iJBzzrKui0uNyE7lQVAeRGHfqUN9YX0DgQ/AvT0BBnCyhMQCD7cJsFJ7A4nRTNsvpPR2uJ2n8fSf2kxXCHH2Tz+CbobVLeZqslKSiz5aO5iKCrHPK7fGnDCKKW8CyYG6V974Q=="
        UserList: "john:ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDPr6Mxx3cdPNm3v4Ufvo5sRfT7jCgDi7z3wwaCufVrw8am+PBW7toRWBQtGddtp7zsdicHy1+FeWHw09txsbzjupO0yynVAtXSxS8HjsWZOcn0ZRQXMtbbikSxWRs9C255yBswPlD7y9OOiUr8OidIHRYq/vMKIPE+066PqVBYIgO4wR9BRhWPz385+Ob+36K+jkSbniiQr4c8Q545Fm+ilCccLCN1KVVj2pYkCyLHSaEJEp57eyU2ZiBqN0ntgqCVo3xery3HQQalin6Uhqaecwla9bpj48Oo22PLYN2yNhxFU66sSN9TkBquP2VGWlHmWRRg3RPnTY1IPBBC4ea3JOurYOEHydHtoMOGQ6irnd8avqFonXKT2cc/UWUsktv5RwI7S+hUbBdy0o/uX6SbecLIyL+iIIcWL5A0loWyjMEPdDdLjdz72EdnuLVeQohFuSeTpVqpiHugzCCZYwItT7N8QRSgx6wF7j8XkTDZYhWTv9nxtjsRwSDfZJbhsPsgjeQh0z1YJEKZ6RMyrHAzsui/6seFzlgvogRH2iJBzzrKui0uNyE7lQVAeRGHfqUN9YX0DgQ/AvT0BBnCyhMQCD7cJsFJ7A4nRTNsvpPR2uJ2n8fSf2kxXCHH2Tz+CbobVLeZqslKSiz5aO5iKCrHPK7fGnDCKKW8CyYG6V974Q=="
    Name: godsmack
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

	// idChars represents the character set used to generate cluster IDs.
	// (does not contain 1 and l, to avoid confusion)
	idChars = "023456789abcdefghijkmnopqrstuvwxyz"

	// idLength represents the number of characters used to create a cluster ID.
	idLength = 5
)

var (
	// Use local instance of RNG. Can be overwritten with fixed seed in tests
	// if needed.
	localRng = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// provider installs the operator and tenant cluster CR.
func provider(ctx context.Context, config Config, giantSwarmRelease releasev1alpha1.Release) error {
	renderedAzureOperatorChartValues := fmt.Sprintf(azureOperatorChartValues, env.AzureClientID(), env.AzureTenantID(), env.AzureLocation(), env.AzureClientID(), env.AzureClientSecret(), env.AzureSubscriptionID(), env.AzureTenantID(), env.CircleSHA())
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring AzureConfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("provider.giantswarm.io", "AzureConfig"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureConfig CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring AzureCluster CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("infrastructure.cluster.x-k8s.io", "AzureCluster"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureCluster CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring AzureMachine CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("infrastructure.cluster.x-k8s.io", "AzureMachine"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureMachine CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring AzureMachinePool CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("exp.infrastructure.cluster.x-k8s.io", "AzureMachinePool"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureMachinePool CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring MachinePool CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("exp.cluster.x-k8s.io", "MachinePool"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured MachinePool CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring Cluster CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("cluster.x-k8s.io", "Cluster"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured Cluster CRD exists")
	}

	var operatorVersion string
	{
		// `operatorVersion` is the link between an operator and a `CustomResource`.
		// azure-operator with version `operatorVersion` will only reconcile `AzureConfig` labeled with `operatorVersion`.
		operatorVersion = project.Version()
		if env.TestDir() == "e2e/test/update" {
			// When testing the update process, we want the latest release of the operator to reconcile the `CustomResource` and create a cluster.
			// We can then update the label in the `CustomResource`, making the operator under test to reconcile it and update the cluster.
			operatorVersion = env.GetLatestOperatorRelease()
		}
	}

	{
		err := installChartPackageBeingTested(ctx, config, renderedAzureOperatorChartValues)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		if env.TestDir() == "e2e/test/update" {
			err := installLatestReleaseChartPackage(ctx, config, project.Name(), renderedAzureOperatorChartValues, CatalogStorageURL)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	{
		encryptionSecret := &corev1.Secret{
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

	clusterOperatorVersion, err := key2.ComponentVersion(giantSwarmRelease, "cluster-operator")
	if err != nil {
		return microerror.Mask(err)
	}

	{
		azureClusterConfig := &v1alpha1.AzureClusterConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      env.ClusterID(),
				Namespace: "default",
				Labels: map[string]string{
					"giantswarm.io/cluster":                env.ClusterID(),
					"azure-operator.giantswarm.io/version": operatorVersion,
					"release.giantswarm.io/version":        strings.TrimPrefix(giantSwarmRelease.GetName(), "v"),
				},
			},
			Spec: v1alpha1.AzureClusterConfigSpec{
				Guest: v1alpha1.AzureClusterConfigSpecGuest{
					ClusterGuestConfig: v1alpha1.ClusterGuestConfig{
						AvailabilityZones: len(env.AzureAvailabilityZones()),
						DNSZone:           env.ClusterID() + ".k8s." + env.CommonDomain(),
						ID:                env.ClusterID(),
						Name:              env.ClusterID(),
						Owner:             "giantswarm",
						ReleaseVersion:    giantSwarmRelease.GetName(),
						VersionBundles: []v1alpha1.ClusterGuestConfigVersionBundle{
							{
								Name:    "cert-operator",
								Version: "0.1.0",
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
				VersionBundle: v1alpha1.AzureClusterConfigSpecVersionBundle{Version: clusterOperatorVersion},
			},
		}
		_, err := config.K8sClients.G8sClient().CoreV1alpha1().AzureClusterConfigs("default").Create(azureClusterConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	nodeSSHConfiguration := providerv1alpha1.ClusterKubernetesSSH{UserList: []providerv1alpha1.ClusterKubernetesSSHUser{}}
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
					"azure-operator.giantswarm.io/version": operatorVersion,
					"release.giantswarm.io/version":        strings.TrimPrefix(giantSwarmRelease.GetName(), "v"),
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
							VMSize: env.AzureVMSize(),
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
							VMSize: env.AzureVMSize(),
						},
						{
							VMSize: env.AzureVMSize(),
						},
					},
				},
				Cluster: providerv1alpha1.Cluster{
					Calico: providerv1alpha1.ClusterCalico{
						CIDR:   16,
						MTU:    1500,
						Subnet: env.AzureCalicoSubnetCIDR(),
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
							ClusterIPRange: ClusterIPRange,
							Domain:         "api." + env.ClusterID() + ".k8s." + env.CommonDomain(),
							SecurePort:     443,
						},
						CloudProvider: "azure",
						DNS:           providerv1alpha1.ClusterKubernetesDNS{IP: "172.31.0.10"},
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
							Labels:   "giantswarm.io/provider=azure,azure-operator.giantswarm.io/version=" + operatorVersion,
							Port:     10250,
						},
						NetworkSetup: providerv1alpha1.ClusterKubernetesNetworkSetup{Docker: providerv1alpha1.ClusterKubernetesNetworkSetupDocker{Image: "quay.io/giantswarm/k8s-setup-network-environment:1f4ffc52095ac368847ce3428ea99b257003d9b9"}},
						SSH:          nodeSSHConfiguration,
					},
					Masters: []providerv1alpha1.ClusterNode{},
					Workers: []providerv1alpha1.ClusterNode{},
				},
				VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{Version: operatorVersion},
			},
		}
		_, err := config.K8sClients.G8sClient().ProviderV1alpha1().AzureConfigs("default").Create(azureConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var machinePoolID string
	machinePoolName := "e2e test node pool"
	var azureMachinePool *expcapzv1alpha3.AzureMachinePool

	{
		const maxIDGenRetries = 5
		var retries int

		for ; retries < maxIDGenRetries; retries++ {
			// Generate internal MachinePool ID.
			machinePoolID = NewRandomEntityID()

			azureMachinePool = &expcapzv1alpha3.AzureMachinePool{
				TypeMeta: metav1.TypeMeta{
					APIVersion: expcapzv1alpha3.GroupVersion.String(),
					Kind:       "AzureMachinePool",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      machinePoolID,
					Namespace: metav1.NamespaceDefault,
					Labels: map[string]string{
						capiv1alpha3.ClusterLabelName: env.ClusterID(),
						label.AzureOperatorVersion:    operatorVersion,
						label.Cluster:                 env.ClusterID(),
						label.MachinePool:             machinePoolID,
						label.Organization:            organization,
						label.ReleaseVersion:          strings.TrimPrefix(giantSwarmRelease.GetName(), "v"),
					},
					Annotations: map[string]string{
						annotation.MachinePoolName: machinePoolName,
					},
				},
				Spec: expcapzv1alpha3.AzureMachinePoolSpec{
					Location: env.AzureLocation(),
					Template: expcapzv1alpha3.AzureMachineTemplate{
						VMSize: env.AzureVMSize(),
					},
				},
			}

			err := config.K8sClients.CtrlClient().Create(ctx, azureMachinePool)
			if apierrors.IsAlreadyExists(err) {
				continue
			} else if err != nil {
				return microerror.Mask(err)
			}

			break
		}

		if retries == maxIDGenRetries {
			return microerror.Mask(idSpaceExhaustedError)
		}
	}

	{
		var infrastructureCRRef *corev1.ObjectReference
		{
			s := runtime.NewScheme()
			err := expcapzv1alpha3.AddToScheme(s)
			if err != nil {
				return microerror.Mask(err)
			}

			infrastructureCRRef, err = reference.GetReference(s, azureMachinePool)
			if err != nil {
				config.Logger.LogCtx(ctx, "level", "warning", fmt.Sprintf("cannot create reference to infrastructure CR: %q", err))
				return microerror.Mask(err)
			}
		}

		machinePool := &expcapiv1alpha3.MachinePool{
			TypeMeta: metav1.TypeMeta{
				APIVersion: expcapiv1alpha3.GroupVersion.String(),
				Kind:       "MachinePool",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      machinePoolID,
				Namespace: azureMachinePool.Namespace,
				Labels: map[string]string{
					capiv1alpha3.ClusterLabelName: env.ClusterID(),
					label.AzureOperatorVersion:    operatorVersion,
					label.Cluster:                 env.ClusterID(),
					label.ClusterOperatorVersion:  clusterOperatorVersion,
					label.MachinePool:             machinePoolID,
					label.Organization:            organization,
					label.ReleaseVersion:          strings.TrimPrefix(giantSwarmRelease.GetName(), "v"),
				},
				Annotations: map[string]string{
					annotation.MachinePoolName: machinePoolName,
				},
			},
			Spec: expcapiv1alpha3.MachinePoolSpec{
				ClusterName:    env.ClusterID(),
				Replicas:       to.Int32Ptr(2),
				FailureDomains: env.AzureAvailabilityZonesAsStrings(),
				Template: capiv1alpha3.MachineTemplateSpec{
					Spec: capiv1alpha3.MachineSpec{
						ClusterName:       env.ClusterID(),
						InfrastructureRef: *infrastructureCRRef,
					},
				},
			},
		}

		err := config.K8sClients.CtrlClient().Create(ctx, machinePool)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		spark := &v1alpha1.Spark{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "core.giantswarm.io/v1alpha1",
				Kind:       "Spark",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      machinePoolID,
				Namespace: azureMachinePool.Namespace,
				Labels: map[string]string{
					capiv1alpha3.ClusterLabelName: env.ClusterID(),
					label.Cluster:                 env.ClusterID(),
					label.ReleaseVersion:          strings.TrimPrefix(giantSwarmRelease.GetName(), "v"),
				},
			},
		}

		err := config.K8sClients.CtrlClient().Create(ctx, spark)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func NewRandomEntityID() string {
	pattern := regexp.MustCompile("^[a-z]+$")
	for {
		letterRunes := []rune(idChars)
		b := make([]rune, idLength)
		for i := range b {
			b[i] = letterRunes[localRng.Intn(len(letterRunes))]
		}

		id := string(b)

		if _, err := strconv.Atoi(id); err == nil {
			// string is numbers only, which we want to avoid
			continue
		}

		if pattern.MatchString(id) {
			// strings is letters only, which we also avoid
			continue
		}

		return id
	}
}

package setup

import (
	"context"
	"fmt"
	"time"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v3/integration/env"
	"github.com/giantswarm/azure-operator/v3/integration/key"
	"github.com/giantswarm/azure-operator/v3/pkg/project"
)

const (
	ClusterIPRange = "172.31.0.0/16"
	// We need this cluster-operator version until we use ClusterAPI.
	ClusterOperatorVersion = "0.23.8"
	ReleaseName            = "1.0.0"
)

// common installs components required to run the operator.
func common(ctx context.Context, config Config) error {
	{
		err := config.K8s.EnsureNamespaceCreated(ctx, namespace)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		c := chartvalues.E2ESetupVaultConfig{
			Vault: chartvalues.E2ESetupVaultConfigVault{
				Token: env.VaultToken(),
			},
		}

		values, err := chartvalues.NewE2ESetupVault(c)
		if err != nil {
			return microerror.Mask(err)
		}

		err = config.Release.Install(ctx, key.VaultReleaseName(), release.NewStableVersion(), values, config.Release.Condition().PodExists(ctx, "default", "app=vault"))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err := installCertOperator(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err := installNodeOperator(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err := installClusterOperator(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err := createGSReleaseContainingOperatorVersion(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func installCertOperator(ctx context.Context, config Config) error {
	certOperatorValues := `Installation:
  V1:
    Auth:
      Vault:
        Address: http://vault.default.svc.cluster.local:8200
        Host: ""
        CA:
          TTL: 720h
        Certificate:
          TTL: 24h
        Token:
          TTL: 24h
        Version: ""
    GiantSwarm:
      CertOperator:
        CRD:
          LabelSelector: ""
    Guest:
      Calico:
        CIDR: ""
        Subnet: ""
      Docker:
        CIDR: ""
      IPAM:
        CIDRMask: ""
        NetworkCIDR: ""
        PrivateSubnetMask: ""
        PublicSubnetMask: ""
      Kubernetes:
        API:
          Auth:
            Provider:
              OIDC:
                ClientID: ""
                IssuerURL: ""
                UsernameClaim: ""
                GroupsClaim: ""
          ClusterIPRange: %s
          EndpointBase: k8s.%s
        ClusterDomain: ""
      SSH:
        UserList: ""
    Provider:
      AWS:
        Route53:
          Enabled: false
        S3AccessLogsExpiration: 0
        TrustedAdvisor:
          Enabled: false
      Kind: ""
    Secret:
      CertOperator:
        Service:
          Vault:
            Config:
              Token: %s
    Security:
      RestrictAccess:
        GuestAPI:
          Public: false
`
	err := installLatestReleaseChartPackage(ctx, config, "cert-operator", fmt.Sprintf(certOperatorValues, ClusterIPRange, env.CommonDomain(), env.VaultToken()))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func installNodeOperator(ctx context.Context, config Config) error {
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring drainerconfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, corev1alpha1.NewDrainerConfigCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured drainerconfig CRD exists")
	}

	{
		nodeOperatorValues := `Installation:
  V1:
    Registry:
      Domain: quay.io
`
		err := installLatestReleaseChartPackage(ctx, config, "node-operator", nodeOperatorValues)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

// We can't install cluster-operator "normally" until we use ClusterAPI.
func installClusterOperator(ctx context.Context, config Config) error {
	chartName := "cluster-operator"
	tarballURL := "https://giantswarm.github.com/control-plane-catalog/cluster-operator-0.23.8-1.tgz"
	chartValues := `---
Installation:
  V1:
    Auth:
      Vault:
        Certificate:
          TTL: 48h
    GiantSwarm:
      Release:
        App:
          Config:
            Default: |
              catalog: default
              namespace: kube-system
              useUpgradeForce: true
            Override: |
              chart-operator:
                chart: chart-operator
                namespace:  giantswarm
    Guest:
      Calico:
        CIDR: %s
        Subnet: %s
      Kubernetes:
        API:
          ClusterIPRange: %s
          EndpointBase: k8s.%s
        ClusterDomain: ""
    Provider:
      Kind: "azure"
    Registry:
      Domain: quay.io
    Secret:
      Registry:
        PullSecret:
          DockerConfigJSON: '{ "auths": { "quay.io": { "auth": "Z2lhbnRzd2FybStnb2RzbWFjazo0MzQ3RTJRSVZaN1Y4TzNUOFk4UlhKNFZGTDU2WjUzQ0FaMEMyVjE1TldJQkNNRkxOUjZCUzRCM1FDMzNWUTk2", "email": "" }}}'
`
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring AWSClusterConfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, corev1alpha1.NewAWSClusterConfigCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured AWSClusterConfig CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring AzureClusterConfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, corev1alpha1.NewAzureClusterConfigCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureClusterConfig CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring KVMClusterConfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, corev1alpha1.NewKVMClusterConfigCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured KVMClusterConfig CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball URL for latest %s release is %s", chartName, tarballURL))
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("pulling tarball for latest %s release", chartName))
		chartPackagePath, err := config.HelmClient.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path for latest %s release is %s", chartName, chartPackagePath))
		err = installChart(ctx, config, chartName, fmt.Sprintf(chartValues, env.AzureCalicoSubnetCIDR(), env.AzureCalicoSubnetCIDR(), ClusterIPRange, env.CommonDomain()), chartPackagePath)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func credentialDefault() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "credential-default",
			Namespace: "giantswarm",
			Labels: map[string]string{
				"app":                        "credentiald",
				"giantswarm.io/managed-by":   "credentiald",
				"giantswarm.io/organization": "giantswarm",
				"giantswarm.io/service-type": "system",
			},
		},
		Data: map[string][]byte{
			"azure.azureoperator.clientid":       []byte(env.AzureClientID()),
			"azure.azureoperator.clientsecret":   []byte(env.AzureClientSecret()),
			"azure.azureoperator.subscriptionid": []byte(env.AzureSubscriptionID()),
			"azure.azureoperator.tenantid":       []byte(env.AzureTenantID()),
		},
		Type: "Opaque",
	}
}

func createGSReleaseContainingOperatorVersion(ctx context.Context, config Config) error {
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring Release CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, releasev1alpha1.NewReleaseCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured Release CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring ReleaseCycle CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, releasev1alpha1.NewReleaseCycleCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured ReleaseCycle CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring Release exists", "release", fmt.Sprintf("v%s", ReleaseName))
		_, err := config.K8sClients.G8sClient().ReleaseV1alpha1().Releases().Create(&releasev1alpha1.Release{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("v%s", ReleaseName),
				Namespace: "default",
				Labels: map[string]string{
					"giantswarm.io/managed-by": "release-operator",
					"giantswarm.io/provider":   "azure",
				},
			},
			Spec: releasev1alpha1.ReleaseSpec{
				Apps: []releasev1alpha1.ReleaseSpecApp{},
				Components: []releasev1alpha1.ReleaseSpecComponent{
					{
						Name:    project.Name(),
						Version: project.Version(),
					},
					{
						Name:    "calico",
						Version: "3.10.1",
					},
					{
						Name:    "containerlinux",
						Version: "2345.3.0",
					},
					{
						Name:    "coredns",
						Version: "1.6.5",
					},
					{
						Name:    "etcd",
						Version: "3.3.17",
					},
					{
						Name:    "kubernetes",
						Version: "1.16.8",
					},
				},
				Date:  &metav1.Time{Time: time.Unix(10, 0)},
				State: "active",
			},
		})
		if err != nil {
			return microerror.Mask(err)
		}
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured Release exists", "release", fmt.Sprintf("v%s", ReleaseName))
	}

	return nil
}

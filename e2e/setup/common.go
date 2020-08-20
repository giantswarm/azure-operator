package setup

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/crd"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/e2e/env"
	"github.com/giantswarm/azure-operator/v4/e2e/key"
	key2 "github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	ClusterIPRange = "172.31.0.0/16"
)

// common installs components required to run the operator.
func common(ctx context.Context, config Config, giantSwarmRelease releasev1alpha1.Release) error {
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
		err := installComponentsFromRelease(ctx, config, giantSwarmRelease)
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

	return nil
}

func installComponentsFromRelease(ctx context.Context, config Config, giantSwarmRelease releasev1alpha1.Release) error {
	clusterOperatorVersion, err := key2.ComponentVersion(giantSwarmRelease, "cluster-operator")
	if err != nil {
		return microerror.Mask(err)
	}

	err = installClusterOperator(ctx, config, clusterOperatorVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	certOperatorVersion, err := key2.ComponentVersion(giantSwarmRelease, "cert-operator")
	if err != nil {
		return microerror.Mask(err)
	}

	err = installCertOperator(ctx, config, certOperatorVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func installCertOperator(ctx context.Context, config Config, version string) error {
	chartName := "cert-operator"
	tarballURL := fmt.Sprintf("https://giantswarm.github.com/control-plane-catalog/%s-%s.tgz", chartName, version)
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
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring CertConfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("core.giantswarm.io", "CertConfig"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured CertConfig CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball URL for %#q release is %#q", chartName, tarballURL))
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("pulling tarball for %#q release", chartName))
		chartPackagePath, err := config.HelmClient.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path for %#q release is %#q", chartName, chartPackagePath))
		err = installChart(ctx, config, chartName, fmt.Sprintf(certOperatorValues, ClusterIPRange, env.CommonDomain(), env.VaultToken()), chartPackagePath)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func installNodeOperator(ctx context.Context, config Config) error {
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring drainerconfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("core.giantswarm.io", "DrainerConfig"), backoff.NewMaxRetries(7, 1*time.Second))
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
		err := installLatestReleaseChartPackage(ctx, config, "node-operator", nodeOperatorValues, CatalogStorageURL)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func installClusterOperator(ctx context.Context, config Config, version string) error {
	chartName := "cluster-operator"
	tarballURL := fmt.Sprintf("https://giantswarm.github.com/control-plane-catalog/%s-%s.tgz", chartName, version)
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

		err := config.LegacyK8sClients.CRDClient().EnsureCreated(ctx, corev1alpha1.NewAWSClusterConfigCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured AWSClusterConfig CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring AzureClusterConfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("core.giantswarm.io", "AzureClusterConfig"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureClusterConfig CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring KVMClusterConfig CRD exists")

		err := config.LegacyK8sClients.CRDClient().EnsureCreated(ctx, corev1alpha1.NewKVMClusterConfigCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured KVMClusterConfig CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring App CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, v1alpha1.NewAppCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured App CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball URL for %#q release is %#q", chartName, tarballURL))
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("pulling tarball for %#q release", chartName))
		chartPackagePath, err := config.HelmClient.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path for %#q release is %#q", chartName, chartPackagePath))
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

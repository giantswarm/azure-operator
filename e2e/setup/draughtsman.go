package setup

import (
	"context"
	"fmt"
	"github.com/giantswarm/azure-operator/v4/e2e/env"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DraughtsmanNamespace     = "draughtsman"
	DraughtsmanConfigMapName = "draughtsman-values-configmap"
	DraughtsmanSecretName    = "draughtsman-values-secret"

	draughtsmanConfigMap = `Auth:
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
      ClusterService:
        Release:
          Endpoints: http://cluster-operator:8000,http://cert-operator:8000,http://azure-operator:8000
      Release:
        App:
          Config:
            Default: |
              catalog: default
              namespace: kube-system
              useUpgradeForce: true
            Override: |
              # chart-operator must be installed first so the chart CRD is
              # created in the tenant cluster.
              chart-operator:
                chart:     "chart-operator"
                namespace: "giantswarm"
              # Upgrade force is disabled to avoid affecting customer workloads.
              coredns:
                useUpgradeForce: false
        IndexBlob: ""
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
        AdvancedMonitoringEC2: false
        AvailabilityZones: []
        CNI:
          ExternalSNAT: true
        DeleteLoggingBucket: true
        EC2:
          Instance:
            Allowed:
            - m4.xlarge
            Capabilities:
              m4.xlarge:
                cpu_cores: 4
                description: M4 General Purpose Extra Large
                memory_size_gb: 16
                storage_size_gb: 0
            Default: m4.xlarge
        Encrypter: kms
        ImageID: ""
        IncludeTags: true
        PodInfraContainerImage: ""
        PublicRouteTableNames: ""
        Route53:
          Enabled: true
        RouteTableNames: e2e_private_0,e2e_private_1,e2e_private_2
        S3AccessLogsExpiration: 365
        TrustedAdvisor:
          Enabled: false
        VPCPeerID: ""
      Azure:
        AvailabilityZones:
        - 1
        - 2
        - 3
        Cloud: AZUREPUBLICCLOUD
        HostCluster:
          CIDR: 10.0.0.0/16
          ResourceGroup: e2e
          VirtualNetwork: e2e
          VirtualNetworkGateway: e2e-vpn-gateway
        Location: westeurope
        MSI:
          Enabled: true
        VM:
          VmSize:
            Allowed:
            - Standard_D4s_v3
            Capabilities:
              Standard_D4s_v3:
                additionalProperties: {}
                description: Dsv3-series, general purpose, 160-190 ACU, premium storage supported
                maxDataDiskCount: 8
                memoryInMb: 17179.869184
                name: Standard_D4s_v3
                numberOfCores: 4
                osDiskSizeInMb: 1047552
                resourceDiskSizeInMb: 34359.738368
            Default: Standard_D4s_v3
      Kind: "azure"
    Security:
      RestrictAccess:
        GuestAPI:
          Public: false
`

	draugthsmanSecret = `Secret:
     CertOperator:
       Service:
         Vault:
           Config:
             Token: %s
     Registry:
       PullSecret:
         DockerConfigJSON: |-
           {
             "auths": {
               "quay.io": {
                 "auth": "%s",
                 "email": ""
               }
             }
           }
`
)

func ensureDraughtsman(ctx context.Context, config Config) error {
	{
		err := config.K8s.EnsureNamespaceCreated(ctx, "draughtsman")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create the draughtsman configmap.
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring draughtsman configmap")
		configMap := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      DraughtsmanConfigMapName,
				Namespace: DraughtsmanNamespace,
			},
			Data: map[string]string{
				"values": getDraughtsmanConfigmap(),
			},
		}
		_, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(DraughtsmanNamespace).Create(ctx, &configMap, metav1.CreateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured draughtsman configmap")
	}

	// Create the draughtsman secret.
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring draughtsman secret")
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      DraughtsmanSecretName,
				Namespace: DraughtsmanNamespace,
			},
			StringData: map[string]string{
				"values": getDraughtsmanSecret(),
			},
		}
		_, err := config.K8sClients.K8sClient().CoreV1().Secrets(DraughtsmanNamespace).Create(ctx, &secret, metav1.CreateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured draughtsman secret")
	}

	return nil
}

func getDraughtsmanConfigmap() string {
	compiled := fmt.Sprintf(draughtsmanConfigMap, ClusterIPRange, env.CommonDomain())

	return fmt.Sprintf("Installation:\n  V1:\n    %s", compiled)
}

func getDraughtsmanSecret() string {
	compiled := fmt.Sprintf(draugthsmanSecret, env.VaultToken(), env.RegistryPullSecret())
	return fmt.Sprintf("Installation:\n  V1:\n    %s", compiled)
}

func getDraughtsmanMergedConfig(additional string) string {
	configMap := fmt.Sprintf(draughtsmanConfigMap, ClusterIPRange, env.CommonDomain())
	secret := fmt.Sprintf(draugthsmanSecret, env.VaultToken(), env.RegistryPullSecret())
	return fmt.Sprintf("Installation:\n  V1:\n    %s\n    %s\n    %s", configMap, secret, additional)
}

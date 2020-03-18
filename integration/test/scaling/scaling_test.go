// +build k8srequired

package scaling

import (
	"context"
	"net"
	"testing"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/key"
)

func Test_Scaling_Workers(t *testing.T) {
	{
		azureConfig := &providerv1alpha1.AzureConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.TestAppReleaseName(),
				Namespace: "giantswarm",
				Labels: map[string]string{
					"azure-operator.giantswarm.io/version": env.VersionBundleVersion(),
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
							VMSize: "Standard_A1",
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
							VMSize: "Standard_A1",
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
						Domain: "etcd" + env.ClusterID() + ".k8s." + env.CommonDomain(),
						Port:   2379,
						Prefix: "giantswarm.io",
					},
					ID: env.ClusterID(),
					Kubernetes: providerv1alpha1.ClusterKubernetes{
						API: providerv1alpha1.ClusterKubernetesAPI{
							ClusterIPRange: "172.31.0.0/16",
							Domain:         "api" + env.ClusterID() + ".k8s." + env.CommonDomain(),
							SecurePort:     443,
						},
						CloudProvider: "azure",
						DNS:           providerv1alpha1.ClusterKubernetesDNS{IP: net.IPv4(172, 31, 0, 10)},
						Domain:        "cluster.local",
						IngressController: providerv1alpha1.ClusterKubernetesIngressController{
							Domain:         "ingress" + env.ClusterID() + ".k8s." + env.CommonDomain(),
							WildcardDomain: "*" + env.ClusterID() + ".k8s." + env.CommonDomain(),
							InsecurePort:   30010,
							SecurePort:     30011,
						},
						Kubelet: providerv1alpha1.ClusterKubernetesKubelet{
							AltNames: "kubernetes,kubernetes.default,kubernetes.default.svc,kubernetes.default.svc.cluster.local",
							Domain:   "worker" + env.ClusterID() + ".k8s." + env.CommonDomain(),
							Labels:   "giantswarm.io/provider=azure,azure-operator.giantswarm.io/version=" + env.VersionBundleVersion(),
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
				VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{Version: env.VersionBundleVersion()},
			},
		}
		_, err := config.K8sClients.G8sClient().ProviderV1alpha1().AzureConfigs("giantswarm").Create(azureConfig)
		if err != nil {
			t.Fatalf("%#v", err)
		}
	}

	err := scalingTest.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

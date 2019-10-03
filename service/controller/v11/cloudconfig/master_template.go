package cloudconfig

import (
	"encoding/base64"
	"fmt"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_4_8_0"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v11/encrypter"
	"github.com/giantswarm/azure-operator/service/controller/v11/templates/ignition"
)

// NewMasterCloudConfig generates a new master cloudconfig and returns it as a
// base64 encoded string.
func (c CloudConfig) NewMasterCloudConfig(customObject providerv1alpha1.AzureConfig, clusterCerts certs.Cluster, encrypter encrypter.Interface) (string, error) {
	apiserverEncryptionKey, err := c.getEncryptionkey(customObject)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// On Azure only master nodes access etcd, so it is locked down.
	customObject.Spec.Cluster.Etcd.Domain = "127.0.0.1"
	customObject.Spec.Cluster.Etcd.Port = 2379

	var k8sAPIExtraArgs []string
	{
		k8sAPIExtraArgs = append(k8sAPIExtraArgs, "--cloud-config=/etc/kubernetes/config/azure.yaml")

		if c.OIDC.ClientID != "" {
			k8sAPIExtraArgs = append(k8sAPIExtraArgs, fmt.Sprintf("--oidc-client-id=%s", c.OIDC.ClientID))
		}
		if c.OIDC.IssuerURL != "" {
			k8sAPIExtraArgs = append(k8sAPIExtraArgs, fmt.Sprintf("--oidc-issuer-url=%s", c.OIDC.IssuerURL))
		}
		if c.OIDC.UsernameClaim != "" {
			k8sAPIExtraArgs = append(k8sAPIExtraArgs, fmt.Sprintf("--oidc-username-claim=%s", c.OIDC.UsernameClaim))
		}
		if c.OIDC.GroupsClaim != "" {
			k8sAPIExtraArgs = append(k8sAPIExtraArgs, fmt.Sprintf("--oidc-groups-claim=%s", c.OIDC.GroupsClaim))
		}
	}

	// NOTE in Azure we disable Calico right now. This is due to a transitioning
	// phase. The k8scloudconfig templates require certain calico valus to be set
	// nonetheless. So we set them here. Later when the Calico setup is
	// straightened out we can improve the handling here.
	customObject.Spec.Cluster.Calico.Subnet = c.azureNetwork.Calico.IP.String()
	customObject.Spec.Cluster.Calico.CIDR, _ = c.azureNetwork.Calico.Mask.Size()

	var params k8scloudconfig.Params
	{
		be := baseExtension{
			azure:        c.azure,
			azureConfig:  c.azureConfig,
			calicoCIDR:   c.azureNetwork.Calico.String(),
			clusterCerts: clusterCerts,
			customObject: customObject,
			encrypter:    encrypter,
			vnetCIDR:     customObject.Spec.Azure.VirtualNetwork.CIDR,
		}

		params = k8scloudconfig.DefaultParams()
		params.APIServerEncryptionKey = apiserverEncryptionKey
		params.Cluster = customObject.Spec.Cluster
		params.DisableCalico = true
		params.DisableIngressControllerService = true
		params.EtcdPort = customObject.Spec.Cluster.Etcd.Port
		params.Hyperkube = k8scloudconfig.Hyperkube{
			Apiserver: k8scloudconfig.HyperkubeApiserver{
				Pod: k8scloudconfig.HyperkubePod{
					HyperkubePodHostExtraMounts: []k8scloudconfig.HyperkubePodHostMount{
						{
							Name:     "k8s-config",
							Path:     "/etc/kubernetes/config/",
							ReadOnly: true,
						},
						{
							Name:     "identity-settings",
							Path:     "/var/lib/waagent/",
							ReadOnly: true,
						},
					},
					CommandExtraArgs: k8sAPIExtraArgs,
				},
			},
			ControllerManager: k8scloudconfig.HyperkubeControllerManager{
				Pod: k8scloudconfig.HyperkubePod{
					HyperkubePodHostExtraMounts: []k8scloudconfig.HyperkubePodHostMount{
						{
							Name:     "identity-settings",
							Path:     "/var/lib/waagent/",
							ReadOnly: true,
						},
					},
					CommandExtraArgs: []string{
						"--cloud-config=/etc/kubernetes/config/azure.yaml",
						"--allocate-node-cidrs=true",
						"--cluster-cidr=" + c.azureNetwork.Calico.String(),
					},
				},
			},
			Kubelet: k8scloudconfig.HyperkubeKubelet{
				Docker: k8scloudconfig.HyperkubeDocker{
					RunExtraArgs: []string{
						"-v /var/lib/waagent:/var/lib/waagent:ro",
					},
					CommandExtraArgs: []string{
						"--cloud-config=/etc/kubernetes/config/azure.yaml",
					},
				},
			},
		}

		params.Extension = &masterExtension{
			baseExtension: be,
		}
		params.ExtraManifests = []string{
			"calico-azure.yaml",
			"k8s-ingress-loadbalancer.yaml",
		}
		params.SSOPublicKey = c.ssoPublicKey
	}
	ignitionPath := k8scloudconfig.GetIgnitionPath(c.ignitionPath)
	params.Files, err = k8scloudconfig.RenderFiles(ignitionPath, params)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return newCloudConfig(k8scloudconfig.MasterTemplate, params)
}

type masterExtension struct {
	baseExtension
}

// Files allows files to be injected into the master cloudconfig.
func (me *masterExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	filesMeta := []k8scloudconfig.FileMetadata{
		{
			AssetContent: ignition.CalicoAzureResources,
			Path:         "/srv/calico-azure.yaml",
			Owner: k8scloudconfig.Owner{
				Group: k8scloudconfig.Group{
					Name: FileOwnerGroupName,
				},
				User: k8scloudconfig.User{
					Name: FileOwnerUserName,
				},
			},
			Permissions: FilePermission,
		},
		{
			AssetContent: ignition.CloudProviderConf,
			Path:         "/etc/kubernetes/config/azure.yaml",
			Owner: k8scloudconfig.Owner{
				Group: k8scloudconfig.Group{
					ID: FileOwnerGroupIDNobody,
				},
				User: k8scloudconfig.User{
					Name: FileOwnerUserName,
				},
			},
			Permissions: CloudProviderFilePermission,
		},
		{
			AssetContent: ignition.DefaultStorageClass,
			Path:         "/srv/default-storage-class.yaml",
			Owner: k8scloudconfig.Owner{
				Group: k8scloudconfig.Group{
					Name: FileOwnerGroupName,
				},
				User: k8scloudconfig.User{
					Name: FileOwnerUserName,
				},
			},
			Permissions: FilePermission,
		},
		{
			AssetContent: ignition.IngressLB,
			Path:         "/srv/k8s-ingress-loadbalancer.yaml",
			Owner: k8scloudconfig.Owner{
				Group: k8scloudconfig.Group{
					Name: FileOwnerGroupName,
				},
				User: k8scloudconfig.User{
					Name: FileOwnerUserName,
				},
			},
			Permissions: FilePermission,
		},
	}

	certFiles := certs.NewFilesClusterMaster(me.clusterCerts)
	data := me.templateData(certFiles)

	var fileAssets []k8scloudconfig.FileAsset

	for _, fm := range filesMeta {
		c, err := k8scloudconfig.RenderFileAssetContent(fm.AssetContent, data)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		asset := k8scloudconfig.FileAsset{
			Metadata: fm,
			Content:  c,
		}

		fileAssets = append(fileAssets, asset)
	}

	var certsMeta []k8scloudconfig.FileMetadata
	for _, f := range certFiles {
		encryptedData, err := me.encrypter.Encrypt(f.Data)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		m := k8scloudconfig.FileMetadata{
			AssetContent: string(encryptedData),
			Path:         f.AbsolutePath + ".enc",
			Owner: k8scloudconfig.Owner{
				Group: k8scloudconfig.Group{
					Name: FileOwnerGroupName,
				},
				User: k8scloudconfig.User{
					Name: FileOwnerUserName,
				},
			},
			Permissions: CertFilePermission,
		}
		certsMeta = append(certsMeta, m)
	}

	for _, cm := range certsMeta {
		c := base64.StdEncoding.EncodeToString([]byte(cm.AssetContent))
		asset := k8scloudconfig.FileAsset{
			Metadata: cm,
			Content:  c,
		}

		fileAssets = append(fileAssets, asset)
	}

	return fileAssets, nil
}

// Units allows systemd units to be injected into the master cloudconfig.
func (me *masterExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	unitsMeta := []k8scloudconfig.UnitMetadata{
		{
			AssetContent: ignition.AzureCNINatRules,
			Name:         "azure-cni-nat-rules.service",
			Enabled:      true,
		},
		{
			AssetContent: ignition.CertificateDecrypterUnit,
			Name:         "certificate-decrypter.service",
			Enabled:      true,
		},
		{
			AssetContent: ignition.EtcdMountUnit,
			Name:         "var-lib-etcd.mount",
			Enabled:      true,
		},
		{
			AssetContent: ignition.DockerMountUnit,
			Name:         "var-lib-docker.mount",
			Enabled:      true,
		},
		{
			AssetContent: ignition.IngressLBUnit,
			Name:         "ingress-lb.service",
			Enabled:      true,
		},
		{
			AssetContent: ignition.VNICConfigurationUnit,
			Name:         "vnic-configuration.service",
			Enabled:      true,
		},
	}

	certFiles := certs.NewFilesClusterMaster(me.clusterCerts)
	data := me.templateData(certFiles)

	var newUnits []k8scloudconfig.UnitAsset

	for _, fm := range unitsMeta {
		c, err := k8scloudconfig.RenderAssetContent(fm.AssetContent, data)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		unitAsset := k8scloudconfig.UnitAsset{
			Metadata: fm,
			Content:  c,
		}

		newUnits = append(newUnits, unitAsset)
	}

	return newUnits, nil
}

// VerbatimSections allows sections to be embedded in the master cloudconfig.
func (me *masterExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}

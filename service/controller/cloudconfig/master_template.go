package cloudconfig

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/giantswarm/apiextensions/v6/pkg/annotation"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v16/pkg/template"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v7/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v7/service/controller/key"
	"github.com/giantswarm/azure-operator/v7/service/controller/templates/ignition"
)

const (
	defaultEtcdPort            = 2379
	EtcdInitialClusterStateNew = "new"

	encryptionConfigFilePath = "/etc/kubernetes/encryption/k8s-encryption-config.yaml"
)

// NewMasterCloudConfig generates a new master cloudconfig and returns it as a
// base64 encoded string.
func (c CloudConfig) NewMasterTemplate(ctx context.Context, data IgnitionTemplateData, encrypter encrypter.Interface) (string, error) {
	var err error
	var k8sAPIExtraArgs []string
	{
		oidcExtraArgs := c.oidcExtraArgs(ctx, data)
		k8sAPIExtraArgs = append(k8sAPIExtraArgs, oidcExtraArgs...)
	}

	var params k8scloudconfig.Params
	{
		be := baseExtension{
			azure:                        c.azure,
			azureClientCredentialsConfig: c.azureClientCredentials,
			calicoCIDR:                   data.CustomObject.Spec.Azure.VirtualNetwork.CalicoSubnetCIDR,
			certFiles:                    data.MasterCertFiles,
			customObject:                 data.CustomObject,
			encrypter:                    encrypter,
			subscriptionID:               c.subscriptionID,
			vnetCIDR:                     data.CustomObject.Spec.Azure.VirtualNetwork.CIDR,
		}

		params = k8scloudconfig.Params{}
		params.BaseDomain = key.ClusterBaseDomain(data.CustomObject)
		params.Cluster = data.CustomObject.Spec.Cluster
		params.CalicoPolicyOnly = true
		params.DockerhubToken = c.dockerhubToken
		params.DisableIngressControllerService = true
		params.EnableCronJobTimeZone = true
		params.Etcd.ClientPort = defaultEtcdPort
		params.Etcd.HighAvailability = false
		params.Etcd.InitialClusterState = EtcdInitialClusterStateNew
		params.ExternalCloudControllerManager = true
		params.Kubernetes = k8scloudconfig.Kubernetes{
			Apiserver: k8scloudconfig.KubernetesPodOptions{
				HostExtraMounts: []k8scloudconfig.KubernetesPodOptionsHostMount{
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
			ControllerManager: k8scloudconfig.KubernetesPodOptions{
				HostExtraMounts: []k8scloudconfig.KubernetesPodOptionsHostMount{
					{
						Name:     "identity-settings",
						Path:     "/var/lib/waagent/",
						ReadOnly: true,
					},
				},
				CommandExtraArgs: []string{
					"--allocate-node-cidrs=true",
					"--cluster-cidr=" + data.CustomObject.Spec.Azure.VirtualNetwork.CalicoSubnetCIDR,
				},
			},
			Kubelet: k8scloudconfig.KubernetesDockerOptions{
				RunExtraArgs: []string{
					"-v /var/lib/waagent:/var/lib/waagent:ro",
				},
			},
		}

		encryptedEncryptionConfig, err := encrypter.Encrypt(data.EncryptionConf)
		if err != nil {
			return "", microerror.Mask(err)
		}

		params.Extension = &masterExtension{
			baseExtension:             be,
			encryptedEncryptionConfig: encryptedEncryptionConfig,
		}
		params.ExtraManifests = []string{}
		params.Debug = k8scloudconfig.Debug{
			Enabled:    c.ignition.Debug,
			LogsPrefix: c.ignition.LogsPrefix,
			LogsToken:  c.ignition.LogsToken,
		}
		params.Images = data.Images
		params.RegistryMirrors = c.registryMirrors
		params.Versions = data.Versions
		params.SSOPublicKey = c.ssoPublicKey
		params.DisableEncryptionAtREST = true // encryption key is now managed by encription-provider-operator
	}
	ignitionPath := k8scloudconfig.GetIgnitionPath(c.ignition.Path)
	params.Files, err = k8scloudconfig.RenderFiles(ignitionPath, params)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return newCloudConfig(k8scloudconfig.MasterTemplate, params)
}

type masterExtension struct {
	baseExtension

	encryptedEncryptionConfig []byte
}

// oidcExtraArgs returns oidc parameters reading the configuration from `Cluster` annotations.
// It uses the oidc configuration passed to the operator as fallback.
func (c CloudConfig) oidcExtraArgs(ctx context.Context, data IgnitionTemplateData) []string {
	var k8sAPIExtraArgs []string

	oidcClientID, oidcClientIDExists := data.Cluster.Annotations[annotation.OIDCClientID]
	oidcIssuerURL, oidcIssuerURLExists := data.Cluster.Annotations[annotation.OIDCIssuerURL]
	oidcUsernameClaim, oidcUsernameClaimExists := data.Cluster.Annotations[annotation.OIDCUsernameClaim]
	oidcGroupsClaim, oidcGroupsClaimExists := data.Cluster.Annotations[annotation.OIDCGroupClaim]

	if !oidcClientIDExists {
		oidcClientID = c.OIDC.ClientID
	}

	if !oidcIssuerURLExists {
		oidcIssuerURL = c.OIDC.IssuerURL
	}

	if !oidcUsernameClaimExists {
		oidcUsernameClaim = c.OIDC.UsernameClaim
	}

	if !oidcGroupsClaimExists {
		oidcGroupsClaim = c.OIDC.GroupsClaim
	}

	if oidcClientID != "" {
		k8sAPIExtraArgs = append(k8sAPIExtraArgs, fmt.Sprintf("--oidc-client-id=%s", oidcClientID))
	}

	if oidcIssuerURL != "" {
		k8sAPIExtraArgs = append(k8sAPIExtraArgs, fmt.Sprintf("--oidc-issuer-url=%s", oidcIssuerURL))
	}

	if oidcUsernameClaim != "" {
		k8sAPIExtraArgs = append(k8sAPIExtraArgs, fmt.Sprintf("--oidc-username-claim=%s", oidcUsernameClaim))
	}

	if oidcGroupsClaim != "" {
		k8sAPIExtraArgs = append(k8sAPIExtraArgs, fmt.Sprintf("--oidc-groups-claim=%s", oidcGroupsClaim))
	}

	return k8sAPIExtraArgs
}

// Files allows files to be injected into the master cloudconfig.
func (me *masterExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	filesMeta := []k8scloudconfig.FileMetadata{
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
		{ // Only needed until https://github.com/kinvolk/init/pull/41 is included into flatcar image.
			AssetContent: ignition.UdevRules,
			Path:         "/etc/udev/rules.d/66-azure-storage.rules",
			Owner: k8scloudconfig.Owner{
				Group: k8scloudconfig.Group{
					Name: FileOwnerGroupName,
				},
				User: k8scloudconfig.User{
					Name: FileOwnerUserName,
				},
			},
			Permissions: CloudProviderFilePermission,
		},
	}

	data := me.templateData(me.certFiles)

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
	for _, f := range me.certFiles {
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

	// Add encryption at rest config.
	{
		m := k8scloudconfig.FileMetadata{
			Path: encryptionConfigFilePath + ".enc",
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

		asset := k8scloudconfig.FileAsset{
			Metadata: m,
			Content:  base64.StdEncoding.EncodeToString(me.encryptedEncryptionConfig),
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
			AssetContent: ignition.KubeletMountUnit,
			Name:         "var-lib-kubelet.mount",
			Enabled:      true,
		},
		{
			AssetContent: ignition.VNICConfigurationUnit,
			Name:         "vnic-configuration.service",
			Enabled:      true,
		},
	}

	data := me.templateData(me.certFiles)

	// To use the certificate decrypter unit for the etcd data encryption config file.
	data.certificateDecrypterUnitParams.CertsPaths = append(data.certificateDecrypterUnitParams.CertsPaths, encryptionConfigFilePath)

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

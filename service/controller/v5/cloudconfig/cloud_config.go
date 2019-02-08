package cloudconfig

import (
	"crypto/aes"
	"encoding/hex"
	"fmt"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_4_0_0"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/randomkeys"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
	"github.com/giantswarm/azure-operator/service/network"
)

const (
	FileOwnerUser  = "root"
	FileOwnerGroup = "root"
	FilePermission = 0700
)

type Config struct {
	CertsSearcher      certs.Interface
	Logger             micrologger.Logger
	RandomkeysSearcher randomkeys.Interface

	Azure setting.Azure
	// TODO(pk) remove as soon as we sort calico in Azure provider.
	AzureConfig  client.AzureClientSetConfig
	AzureNetwork network.Subnets
	IgnitionPath string
	OIDC         setting.OIDC
	SSOPublicKey string
}

type CloudConfig struct {
	certsSearcher      certs.Interface
	logger             micrologger.Logger
	randomkeysSearcher randomkeys.Interface

	azure        setting.Azure
	azureConfig  client.AzureClientSetConfig
	azureNetwork network.Subnets
	// encrypter is initialized in runtime
	encrypter    Encrypter
	ignitionPath string
	OIDC         setting.OIDC
	ssoPublicKey string
}

func New(config Config) (*CloudConfig, error) {
	if config.CertsSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CertsSearcher must not be empty", config)
	}
	if config.IgnitionPath == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.IgnitionPath must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.RandomkeysSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.RandomkeysSearcher must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureConfig.%s", config, err)
	}

	encrypter, err := NewEncrypter()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	c := &CloudConfig{
		certsSearcher:      config.CertsSearcher,
		logger:             config.Logger,
		randomkeysSearcher: config.RandomkeysSearcher,

		azure:        config.Azure,
		azureConfig:  config.AzureConfig,
		azureNetwork: config.AzureNetwork,
		encrypter:    encrypter,
		ignitionPath: config.IgnitionPath,
		OIDC:         config.OIDC,
		ssoPublicKey: config.SSOPublicKey,
	}

	return c, nil
}

// GetEncryptionKey returns hex of the key, which is used for certificates encryption.
func (c CloudConfig) GetEncryptionKey() string {
	return hex.EncodeToString(c.encrypter.key[aes.BlockSize:])
}

// GetInitialVector returns hex of the initial vector, which is used in certificate encryption.
func (c CloudConfig) GetInitialVector() string {
	return hex.EncodeToString(c.encrypter.key[:aes.BlockSize])
}

// NewMasterCloudConfig generates a new master cloudconfig and returns it as a
// base64 encoded string.
func (c CloudConfig) NewMasterCloudConfig(customObject providerv1alpha1.AzureConfig) (string, error) {
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
		params = k8scloudconfig.DefaultParams()
		params.APIServerEncryptionKey = apiserverEncryptionKey
		params.Cluster = customObject.Spec.Cluster
		params.DisableCalico = true
		params.DisableIngressControllerService = true
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
			Azure:         c.azure,
			AzureConfig:   c.azureConfig,
			CalicoCIDR:    c.azureNetwork.Calico.String(),
			CertsSearcher: c.certsSearcher,
			CustomObject:  customObject,
			Encrypter:     c.encrypter,
		}
		params.ExtraManifests = []string{
			"calico-azure.yaml",
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

// NewWorkerCloudConfig generates a new worker cloudconfig and returns it as a
// base64 encoded string.
func (c CloudConfig) NewWorkerCloudConfig(customObject providerv1alpha1.AzureConfig) (string, error) {
	var err error

	// NOTE in Azure we disable Calico right now. This is due to a transitioning
	// phase. The k8scloudconfig templates require certain calico valus to be set
	// nonetheless. So we set them here. Later when the Calico setup is
	// straightened out we can improve the handling here.
	customObject.Spec.Cluster.Calico.Subnet = c.azureNetwork.Calico.IP.String()
	customObject.Spec.Cluster.Calico.CIDR, _ = c.azureNetwork.Calico.Mask.Size()

	var params k8scloudconfig.Params
	{
		params = k8scloudconfig.DefaultParams()

		params.Cluster = customObject.Spec.Cluster
		params.Hyperkube = k8scloudconfig.Hyperkube{
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
		params.Extension = &workerExtension{
			Azure:         c.azure,
			AzureConfig:   c.azureConfig,
			CertsSearcher: c.certsSearcher,
			CustomObject:  customObject,
			Encrypter:     c.encrypter,
		}
		params.SSOPublicKey = c.ssoPublicKey

		ignitionPath := k8scloudconfig.GetIgnitionPath(c.ignitionPath)
		params.Files, err = k8scloudconfig.RenderFiles(ignitionPath, params)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	return newCloudConfig(k8scloudconfig.WorkerTemplate, params)
}

// certFiles returns list of certificates paths for particular role.
// It is used for rendering cert-decrypter systemd unit.
func (c CloudConfig) certFiles(customObject providerv1alpha1.AzureConfig, role string) (certs.Files, error) {
	var certFiles certs.Files

	clusterCerts, err := c.certsSearcher.SearchCluster(key.ClusterID(customObject))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	switch role {
	case key.PrefixMaster():
		certFiles = certs.NewFilesClusterMaster(clusterCerts)
	case key.PrefixWorker():
		certFiles = certs.NewFilesClusterWorker(clusterCerts)
	default:
		return nil, microerror.Maskf(err, "unknown role %#q", role)
	}

	return certFiles, nil
}

func (c CloudConfig) getEncryptionkey(customObject providerv1alpha1.AzureConfig) (string, error) {
	cluster, err := c.randomkeysSearcher.SearchCluster(key.ClusterID(customObject))
	if err != nil {
		return "", microerror.Mask(err)
	}
	return string(cluster.APIServerEncryptionKey), nil
}

func newCloudConfig(template string, params k8scloudconfig.Params) (string, error) {
	c := k8scloudconfig.DefaultCloudConfigConfig()
	c.Params = params
	c.Template = template

	cloudConfig, err := k8scloudconfig.NewCloudConfig(c)
	if err != nil {
		return "", microerror.Mask(err)
	}
	err = cloudConfig.ExecuteTemplate()
	if err != nil {
		return "", microerror.Mask(err)
	}

	return cloudConfig.String(), nil
}

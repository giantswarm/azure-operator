package cloudconfig

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_3_4_0"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/randomkeys"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

type Config struct {
	CertsSearcher      certs.Interface
	Logger             micrologger.Logger
	RandomkeysSearcher randomkeys.Interface

	Azure setting.Azure
	// TODO(pk) remove as soon as we sort calico in Azure provider.
	AzureConfig  client.AzureClientSetConfig
	OIDC         setting.OIDC
	SSOPublicKey string
}

type CloudConfig struct {
	certsSearcher      certs.Interface
	logger             micrologger.Logger
	randomkeysSearcher randomkeys.Interface

	azure        setting.Azure
	azureConfig  client.AzureClientSetConfig
	OIDC         setting.OIDC
	ssoPublicKey string
}

func New(config Config) (*CloudConfig, error) {
	if config.CertsSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CertsSearcher must not be empty", config)
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

	c := &CloudConfig{
		certsSearcher:      config.CertsSearcher,
		logger:             config.Logger,
		randomkeysSearcher: config.RandomkeysSearcher,

		azure:        config.Azure,
		azureConfig:  config.AzureConfig,
		OIDC:         config.OIDC,
		ssoPublicKey: config.SSOPublicKey,
	}

	return c, nil
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

	params := k8scloudconfig.Params{
		APIServerEncryptionKey:   apiserverEncryptionKey,
		Cluster:                  customObject.Spec.Cluster,
		DisableCalico:            true,
		DisableIngressController: true,
		Hyperkube: k8scloudconfig.Hyperkube{
			Apiserver: k8scloudconfig.HyperkubeApiserver{
				Pod: k8scloudconfig.HyperkubePod{
					HyperkubePodHostExtraMounts: []k8scloudconfig.HyperkubePodHostMount{
						k8scloudconfig.HyperkubePodHostMount{
							Name:     "k8s-config",
							Path:     "/etc/kubernetes/config/",
							ReadOnly: true,
						},
						k8scloudconfig.HyperkubePodHostMount{
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
						k8scloudconfig.HyperkubePodHostMount{
							Name:     "identity-settings",
							Path:     "/var/lib/waagent/",
							ReadOnly: true,
						},
					},
					CommandExtraArgs: []string{
						"--cloud-config=/etc/kubernetes/config/azure.yaml",
						"--allocate-node-cidrs=true",
						"--cluster-cidr=" + key.VnetCalicoSubnetCIDR(customObject),
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
		},
		Extension: &masterExtension{
			Azure:         c.azure,
			AzureConfig:   c.azureConfig,
			CertsSearcher: c.certsSearcher,
			CustomObject:  customObject,
		},
		ExtraManifests: []string{
			"calico-azure.yaml",
		},
		SSOPublicKey: c.ssoPublicKey,
	}

	return newCloudConfig(k8scloudconfig.MasterTemplate, params)
}

// NewWorkerCloudConfig generates a new worker cloudconfig and returns it as a
// base64 encoded string.
func (c CloudConfig) NewWorkerCloudConfig(customObject providerv1alpha1.AzureConfig) (string, error) {
	params := k8scloudconfig.Params{
		Cluster: customObject.Spec.Cluster,
		Hyperkube: k8scloudconfig.Hyperkube{
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
		},
		Extension: &workerExtension{
			Azure:         c.azure,
			AzureConfig:   c.azureConfig,
			CertsSearcher: c.certsSearcher,
			CustomObject:  customObject,
		},
		SSOPublicKey: c.ssoPublicKey,
	}

	return newCloudConfig(k8scloudconfig.WorkerTemplate, params)
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

	compressed, err := gzipBase64(cloudConfig.String())
	if err != nil {
		return "", microerror.Mask(err)
	}

	// cloud-config is compressed so we fit the tight 85kB limit of
	// customData parameter.
	//
	// "Custom data in OSProfile must be in Base64 encoding and with
	// a maximum length of 87380 characters."
	//
	//  87380 / 1024 = 85
	customData := fmt.Sprintf(`#!/bin/bash
D="/root/cloudinit"
mkdir -m 700 -p ${D}
echo -n "%s" | base64 -d | gzip -d -c > ${D}/cc
coreos-cloudinit --from-file=${D}/cc`, compressed)

	customData = base64.StdEncoding.EncodeToString([]byte(customData))
	return customData, nil
}

func gzipBase64(s string) (string, error) {
	buf := new(bytes.Buffer)

	w, err := gzip.NewWriterLevel(buf, gzip.BestCompression)
	if err != nil {
		return "", microerror.Mask(err)
	}
	_, err = io.WriteString(w, s)
	if err != nil {
		return "", microerror.Mask(err)
	}
	err = w.Close()
	if err != nil {
		return "", microerror.Mask(err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

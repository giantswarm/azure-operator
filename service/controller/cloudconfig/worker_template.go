package cloudconfig

import (
	"context"
	"encoding/base64"
	"fmt"

	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v17/pkg/template"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v8/pkg/label"
	"github.com/giantswarm/azure-operator/v8/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v8/service/controller/key"
	"github.com/giantswarm/azure-operator/v8/service/controller/templates/ignition"
)

// NewWorkerCloudConfig generates a new worker cloudconfig and returns it as a
// base64 encoded string.
func (c CloudConfig) NewWorkerTemplate(ctx context.Context, data IgnitionTemplateData, encrypter encrypter.Interface) (string, error) {
	var err error

	var params k8scloudconfig.Params
	{
		be := baseExtension{
			azure:                        c.azure,
			azureClientCredentialsConfig: c.azureClientCredentials,
			certFiles:                    data.WorkerCertFiles,
			customObject:                 data.CustomObject,
			encrypter:                    encrypter,
			azureMachinePool:             data.AzureMachinePool,
			subscriptionID:               c.subscriptionID,
			vnetCIDR:                     data.CustomObject.Spec.Azure.VirtualNetwork.CIDR,
		}

		params = k8scloudconfig.Params{}
		params.Cluster = data.CustomObject.Spec.Cluster
		// Add node label for cgroups version
		params.Cluster.Kubernetes.Kubelet.Labels = fmt.Sprintf("%s,%s=%s", params.Cluster.Kubernetes.Kubelet.Labels, label.CGroupVersion, key.CGroupVersion(data.MachinePool))
		params.CalicoPolicyOnly = true
		params.Kubernetes = k8scloudconfig.Kubernetes{
			Kubelet: k8scloudconfig.KubernetesDockerOptions{
				RunExtraArgs: []string{
					"-v /var/lib/waagent:/var/lib/waagent:ro",
				},
			},
		}
		params.EnableCronJobTimeZone = true
		params.Extension = &workerExtension{
			baseExtension: be,
		}
		params.ExternalCloudControllerManager = true
		params.Debug = k8scloudconfig.Debug{
			Enabled:    c.ignition.Debug,
			LogsPrefix: c.ignition.LogsPrefix,
			LogsToken:  c.ignition.LogsToken,
		}
		params.DockerhubToken = c.dockerhubToken
		params.Images = data.Images
		params.RegistryMirrors = c.registryMirrors
		params.Versions = data.Versions
		params.SSOPublicKey = c.ssoPublicKey

		forceCGroupsV1 := false
		if data.MachinePool.Annotations != nil {
			_, forceCGroupsV1 = data.MachinePool.Annotations[annotation.NodeForceCGroupsV1]
		}
		params.ForceCGroupsV1 = forceCGroupsV1

		ignitionPath := k8scloudconfig.GetIgnitionPath(c.ignition.Path)
		params.Files, err = k8scloudconfig.RenderFiles(ignitionPath, params)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	c.logger.Debugf(ctx, "rendering cloudconfig with kubelet labels %v", params.Cluster.Kubernetes.Kubelet.Labels)

	return newCloudConfig(k8scloudconfig.WorkerTemplate, params)
}

type workerExtension struct {
	baseExtension
}

// Files allows files to be injected into the master cloudconfig.
func (we *workerExtension) Files() ([]k8scloudconfig.FileAsset, error) {
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

	data := we.templateData(we.certFiles)

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
	for _, f := range we.certFiles {
		encryptedData, err := we.encrypter.Encrypt(f.Data)
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
func (we *workerExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
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

	data := we.templateData(we.certFiles)

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

// VerbatimSections allows sections to be embedded in the worker cloudconfig.
func (we *workerExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}

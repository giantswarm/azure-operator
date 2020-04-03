package blobobject

import (
	"context"

	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8scloudconfig/v_6_0_0"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/azure-operator/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var clusterCerts certs.Cluster
	{
		clusterCerts, err = r.certsSearcher.SearchCluster(key.ClusterID(cr))
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	storageAccountName := key.StorageAccountName(cr)
	containerName := key.BlobContainerName()
	certificateEncryptionSecretName := key.CertificateEncryptionSecretName(cr)

	encrypter, err := r.toEncrypterObject(ctx, certificateEncryptionSecretName)
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "encryptionkey resource is not ready") // nolint: errcheck
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")                  // nolint: errcheck
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	var ignitionTemplateData cloudconfig.IgnitionTemplateData
	{
		versions, err := v_6_0_0.ExtractComponentVersions(cc.Release.Components)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		defaultVersions := key.DefaultVersions()
		versions.Kubectl = defaultVersions.Kubectl
		versions.KubernetesAPIHealthz = defaultVersions.KubernetesAPIHealthz
		versions.KubernetesNetworkSetupDocker = defaultVersions.KubernetesNetworkSetupDocker
		images := v_6_0_0.BuildImages(r.registryDomain, versions)

		ignitionTemplateData = cloudconfig.IgnitionTemplateData{
			CustomObject: cr,
			ClusterCerts: clusterCerts,
			Images:       images,
		}
	}

	output := []ContainerObjectState{}
	{
		b, err := cc.CloudConfig.NewMasterTemplate(ctx, ignitionTemplateData, encrypter)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		k := key.BlobName(cr, prefixMaster)
		containerObjectState := ContainerObjectState{
			Body:               b,
			ContainerName:      containerName,
			Key:                k,
			StorageAccountName: storageAccountName,
		}

		output = append(output, containerObjectState)
	}

	{
		b, err := cc.CloudConfig.NewWorkerTemplate(ctx, ignitionTemplateData, encrypter)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		k := key.BlobName(cr, prefixWorker)
		containerObjectState := ContainerObjectState{
			Body:               b,
			ContainerName:      containerName,
			Key:                k,
			StorageAccountName: storageAccountName,
		}

		output = append(output, containerObjectState)
	}

	return output, nil
}

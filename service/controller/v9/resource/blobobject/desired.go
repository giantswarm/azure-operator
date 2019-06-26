package blobobject

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/azure-operator/service/controller/v8/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v8/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	ctlCtx, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clusterCerts, err := r.certsSearcher.SearchCluster(key.ClusterID(customObject))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	storageAccountName := key.StorageAccountName(customObject)
	containerName := key.BlobContainerName()
	certificateEncryptionSecretName := key.CertificateEncryptionSecretName(customObject)

	encrypter, err := r.toEncrypterObject(ctx, certificateEncryptionSecretName)
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "encryptionkey resource is not ready")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	output := []ContainerObjectState{}

	{
		b, err := ctlCtx.CloudConfig.NewMasterCloudConfig(customObject, clusterCerts, encrypter)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		k := key.BlobName(customObject, prefixMaster)
		containerObjectState := ContainerObjectState{
			Body:               b,
			ContainerName:      containerName,
			Key:                k,
			StorageAccountName: storageAccountName,
		}

		output = append(output, containerObjectState)
	}

	{
		b, err := ctlCtx.CloudConfig.NewWorkerCloudConfig(customObject, clusterCerts, encrypter)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		k := key.BlobName(customObject, prefixWorker)
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

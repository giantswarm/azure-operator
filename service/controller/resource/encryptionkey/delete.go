package encryptionkey

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting encryptionkey secret upon delete event")

		err = r.k8sClient.CoreV1().Secrets(key.CertificateEncryptionNamespace).Delete(ctx, key.CertificateEncryptionSecretName(&cr), metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "encryptionkey secret already deleted")
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "deleted encryptionkey secret upon delete event")
		}
	}

	return nil
}

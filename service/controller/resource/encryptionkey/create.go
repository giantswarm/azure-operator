package encryptionkey

import (
	"context"
	"crypto/aes"
	"crypto/rand"
	"io"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var secret *corev1.Secret
	var encKey, encIV []byte

	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	encKey = make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, encKey); err != nil {
		return microerror.Mask(err)
	}

	encIV = make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, encIV); err != nil {
		return microerror.Mask(err)
	}

	secret = &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.CertificateEncryptionSecretName(&cr),
			Namespace: key.CertificateEncryptionNamespace,
			Labels: map[string]string{
				key.LabelCluster:      key.ClusterID(&cr),
				key.LabelManagedBy:    r.projectName,
				key.LabelOrganization: key.ClusterCustomer(cr),
			},
		},
		Data: map[string][]byte{
			key.CertificateEncryptionKeyName: encKey,
			key.CertificateEncryptionIVName:  encIV,
		},
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "creating encryptionkey secret")

	_, err = r.k8sClient.CoreV1().Secrets(key.CertificateEncryptionNamespace).Create(ctx, secret, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "creating encryptionkey: already created")
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

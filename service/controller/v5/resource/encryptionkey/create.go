package encryptionkey

import (
	"context"
	"crypto/aes"
	"crypto/rand"
	"io"

	"github.com/giantswarm/microerror"
	apiv1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/v5/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var secret *apiv1.Secret
	var encKey, encIV []byte

	customObject, err := key.ToCustomObject(obj)
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

	secret = &apiv1.Secret{
		Type: apiv1.SecretTypeOpaque,
		ObjectMeta: apismetav1.ObjectMeta{
			Name:      key.CertificateEncryptionName(customObject),
			Namespace: key.CertificateEncryptionNamespace,
			Labels: map[string]string{
				key.LegacyLabelCluster: key.ClusterID(customObject),
				key.LabelCustomer:      key.ClusterCustomer(customObject),
				key.LabelCluster:       key.ClusterID(customObject),
				key.LabelOrganization:  key.ClusterCustomer(customObject),
				key.LabelVersionBundle: key.VersionBundleVersion(customObject),
			},
		},
		Data: map[string][]byte{
			key.CertificateEncryptionKeyName: encKey,
			key.CertificateEncryptionIVName:  encIV,
		},
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "creating encryptionkey secret")

	_, err = r.k8sClient.CoreV1().Secrets(key.CertificateEncryptionNamespace).Create(secret)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "created encryptionkey secret")

	return nil
}

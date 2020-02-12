// +build k8srequired

package storage

import (
	"context"
	"testing"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Storage(t *testing.T) {
	err := storage.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

type Config struct {
	Logger   micrologger.Logger
	Provider *Provider
}

type Storage struct {
	logger   micrologger.Logger
	provider *Provider
}

func New(config Config) (*Storage, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Provider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}

	s := &Storage{
		logger:   config.Logger,
		provider: config.Provider,
	}

	return s, nil
}

func (s *Storage) Test(ctx context.Context) error {
	{
		pvcName := "e2e-pvc"
		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: "default",
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceName(v1.ResourceStorage): resource.MustParse("1Gi"),
					},
				},
			},
		}
		s.logger.LogCtx(ctx, "level", "debug", "message", "creating pvc '%s'", pvcName)
		_, err := s.provider.k8sClient.CoreV1().PersistentVolumeClaims("default").Create(pvc)
		if err != nil {
			return microerror.Mask(err)
		}

		o := func() error {
			pvc, err := s.provider.k8sClient.CoreV1().PersistentVolumeClaims("default").Get(pvcName, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			if pvc.Status.Phase != v1.ClaimBound {
				return microerror.Maskf(executionFailedError, "PVC '%s' is not bound", pvcName)
			}

			return nil
		}

		b := backoff.NewConstant(backoff.ShortMaxWait, backoff.ShortMaxInterval)
		n := func(err error, delay time.Duration) {
			s.logger.Log("level", "debug", "message", err.Error())
		}

		err = backoff.RetryNotify(o, b, n)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

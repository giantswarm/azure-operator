package locker

import (
	"context"
	"time"

	"github.com/giantswarm/kubelock/v2"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/azure-operator/v4/pkg/project"
)

const (
	lockName          = "ipam"
	lockNamespaceName = "giantswarm"
)

var (
	lockOwner = project.Name() + "@" + project.Version()
	lockTTL   = 30 * time.Second
)

type KubeLockLockerConfig struct {
	Logger     micrologger.Logger
	RestConfig *rest.Config
}

type KubeLockLocker struct {
	logger micrologger.Logger

	kubelock kubelock.Interface
}

func NewKubeLockLocker(config KubeLockLockerConfig) (*KubeLockLocker, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.RestConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.RestConfig must not be empty", config)
	}

	var err error

	var dynClient dynamic.Interface
	{
		dynClient, err = dynamic.NewForConfig(config.RestConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

	}

	var k kubelock.Interface
	{
		c := kubelock.Config{
			DynClient: dynClient,
			GVR: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "namespaces",
			},
		}
		k, err = kubelock.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	d := &KubeLockLocker{
		logger: config.Logger,

		kubelock: k,
	}

	return d, nil
}

func (d KubeLockLocker) Lock(ctx context.Context) error {
	err := d.kubelock.Lock(lockName).Acquire(ctx, lockNamespaceName, kubelock.AcquireOptions{
		Owner: lockOwner,
		TTL:   lockTTL,
	})
	if kubelock.IsAlreadyExists(err) {
		return microerror.Maskf(alreadyExistsError, err.Error())
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (d KubeLockLocker) Unlock(ctx context.Context) error {
	err := d.kubelock.Lock(lockName).Release(ctx, lockNamespaceName, kubelock.ReleaseOptions{
		Owner: lockOwner,
	})
	if kubelock.IsNotFound(err) {
		return microerror.Maskf(notFoundError, err.Error())
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

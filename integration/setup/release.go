package setup

import (
	"context"
	"time"

	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/crd"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/pkg/project"
)

const (
	ReleaseName = "v1.0.0"
)

func createGSReleaseContainingOperatorVersion(ctx context.Context, config Config) (*releasev1alpha1.Release, error) {
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring Release CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("release.giantswarm.io", "Release"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return &releasev1alpha1.Release{}, microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured Release CRD exists")
	}

	var release *releasev1alpha1.Release
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring Release exists", "release", ReleaseName)
		release = &releasev1alpha1.Release{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ReleaseName,
				Namespace: "default",
				Labels: map[string]string{
					"giantswarm.io/managed-by": "release-operator",
					"giantswarm.io/provider":   "azure",
				},
			},
			Spec: releasev1alpha1.ReleaseSpec{
				Apps: []releasev1alpha1.ReleaseSpecApp{},
				Components: []releasev1alpha1.ReleaseSpecComponent{
					{
						Name:    project.Name(),
						Version: project.Version(),
					},
					{
						Name:    "cluster-operator",
						Version: "0.23.12",
					},
					{
						Name:    "cert-operator",
						Version: "0.1.0",
					},
					{
						Name:    "app-operator",
						Version: "1.0.0",
					},
					{
						Name:    "calico",
						Version: "3.15.1",
					},
					{
						Name:    "containerlinux",
						Version: "2512.2.1",
					},
					{
						Name:    "coredns",
						Version: "1.6.5",
					},
					{
						Name:    "etcd",
						Version: "3.4.9",
					},
					{
						Name:    "kubernetes",
						Version: "1.17.9",
					},
				},
				Date:  &metav1.Time{Time: time.Unix(10, 0)},
				State: "active",
			},
		}
		_, err := config.K8sClients.G8sClient().ReleaseV1alpha1().Releases().Create(release)
		if err != nil {
			return &releasev1alpha1.Release{}, microerror.Mask(err)
		}
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured Release exists", "release", ReleaseName)
	}

	return release, nil
}

// +build k8srequired

package setup

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/giantswarm/microerror"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Setup(m *testing.M, config Config) {
	ctx := context.Background()

	var v int
	var err error

	err = installResources(ctx, config)
	if err != nil {
		config.Logger.LogCtx(ctx, "level", "error", "message", "failed to install resources", "stack", fmt.Sprintf("%#v", err))
		v = 1
	}

	if v == 0 {
		v = m.Run()
	}

	os.Exit(v)
}

func installResources(ctx context.Context, config Config) error {
	var err error

	{
		err = config.K8s.EnsureNamespaceCreated(ctx, "giantswarm")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		replicas := int32(1)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cnr-server",
				Namespace: metav1.NamespaceDefault,
				Labels: map[string]string{
					"app": "cnr-server",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "cnr-server",
					},
				},
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cnr-server",
						Labels: map[string]string{
							"app": "cnr-server",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "cnr-server",
								Image:           "quay.io/giantswarm/cnr-server:latest",
								ImagePullPolicy: corev1.PullIfNotPresent,
							},
						},
					},
				},
			},
		}

		_, err := config.CPK8sClients.K8sClient().AppsV1().Deployments(metav1.NamespaceDefault).Create(deployment)
		if apierrors.IsAlreadyExists(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		service := &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "service",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cnr-server",
				Namespace: metav1.NamespaceDefault,
				Labels: map[string]string{
					"app": "cnr-server",
				},
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{
						Name:     "cnr-server",
						Port:     int32(5000),
						Protocol: "TCP",
					},
				},
				Selector: map[string]string{
					"app": "cnr-server",
				},
			},
		}

		_, err := config.CPK8sClients.K8sClient().CoreV1().Services(metav1.NamespaceDefault).Create(service)
		if apierrors.IsAlreadyExists(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

package setup

import (
	"context"

	"github.com/giantswarm/e2e-harness/v2/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/microerror"
	v12 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/e2e/env"
	"github.com/giantswarm/azure-operator/v4/e2e/key"
)

func installVault(ctx context.Context, config Config) error {
	// Create RBAC rule to allow Vault to use the kubernetes auth backend.
	{
		clusterRoleBinding := v12.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "jwt-reviewer",
			},
			Subjects: []v12.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "default",
					Namespace: "default",
				},
			},
			RoleRef: v12.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
		}

		_, err := config.K8sClients.K8sClient().RbacV1().ClusterRoleBindings().Create(ctx, &clusterRoleBinding, metav1.CreateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Install Vault chart.
	{
		c := chartvalues.E2ESetupVaultConfig{
			Vault: chartvalues.E2ESetupVaultConfigVault{
				Token: env.VaultToken(),
			},
		}

		values, err := chartvalues.NewE2ESetupVault(c)
		if err != nil {
			return microerror.Mask(err)
		}

		err = config.Release.Install(ctx, key.VaultReleaseName(), release.NewStableVersion(), values, config.Release.Condition().PodExists(ctx, "default", "app=vault"))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

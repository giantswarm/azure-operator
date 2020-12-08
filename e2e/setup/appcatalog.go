package setup

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type catalogDef struct {
	ConfigmapName      string
	ConfigmapNamespace string
	Name               string
	SecretName         string
	SecretNamespace    string
	Type               string
	URL                string
	Visibility         string
}

func ensureAppCatalogs(ctx context.Context, config Config, version string) error {

	catalogs := []catalogDef{
		{
			ConfigmapName:      "default-catalog",
			ConfigmapNamespace: "giantswarm",
			Name:               "default",
			Type:               "stable",
			URL:                "https://giantswarm.github.io/default-catalog/",
			Visibility:         "internal",
		},
		{
			ConfigmapName:      DraughtsmanConfigMapName,
			ConfigmapNamespace: DraughtsmanNamespace,
			Name:               "control-plane-catalog",
			SecretName:         DraughtsmanSecretName,
			SecretNamespace:    DraughtsmanNamespace,
			Type:               "stable",
			URL:                "https://giantswarm.github.io/control-plane-catalog/",
			Visibility:         "internal",
		},
	}

	for _, def := range catalogs {
		// If a configmap is set for this catalog, ensure it exists.
		{
			err := ensureConfigmapForCatalog(ctx, config, def)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		// Create the AppCatalog CR.
		config.Logger.Debugf(ctx, "ensuring app catalog %s", def.Name)
		{
			catalog := v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: def.Name,
					Labels: map[string]string{
						"app-operator.giantswarm.io/version":           version,
						"application.giantswarm.io/catalog-type":       def.Type,
						"application.giantswarm.io/catalog-visibility": def.Visibility,
					},
				},
				Spec: v1alpha1.AppCatalogSpec{
					Title:       fmt.Sprintf("Giant Swarm %s Catalog", def.Name),
					Description: "",
					Config: v1alpha1.AppCatalogSpecConfig{
						ConfigMap: v1alpha1.AppCatalogSpecConfigConfigMap{
							Name:      def.ConfigmapName,
							Namespace: def.ConfigmapNamespace,
						},
						Secret: v1alpha1.AppCatalogSpecConfigSecret{
							Name:      def.SecretName,
							Namespace: def.SecretNamespace,
						},
					},
					LogoURL: "/images/repo_icons/giantswarm.png",
					Storage: v1alpha1.AppCatalogSpecStorage{
						Type: "helm",
						URL:  def.URL,
					},
				},
			}

			_, err := config.K8sClients.G8sClient().ApplicationV1alpha1().AppCatalogs().Create(ctx, &catalog, metav1.CreateOptions{})
			if err != nil {
				return microerror.Mask(err)
			}
		}
		config.Logger.Debugf(ctx, "ensured app catalog %s", def.Name)
	}

	return nil
}

func ensureConfigmapForCatalog(ctx context.Context, config Config, def catalogDef) error {
	if def.ConfigmapNamespace == "" || def.ConfigmapName == "" {
		config.Logger.Debugf(ctx, "No configmap defined for catalog %s", def.Name)
		return nil
	}

	// Check if the configmap already exists.
	_, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(def.ConfigmapNamespace).Get(ctx, def.ConfigmapName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// Config Map has to be created.
		config.Logger.Debugf(ctx, "Creating configmap for catalog %s", def.Name)
		cm := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      def.ConfigmapName,
				Namespace: def.ConfigmapNamespace,
			},
			Data: map[string]string{
				"values": `    t: t
    NetExporter:
      DNSCheck:
        TCP:
          Disabled: false
      Hosts: giantswarm.io.,kubernetes.default.svc.cluster.local.
      NTPServers: 0.flatcar.pool.ntp.org,1.flatcar.pool.ntp.org
    exporter:
      namespace: kube-system
    externalDNSIP: 8.8.8.8
    image:
      registry: quay.io
    minReplicas: 3
    provider: azure
    tiller:
      namespace: giantswarm
`,
			},
		}

		_, err := config.K8sClients.K8sClient().CoreV1().ConfigMaps(def.ConfigmapNamespace).Create(ctx, &cm, metav1.CreateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.Debugf(ctx, "Created configmap for catalog %s", def.Name)
		return nil

	} else if err != nil {
		return microerror.Mask(err)
	}

	// Configmap exists.
	config.Logger.Debugf(ctx, "Configmap for catalog %s already exists", def.Name)

	return nil
}

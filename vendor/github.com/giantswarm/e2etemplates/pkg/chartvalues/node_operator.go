package chartvalues

import (
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/e2etemplates/internal/render"
)

type NodeOperatorConfig struct {
	Namespace          string
	RegistryPullSecret string
}

func NewNodeOperator(config NodeOperatorConfig) (string, error) {
	if config.Namespace == "" {
		config.Namespace = namespaceGiantswarm
	}
	if config.RegistryPullSecret == "" {
		return "", microerror.Maskf(invalidConfigError, "%T.RegistryPullSecret must not be empty", config)
	}

	values, err := render.Render(nodeOperatorTemplate, config)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return values, nil
}

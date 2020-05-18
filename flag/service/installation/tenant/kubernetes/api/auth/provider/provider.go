package provider

import (
	"github.com/giantswarm/azure-operator/v4/flag/service/installation/tenant/kubernetes/api/auth/provider/oidc"
)

type Provider struct {
	OIDC oidc.OIDC
}

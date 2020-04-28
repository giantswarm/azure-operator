package auth

import (
	"github.com/giantswarm/azure-operator/v3/flag/service/installation/tenant/kubernetes/api/auth/provider"
)

type Auth struct {
	Provider provider.Provider
}

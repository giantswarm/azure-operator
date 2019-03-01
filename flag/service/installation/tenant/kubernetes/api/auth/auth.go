package auth

import (
	"github.com/giantswarm/azure-operator/flag/service/installation/tenant/kubernetes/api/auth/provider"
)

type Auth struct {
	Provider provider.Provider
}

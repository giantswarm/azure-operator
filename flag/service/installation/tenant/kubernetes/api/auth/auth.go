package auth

import (
	"github.com/giantswarm/azure-operator/v4/flag/service/installation/tenant/kubernetes/api/auth/provider"
)

type Auth struct {
	Provider provider.Provider
}

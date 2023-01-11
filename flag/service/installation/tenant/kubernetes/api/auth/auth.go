package auth

import (
	"github.com/giantswarm/azure-operator/v7/flag/service/installation/tenant/kubernetes/api/auth/provider"
)

type Auth struct {
	Provider provider.Provider
}

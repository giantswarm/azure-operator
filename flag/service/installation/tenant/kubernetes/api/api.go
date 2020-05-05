package api

import (
	"github.com/giantswarm/azure-operator/v3/flag/service/installation/tenant/kubernetes/api/auth"
)

type API struct {
	Auth auth.Auth
}

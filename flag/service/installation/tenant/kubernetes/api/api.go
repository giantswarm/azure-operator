package api

import (
	"github.com/giantswarm/azure-operator/v6/flag/service/installation/tenant/kubernetes/api/auth"
)

type API struct {
	Auth auth.Auth
}

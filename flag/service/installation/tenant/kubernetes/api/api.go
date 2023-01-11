package api

import (
	"github.com/giantswarm/azure-operator/v7/flag/service/installation/tenant/kubernetes/api/auth"
)

type API struct {
	Auth auth.Auth
}

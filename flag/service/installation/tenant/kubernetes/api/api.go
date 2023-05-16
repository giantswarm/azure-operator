package api

import (
	"github.com/giantswarm/azure-operator/v8/flag/service/installation/tenant/kubernetes/api/auth"
)

type API struct {
	Auth auth.Auth
}

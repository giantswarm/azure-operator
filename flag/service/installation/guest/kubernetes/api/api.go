package api

import (
	"github.com/giantswarm/azure-operator/flag/service/installation/guest/kubernetes/api/auth"
)

type API struct {
	Auth auth.Auth
}

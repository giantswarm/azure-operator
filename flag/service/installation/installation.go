package installation

import (
	"github.com/giantswarm/azure-operator/flag/service/installation/guest"
)

type Installation struct {
	Name  string
	Guest guest.Guest
}

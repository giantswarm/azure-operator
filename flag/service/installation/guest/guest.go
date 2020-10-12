package guest

import (
	"github.com/giantswarm/azure-operator/v5/flag/service/installation/guest/ipam"
)

type Guest struct {
	IPAM ipam.IPAM
}

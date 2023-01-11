package guest

import (
	"github.com/giantswarm/azure-operator/v7/flag/service/installation/guest/ipam"
)

type Guest struct {
	IPAM ipam.IPAM
}

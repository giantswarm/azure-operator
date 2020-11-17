package conditions

import (
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

type CR interface {
	key.LabelsGetter
	capiconditions.Setter
}

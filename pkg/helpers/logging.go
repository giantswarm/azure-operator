package helpers

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
)

func LogMachinePoolObject(ctx context.Context, logger micrologger.Logger, machinePool capiexp.MachinePool) error {
	machinePoolJson, err := json.Marshal(machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	logger.Debugf(ctx, "MachinePool CR: %s", machinePoolJson)
	return nil
}

package state

import (
	"context"

	"github.com/giantswarm/microerror"
)

func (m Machine) Execute(ctx context.Context, obj interface{}, currentState State) (State, error) {
	transitionFunc, exists := m[currentState]
	if !exists {
		return "", microerror.Maskf(executionFailedError, "State: %q is not configured in this state machine", currentState)
	}

	newState, err := transitionFunc(ctx, obj, currentState)
	if err != nil {
		return "", microerror.Mask(err)
	}

	_, exists = m[newState]
	if !exists {
		return "", microerror.Maskf(executionFailedError, "State transition returned new unknown state: %q. Input state: %q", newState, currentState)
	}

	return newState, nil
}

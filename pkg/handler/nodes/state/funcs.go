package state

import (
	"context"

	"github.com/giantswarm/microerror"
)

func (m Machine) Execute(ctx context.Context, obj interface{}, currentState State) (State, error) {
	transitionFunc, exists := m.Transitions[currentState]
	if !exists {
		return "", microerror.Maskf(unknownStateError, "State: %q is not configured in this state machine", currentState)
	}

	newState, err := transitionFunc(ctx, obj, currentState)
	if err != nil {
		return newState, microerror.Mask(err)
	}

	_, exists = m.Transitions[newState]
	if !exists {
		return newState, microerror.Maskf(executionFailedError, "State transition returned new unknown state: %q. Input state: %q", newState, currentState)
	}

	m.Logger.LogCtx(ctx, "resource", m.ResourceName, "message", "state changed", "oldState", currentState, "newState", newState)
	return newState, nil
}

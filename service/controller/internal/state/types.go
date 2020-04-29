package state

import (
	"context"

	"github.com/giantswarm/micrologger"
)

// Machine is a simple type to hold state machine configuration.
type Machine struct {
	Logger       micrologger.Logger
	ResourceName string
	Transitions  TransitionMap
}

type State string
type TransitionMap map[State]TransitionFunc

// TransitionFunc defines state transition function signature.
type TransitionFunc func(ctx context.Context, obj interface{}, currentState State) (State, error)

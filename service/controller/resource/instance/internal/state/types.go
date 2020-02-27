package state

import "context"

// Machine is a simple type to hold state machine configuration.
type Machine map[State]TransitionFunc

type State string

// TransitionFunc defines state transition function signature.
type TransitionFunc func(ctx context.Context, obj interface{}, currentState State) (State, error)

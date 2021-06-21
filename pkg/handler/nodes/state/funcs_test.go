package state

import (
	"context"
	"strconv"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/google/go-cmp/cmp"
)

const (
	OpenState   = "open"
	ClosedState = "closed"
)

func Test_StateMachine(t *testing.T) {
	logger, err := micrologger.New(micrologger.Config{})
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name             string
		machine          Machine
		currentState     State
		expectedNewState State
		errorMatcher     func(error) bool
	}{
		{
			name: "case 0: simple state transition",
			machine: Machine{
				Logger:       logger,
				ResourceName: "",
				Transitions: TransitionMap{
					OpenState:   func(ctx context.Context, obj interface{}, currentState State) (State, error) { return ClosedState, nil },
					ClosedState: func(ctx context.Context, obj interface{}, currentState State) (State, error) { return OpenState, nil },
				},
			},
			currentState:     OpenState,
			expectedNewState: ClosedState,
			errorMatcher:     nil,
		},
		{
			name: "case 1: unknown start state",
			machine: Machine{
				Logger:       logger,
				ResourceName: "",
				Transitions: TransitionMap{
					OpenState:   func(ctx context.Context, obj interface{}, currentState State) (State, error) { return ClosedState, nil },
					ClosedState: func(ctx context.Context, obj interface{}, currentState State) (State, error) { return OpenState, nil },
				},
			},
			currentState:     "half-way",
			expectedNewState: "",
			errorMatcher:     IsUnkownStateError,
		},
		{
			name: "case 2: unknown new state",
			machine: Machine{
				Logger:       logger,
				ResourceName: "",
				Transitions: TransitionMap{
					OpenState:   func(ctx context.Context, obj interface{}, currentState State) (State, error) { return "half-way", nil },
					ClosedState: func(ctx context.Context, obj interface{}, currentState State) (State, error) { return OpenState, nil },
				},
			},
			currentState:     OpenState,
			expectedNewState: "half-way",
			errorMatcher:     IsExecutionFailedError,
		},
		{
			name:             "case 3: execute state on empty state machine",
			machine:          Machine{},
			currentState:     "start",
			expectedNewState: "",
			errorMatcher:     IsUnkownStateError,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			newState, err := tc.machine.Execute(context.Background(), nil, tc.currentState)

			switch {
			case err == nil && tc.errorMatcher == nil:
				// correct; carry on
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if !cmp.Equal(newState, tc.expectedNewState) {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.expectedNewState, newState))
			}
		})
	}
}

func IsExecutionFailedError(err error) bool {
	return microerror.Cause(err) == executionFailedError
}

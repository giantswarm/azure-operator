package state

import (
	"context"
	"strconv"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
)

func Test_StateMachine(t *testing.T) {
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
				"open":   func(ctx context.Context, obj interface{}, currentState State) (State, error) { return "closed", nil },
				"closed": func(ctx context.Context, obj interface{}, currentState State) (State, error) { return "open", nil },
			},
			currentState:     "open",
			expectedNewState: "closed",
			errorMatcher:     nil,
		},
		{
			name: "case 1: unknown start state",
			machine: Machine{
				"open":   func(ctx context.Context, obj interface{}, currentState State) (State, error) { return "closed", nil },
				"closed": func(ctx context.Context, obj interface{}, currentState State) (State, error) { return "open", nil },
			},
			currentState:     "half-way",
			expectedNewState: "",
			errorMatcher:     IsExecutionFailedError,
		},
		{
			name: "case 2: unknown new state",
			machine: Machine{
				"open":   func(ctx context.Context, obj interface{}, currentState State) (State, error) { return "half-way", nil },
				"closed": func(ctx context.Context, obj interface{}, currentState State) (State, error) { return "open", nil },
			},
			currentState:     "open",
			expectedNewState: "",
			errorMatcher:     IsExecutionFailedError,
		},
		{
			name:             "case 3: execute state on empty state machine",
			machine:          Machine{},
			currentState:     "start",
			expectedNewState: "",
			errorMatcher:     IsExecutionFailedError,
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

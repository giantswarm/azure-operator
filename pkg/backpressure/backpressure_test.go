package backpressure

import (
	"strconv"
	"testing"
	"time"
)

func Test_Backpressure(t *testing.T) {
	testCases := []struct {
		name        string
		initial     *Backpressure
		modifyFunc  func(g *Backpressure)
		expectation func(g *Backpressure) bool
	}{
		{
			name:        "case 0: empty value can proceed",
			initial:     &Backpressure{},
			modifyFunc:  func(g *Backpressure) {},
			expectation: func(g *Backpressure) bool { return g.CanProceed() },
		},
		{
			name:        "case 1: CanProceed() returns false after NotBefore() set",
			initial:     &Backpressure{},
			modifyFunc:  func(g *Backpressure) { g.NotBefore(time.Now().Add(100 * time.Second)) },
			expectation: func(g *Backpressure) bool { return (g.CanProceed() == false) },
		},
		{
			name:        "case 2: CanProceed() returns true after value set in NotBefore() has expired",
			initial:     &Backpressure{},
			modifyFunc:  func(g *Backpressure) { g.NotBefore(time.Now().Add(-1 * time.Second)) },
			expectation: func(g *Backpressure) bool { return (g.CanProceed() == true) },
		},
		{
			name:        "case 3: RetryAfter() returns value set in NotBefore()",
			initial:     &Backpressure{},
			modifyFunc:  func(g *Backpressure) { g.NotBefore(time.Unix(100, 0)) },
			expectation: func(g *Backpressure) bool { return (g.RetryAfter() == time.Unix(100, 0)) },
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			g := tc.initial
			tc.modifyFunc(g)

			if !tc.expectation(g) {
				t.Fatalf("expectation failed; g: %#v", g)
			}
		})
	}
}

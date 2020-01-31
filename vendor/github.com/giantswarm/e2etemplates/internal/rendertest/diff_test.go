package rendertest

import (
	"reflect"
	"testing"
)

func Test_diff(t *testing.T) {
	testCases := []struct {
		name               string
		a                  string
		b                  string
		expectedLine       int
		expectedDifference string
		errorMatcher       func(err error) bool
	}{
		{
			name: "case 1",
			a: `x: 1
			    y: 2
			`,
			b: `x: 1
			    y: 2
			`,
			expectedLine:       0,
			expectedDifference: "",
		},
		{
			name: "case 2",
			a: `x: 1
			    y: 2`,
			b: `x: 4
			    y: 2`,
			expectedLine:       1,
			expectedDifference: `a: "x: 1" b: "x: 4"`,
		},
		{
			name: "case 3",
			a: `x: 1
			    y: 2`,
			b: `x: 1
			    y: 5`,
			expectedLine:       2,
			expectedDifference: `a: "\t\t\t    y: 2" b: "\t\t\t    y: 5"`,
		},
		{
			name: "case 4",
			a: `x: 1
			    y: 2`,
			b: `x: 1
			    y: 2
			    z`,
			expectedLine:       3,
			expectedDifference: `a: EOF b: "\t\t\t    z"`,
		},
		{
			name: "case 5",
			a: `x: 1
			    y: 2
			    z`,
			b: `x: 1
			    y: 2`,
			expectedLine:       3,
			expectedDifference: `a: "\t\t\t    z" b: EOF`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			line, difference := Diff(tc.a, tc.b)

			if !reflect.DeepEqual(line, tc.expectedLine) {
				t.Fatalf("line == %d, want %d", line, tc.expectedLine)
			}

			if !reflect.DeepEqual(difference, tc.expectedDifference) {
				t.Fatalf("difference == %s, want %s", difference, tc.expectedDifference)
			}
		})
	}
}

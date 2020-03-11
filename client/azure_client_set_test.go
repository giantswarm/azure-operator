package client

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_removeElementFromSlice(t *testing.T) {
	testCases := []struct {
		name       string
		xs         []int
		x          int
		expectedXs []int
	}{
		{

			name:       "case 0: simple case, remove number from slice",
			xs:         []int{0, 1, 2, 3},
			x:          2,
			expectedXs: []int{0, 1, 3},
		},
		{

			name:       "case 1: remove number from empty slice",
			xs:         []int{},
			x:          2,
			expectedXs: []int{},
		},
		{
			name:       "case 2: remove number from nil slice",
			xs:         nil,
			x:          2,
			expectedXs: nil,
		},
		{
			name:       "case 3: remove first number from slice",
			xs:         []int{0, 1, 2, 3, 4},
			x:          0,
			expectedXs: []int{1, 2, 3, 4},
		},
		{
			name:       "case 4: remove last number from slice",
			xs:         []int{0, 1, 2, 3, 4},
			x:          4,
			expectedXs: []int{0, 1, 2, 3},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			xs := removeElementFromSlice(tc.xs, tc.x)

			if !cmp.Equal(xs, tc.expectedXs) {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.expectedXs, xs))
			}
		})
	}
}

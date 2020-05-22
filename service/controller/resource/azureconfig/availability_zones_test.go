package azureconfig

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_sortAndUniqInts(t *testing.T) {
	testCases := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "case 0: nil slice",
			input:    nil,
			expected: []int{},
		},
		{
			name:     "case 1: empty slice",
			input:    []int{},
			expected: []int{},
		},
		{
			name:     "case 2: single element slice",
			input:    []int{1},
			expected: []int{1},
		},
		{
			name:     "case 3: multi element sorted uniq slice",
			input:    []int{1, 2, 3, 4},
			expected: []int{1, 2, 3, 4},
		},
		{
			name:     "case 4: multi element unsorted uniq slice",
			input:    []int{1, 4, 2, 3},
			expected: []int{1, 2, 3, 4},
		},
		{
			name:     "case 5: multi element sorted non-uniq slice",
			input:    []int{1, 2, 2, 2, 3, 3, 4},
			expected: []int{1, 2, 3, 4},
		},
		{
			name:     "case 4: multi element unsorted non-uniq slice",
			input:    []int{1, 4, 2, 1, 4, 1, 2, 4, 2, 1, 3},
			expected: []int{1, 2, 3, 4},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			xs := sortAndUniq(tc.input)

			if !cmp.Equal(xs, tc.expected) {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.expected, xs))
			}
		})
	}
}

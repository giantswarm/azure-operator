package azureclusterconfig

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_dnsZoneFromAPIEndpoint(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "case 0: empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "case 1: single element string",
			input:    "foobar",
			expected: "foobar",
		},
		{
			name:     "case 2: single element string with dot suffix",
			input:    "foobar.",
			expected: "foobar.",
		},
		{
			name:     "case 3: single element string with dot prefix",
			input:    ".foobar",
			expected: ".foobar",
		},
		{
			name:     "case 4: two element string",
			input:    "foobar.bazfoo",
			expected: "bazfoo",
		},
		{
			name:     "case 5: two element string with dot suffix",
			input:    "api.foobar.",
			expected: "foobar.",
		},
		{
			name:     "case 6: four element string",
			input:    "api.foobar.bazfoo.com",
			expected: "foobar.bazfoo.com",
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			zone := dnsZoneFromAPIEndpoint(tc.input)

			if !cmp.Equal(zone, tc.expected) {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.expected, zone))
			}
		})
	}
}

package employees

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

func Test_FromDraughtsmanString(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		output       SSHUserList
		errorMatcher func(err error) bool
	}{
		{
			name:         "case 0: One user",
			input:        "john:ssh-rsa abab111222333 john",
			output:       SSHUserList{"john": []string{"ssh-rsa abab111222333 john"}},
			errorMatcher: nil,
		},
		{
			name:         "case 1: Multiple users",
			input:        "john:ssh-rsa abab111222333 john,frank:ssh-rsa cdcd444555666 frank",
			output:       SSHUserList{"john": []string{"ssh-rsa abab111222333 john"}, "frank": []string{"ssh-rsa cdcd444555666 frank"}},
			errorMatcher: nil,
		},
		{
			name:         "case 2: Zero users",
			input:        "",
			output:       SSHUserList{},
			errorMatcher: nil,
		},
		{
			name:         "case 3: User without name",
			input:        ":ssh-rsa abab111222333 john",
			output:       SSHUserList{},
			errorMatcher: nil,
		},
		{
			name:         "case 4: User without key",
			input:        "john:",
			output:       SSHUserList{},
			errorMatcher: nil,
		},
		{
			name:         "case 5: Empty substring",
			input:        ",",
			output:       SSHUserList{},
			errorMatcher: nil,
		},
		{
			name:         "case 6: Invalid string",
			input:        "john;ssh-rsa abab111222333 john",
			output:       SSHUserList{},
			errorMatcher: IsParsingFailedError,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			list, err := FromDraughtsmanString(tc.input)
			switch {
			case err == nil && tc.errorMatcher == nil:
				// No error, compare the output.
				if !reflect.DeepEqual(list, tc.output) {
					t.Fatal(fmt.Sprintf("Wrong output: expected %s got %s", tc.output, list))
				}
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("expected %#v got %#v", nil, err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("expected %#v got %#v", "error", nil)
			case !tc.errorMatcher(err):
				t.Fatalf("expected %#v got %#v", true, false)
			}
		})
	}
}

//func Test_ToClusterKubernetesSSHUser(t *testing.T) {
//
//}

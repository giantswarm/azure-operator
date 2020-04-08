package httputil

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func notBefore(d int) time.Time { // nolint: unparam
	return time.Now().UTC().Add(time.Duration(d-1) * time.Second)
}

func notAfter(d int) time.Time { // nolint: unparam
	return time.Now().UTC().Add(time.Duration(d+10) * time.Second)
}

func Test_ParseRetryAfter(t *testing.T) {
	testCases := []struct {
		name              string
		response          *http.Response
		expectedNotBefore time.Time
		expectedNotAfter  time.Time
		errorMatcher      func(err error) bool
	}{
		{
			name: "case 0: handle single delay-seconds case",
			response: &http.Response{
				Header: map[string][]string{
					"Retry-After": []string{"600"},
				},
			},
			expectedNotBefore: notBefore(600),
			expectedNotAfter:  notAfter(600),
			errorMatcher:      nil,
		},
		{
			name: "case 1: handle two delay-seconds case",
			response: &http.Response{
				Header: map[string][]string{
					"Retry-After": []string{"600", "900"},
				},
			},
			expectedNotBefore: notBefore(600),
			expectedNotAfter:  notAfter(600),
			errorMatcher:      nil,
		},
		{
			name: "case 2: handle single http-date case",
			response: &http.Response{
				Header: map[string][]string{
					"Retry-After": []string{time.Now().UTC().Add(600 * time.Second).Format(http.TimeFormat)},
				},
			},
			expectedNotBefore: notBefore(600),
			expectedNotAfter:  notAfter(600),
			errorMatcher:      nil,
		},
		{
			name: "case 3: handle two http-dates case",
			response: &http.Response{
				Header: map[string][]string{
					"Retry-After": []string{time.Now().UTC().Add(600 * time.Second).Format(http.TimeFormat), time.Now().UTC().Add(900 * time.Second).Format(http.TimeFormat)},
				},
			},
			expectedNotBefore: notBefore(600),
			expectedNotAfter:  notAfter(600),
			errorMatcher:      nil,
		},
		{
			name: "case 3: handle mixed http-date and delay-seconds case",
			response: &http.Response{
				Header: map[string][]string{
					"Retry-After": []string{time.Now().UTC().Add(600 * time.Second).Format(http.TimeFormat), "900"},
				},
			},
			expectedNotBefore: notBefore(600),
			expectedNotAfter:  notAfter(600),
			errorMatcher:      nil,
		},
		{
			name: "case 4: handle mixed delay-seconds and http-date case",
			response: &http.Response{
				Header: map[string][]string{
					"Retry-After": []string{"600", time.Now().UTC().Add(900 * time.Second).Format(http.TimeFormat)},
				},
			},
			expectedNotBefore: notBefore(600),
			expectedNotAfter:  notAfter(600),
			errorMatcher:      nil,
		},
		{
			name: "case 5: handle missing Retry-After value",
			response: &http.Response{
				Header: map[string][]string{},
			},
			errorMatcher: IsParse,
		},
		{
			name:         "case 6: handle nil response",
			response:     nil,
			errorMatcher: IsParse,
		},
		{
			name: "case 6: handle garbage data",
			response: &http.Response{
				Header: map[string][]string{
					"Retry-After": []string{"foobar"},
				},
			},
			errorMatcher: IsParse,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			retryAfter, err := ParseRetryAfter(tc.response)

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

			if retryAfter.Before(tc.expectedNotBefore) {
				t.Fatalf("got %q, expected after %q", retryAfter, tc.expectedNotBefore)
			}

			if retryAfter.After(tc.expectedNotAfter) {
				t.Fatalf("got %q, expected before %q", retryAfter, tc.expectedNotAfter)
			}
		})
	}
}

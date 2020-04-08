// Package httputil provides useful HTTP utilities such as header parsers etc.
package httputil

import (
	"net/http"
	"strconv"
	"time"

	"github.com/giantswarm/microerror"
)

// ParseRetryAfter tries to parse `Retry-After` value from given HTTP response.
// In case there are multiple `Retry-After` header values present, first
// parseable one is returned. In case of nil response, missing or unparseable
// header value a `parseError` is returned.
func ParseRetryAfter(r *http.Response) (time.Time, error) {
	if r == nil {
		return time.Time{}, microerror.Maskf(parseError, "nil response")
	}

	// Iterate over possible values for `Retry-After` header. First parseable
	// wins.
	for _, v := range r.Header.Values("Retry-After") {
		// `Retry-After` value can be either <http-date> or <delay-seconds>.

		// Try to parse integer value of <delay-seconds> first.
		i64, err := strconv.ParseInt(v, 10, 32)
		if err == nil && i64 > 0 {
			return time.Now().UTC().Add(time.Duration(i64) * time.Second), nil
		}

		// Try <http-date> instead.
		t, err := http.ParseTime(v)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, microerror.Maskf(parseError, "parseable Retry-After missing")
}

package client

import (
	"net/http"
	"testing"

	"github.com/Azure/go-autorest/autorest"
)

func Test_ResponseWasNotFound(t *testing.T) {
	testCases := []struct {
		StatusCode     int
		ExpectedResult bool
	}{
		{
			StatusCode:     http.StatusOK,
			ExpectedResult: false,
		},
		{
			StatusCode:     http.StatusInternalServerError,
			ExpectedResult: false,
		},
		{
			StatusCode:     http.StatusNotFound,
			ExpectedResult: true,
		},
	}

	for i, tc := range testCases {
		resp := autorest.Response{
			Response: &http.Response{
				StatusCode: tc.StatusCode,
			},
		}

		if tc.ExpectedResult != ResponseWasNotFound(resp) {
			t.Fatalf("case %d expected %t got %t", i+1, tc.ExpectedResult, ResponseWasNotFound(resp))
		}
	}
}

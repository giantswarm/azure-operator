package client

import (
	"net/http"
	"testing"

	"github.com/Azure/go-autorest/autorest"
)

func Test_ResponseWasNotFound(t *testing.T) {
	testCases := []struct {
		Response       *http.Response
		ExpectedResult bool
	}{
		{
			Response: &http.Response{
				StatusCode: http.StatusNotFound,
			},
			ExpectedResult: true,
		},
		{
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
			ExpectedResult: false,
		},
		{
			Response: &http.Response{
				StatusCode: http.StatusInternalServerError,
			},
			ExpectedResult: false,
		},
		{
			Response:       nil,
			ExpectedResult: false,
		},
	}

	for i, tc := range testCases {
		resp := autorest.Response{
			Response: tc.Response,
		}

		if tc.ExpectedResult != ResponseWasNotFound(resp) {
			t.Fatalf("case %d expected %t got %t", i+1, tc.ExpectedResult, ResponseWasNotFound(resp))
		}
	}
}

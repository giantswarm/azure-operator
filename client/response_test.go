package client

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/Azure/go-autorest/autorest"
	"github.com/google/go-cmp/cmp"
)

func Test_ResponseWasNotFound(t *testing.T) {
	testCases := []struct {
		name           string
		response       *http.Response
		expectedResult bool
	}{
		{
			name: "case 0: Azure API returns 404",
			response: &http.Response{
				StatusCode: http.StatusNotFound,
			},
			expectedResult: true,
		},
		{
			name: "case 1: Azure API responds normally",
			response: &http.Response{
				StatusCode: http.StatusOK,
			},
			expectedResult: false,
		},
		{
			name: "case 2: Internal server error",
			response: &http.Response{
				StatusCode: http.StatusInternalServerError,
			},
			expectedResult: false,
		},
		{
			name:           "case 3: Reponse is nil (response does not exist)",
			response:       nil,
			expectedResult: false,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			resp := autorest.Response{
				Response: tc.response,
			}
			notFoundResult := ResponseWasNotFound(resp)
			if !cmp.Equal(notFoundResult, tc.expectedResult) {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.expectedResult, notFoundResult))

			}
		})
	}
}

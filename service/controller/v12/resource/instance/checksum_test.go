package instance

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_getDeploymentTemplateChecksum(t *testing.T) {
	testCases := []struct {
		Name                string
		TemplateLinkPresent bool
		StatusCode          int
		ResponseBody        string
		ExpectedChecksum    string
		ErrorMatcher        func(err error) bool
	}{
		{
			Name:                "Case 1: Successful checksum calculation",
			TemplateLinkPresent: true,
			StatusCode:          http.StatusOK,
			ResponseBody:        `{"fake": "json string"}`,
			ExpectedChecksum:    "0cfe91509c17c2a9f230cd117d90e837d948639c3a2d559cf1ef6ca6ae24ec79",
		},
		{
			Name:                "Case 2: Missing template link",
			TemplateLinkPresent: false,
			ExpectedChecksum:    "",
			ErrorMatcher:        IsNilTemplateLinkError,
		},
		{
			Name:                "Case 3: Error downloading template from external URI",
			TemplateLinkPresent: true,
			ExpectedChecksum:    "",
			StatusCode:          http.StatusInternalServerError,
			ResponseBody:        `{"error": "500 - Internal server error"}`,
			ErrorMatcher:        IsUnableToGetTemplateError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.StatusCode)
				w.Write([]byte(tc.ResponseBody))
			}))
			defer ts.Close()

			var templateLink *resources.TemplateLink
			if tc.TemplateLinkPresent {
				templateLink = &resources.TemplateLink{
					URI: to.StringPtr(ts.URL),
				}
			}

			properties := resources.DeploymentProperties{
				TemplateLink: templateLink,
			}
			deployment := resources.Deployment{
				Properties: &properties,
			}

			chk, err := getDeploymentTemplateChecksum(deployment)

			if chk != tc.ExpectedChecksum {
				t.Fatal(fmt.Sprintf("Wrong checksum: expected %s got %s", tc.ExpectedChecksum, chk))
			}

			switch {
			case err == nil && tc.ErrorMatcher == nil:
				// fall through
			case err != nil && tc.ErrorMatcher == nil:
				t.Fatalf("expected %#v got %#v", nil, err)
			case err == nil && tc.ErrorMatcher != nil:
				t.Fatalf("expected %#v got %#v", "error", nil)
			case !tc.ErrorMatcher(err):
				t.Fatalf("expected %#v got %#v", true, false)
			}
		})
	}
}

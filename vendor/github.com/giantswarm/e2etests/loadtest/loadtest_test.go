package loadtest

import (
	"context"
	"strconv"
	"testing"

	"github.com/giantswarm/apprclient/apprclienttest"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/micrologger/microloggertest"
)

const (
	apdexFails = `{
		"data": {
			"id": "kITVZoC0",
			"type": "test_runs",
			"attributes": {
				"basic_statistics": {
					"apdex_75": 0.50
				}
			}
		}
	}`
	apdexPasses = `{
		"data": {
			"id": "kITVZoC0",
			"type": "test_runs",
			"attributes": {
				"basic_statistics": {
					"apdex_75": 0.95
				}
			}
		}
	}`
)

func Test_LoadTest_CheckLoadTestResults(t *testing.T) {
	var err error

	testCases := []struct {
		name         string
		results      string
		errorMatcher func(error) bool
	}{
		{
			name:         "case 0: apdex passes",
			results:      apdexPasses,
			errorMatcher: nil,
		},
		{
			name:         "case 1: apdex fails",
			results:      apdexFails,
			errorMatcher: IsFailedLoadTest,
		},
	}

	ctx := context.Background()

	c := Config{
		ApprClient:   apprclienttest.New(apprclienttest.Config{}),
		CPClients:    &k8sclient.Clients{},
		CPHelmClient: &helmclient.Client{},
		Logger:       microloggertest.New(),
		TCClients:    &k8sclient.Clients{},
		TCHelmClient: &helmclient.Client{},

		ClusterID:            "eggs2",
		CommonDomain:         "aws.gigantic.io",
		StormForgerAuthToken: "secret",
	}

	l, err := New(c)
	if err != nil {
		t.Fatalf("error == %#v, want nil", err)
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			err = l.checkLoadTestResults(ctx, []byte(tc.results))

			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}
		})
	}
}

package e2etemplates

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newAWSHostPeerStackConfigFromFilled(modifyFunc func(*AWSHostPeerStackConfig)) AWSHostPeerStackConfig {
	c := AWSHostPeerStackConfig{
		Stack: AWSHostPeerStackConfigStack{
			Name: "test-stack-name",
		},
		RouteTable0: AWSHostPeerStackConfigRouteTable0{
			Name: "test-route-table-0-name",
		},
		RouteTable1: AWSHostPeerStackConfigRouteTable1{
			Name: "test-route-table-1-name",
		},
	}

	modifyFunc(&c)

	return c
}

func Test_NewAWSHostPeerStack(t *testing.T) {
	testCases := []struct {
		name           string
		config         AWSHostPeerStackConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         AWSHostPeerStackConfig{},
			expectedValues: "",
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newAWSHostPeerStackConfigFromFilled(func(v *AWSHostPeerStackConfig) {}),
			expectedValues: `
AWSTemplateFormatVersion: 2010-09-09
Description: Control Plane Peer Stack with VPC peering and route tables for testing purposes
Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.11.0.0/16
      Tags:
      - Key: Name
        Value: test-stack-name
      - Key: giantswarm.io/installation
        Value: cp-peer-test-stack-name
  PeerRouteTable0:
    Type: AWS::EC2::RouteTable
    Properties:
      VpcId: !Ref VPC
      Tags:
      - Key: Name
        Value: test-route-table-0-name
  PeerRouteTable1:
    Type: AWS::EC2::RouteTable
    Properties:
      VpcId: !Ref VPC
      Tags:
      - Key: Name
        Value: test-route-table-1-name
Outputs:
  VPCID:
    Description: Accepter VPC ID
    Value: !Ref VPC
`,
			errorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewAWSHostPeerStack(tc.config)

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

			if tc.errorMatcher != nil {
				return
			}

			line, difference := rendertest.Diff(values, tc.expectedValues)
			if line > 0 {
				t.Fatalf("line == %d, want 0, diff: %s", line, difference)
			}
		})
	}
}

func Test_NewAWSHostPeerStack_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       AWSHostPeerStackConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .Stack.Name",
			config: newAWSHostPeerStackConfigFromFilled(func(v *AWSHostPeerStackConfig) {
				v.Stack.Name = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: invalid .RouteTable0.Name",
			config: newAWSHostPeerStackConfigFromFilled(func(v *AWSHostPeerStackConfig) {
				v.RouteTable0.Name = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: invalid .RouteTable1.Name",
			config: newAWSHostPeerStackConfigFromFilled(func(v *AWSHostPeerStackConfig) {
				v.RouteTable1.Name = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAWSHostPeerStack(tc.config)

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

			if tc.errorMatcher != nil {
				return
			}
		})
	}
}

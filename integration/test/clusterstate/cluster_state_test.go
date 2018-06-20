// +build k8srequired

package clusterstate

import (
	"context"
	"testing"
)

func Test_Cluster_State(t *testing.T) {
	err := s.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

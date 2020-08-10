// +build k8srequired

package clusterstate

import (
	"context"
	"testing"
)

func Test_ClusterState(t *testing.T) {
	err := clusterStateTest.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

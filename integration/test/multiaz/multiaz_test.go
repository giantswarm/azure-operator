// +build k8srequired

package multiaz

import (
	"context"
	"testing"
)

func Test_AZ(t *testing.T) {
	err := multiaz.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

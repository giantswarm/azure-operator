package ipam

import (
	"context"
	"net"
)

type TestChecker struct {
	subnet *net.IPNet
}

func NewTestChecker(subnet *net.IPNet) *TestChecker {
	a := &TestChecker{
		subnet: subnet,
	}

	return a
}

func (c *TestChecker) Check(ctx context.Context, namespace string, name string) (*net.IPNet, error) {
	return c.subnet, nil
}

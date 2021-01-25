package ipam

import (
	"context"
	"net"
)

type nopReleaser struct{}

func NewNOPReleaser() Releaser {
	return &nopReleaser{}
}

func (r *nopReleaser) Release(ctx context.Context, subnet net.IPNet, namespace, name string) error {
	return nil
}

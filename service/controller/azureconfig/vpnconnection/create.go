package vpnconnection

import (
	"context"
)

// ApplyCreateChange is noop. Creation goes through ApplyUpdateChange.
func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	return nil
}

package senddecorator

import (
	"github.com/Azure/go-autorest/autorest"
)

// WrapClient accepts variable number of SendDecorators that wrap the given
// autorest Client.
//
// Existing SendDecorators are preserved, but moved to end of slice.
func WrapClient(c *autorest.Client, decorators ...autorest.SendDecorator) {
	// NOTE: Order matters here since these decorators are executed in
	// order. See: https://godoc.org/github.com/Azure/go-autorest/autorest#Client
	c.SendDecorators = append(decorators, c.SendDecorators...)
}

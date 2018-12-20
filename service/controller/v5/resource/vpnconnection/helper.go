package vpnconnection

import (
	"math/rand"

	"github.com/giantswarm/microerror"
)

// toVPNGatewayConnections convert v to network.VirtualNetworkGatewayConnection.
// If v is nil and empty network.VirtualNetworkGatewayConnection is returned.
func toVPNGatewayConnections(v interface{}) (connections, error) {
	if v == nil {
		return connections{}, nil
	}

	vpnGatewayConnections, ok := v.(connections)
	if !ok {
		return connections{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", connections{}, v)
	}

	return vpnGatewayConnections, nil
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}

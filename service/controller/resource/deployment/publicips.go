package deployment

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
)

func (r Resource) getCPPublicIPAddresses(ctx context.Context) ([]string, error) {
	names := []string{
		fmt.Sprintf("%s_api_ip", r.installationName),
		fmt.Sprintf("%s_ingress_ip", r.installationName),
	}

	var ret []string

	for _, name := range names {
		ipAddr, err := r.cpPublicIpAddressesClient.Get(ctx, r.installationName, name, "")
		if err != nil {
			return []string{}, microerror.Mask(err)
		}
		ret = append(ret, *ipAddr.IPAddress)
	}

	return ret, nil
}
